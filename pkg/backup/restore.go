package backup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/storage"
)

const restoreInsertBatchSize = 100

// RestoreOptions represents the options of restoring a backup
type RestoreOptions struct {
	// Passphrase is the passphrase used to decrypt the backup file
	Passphrase string

	// DryRun only validates the backup and the target, without any modification
	DryRun bool

	// Force allows restoring into a non-empty target (existing rows of all the tables are DELETED)
	Force bool

	// ConfirmOverwrite is the explicit confirmation of overwriting existing data, required together with Force
	ConfirmOverwrite bool

	// ConfigOutputPath is the file path where the configuration file in backup is written to (optional)
	ConfigOutputPath string
}

// RestoreTableResult represents the restore result of a table
type RestoreTableResult struct {
	Name         string
	Description  string
	BackupRows   int64
	RestoredRows int64
	ExistingRows int64
	DigestMatch  bool
	DigestNote   string
}

// RestoreReport represents the result of restoring a backup
type RestoreReport struct {
	Verify                 *VerifyReport
	DryRun                 bool
	TargetEmpty            bool
	NonEmptyTables         []string
	ConflictingObjects     int
	Tables                 []*RestoreTableResult
	StorageObjects         int
	StorageObjectsVerified int
	ConfigWrittenTo        string
	UsersTotal             int
	UsersLoaded            int
	Highlights             []string
	Problems               []string
	Warnings               []string
}

// OK returns whether the restore (or dry run) succeeded without problems
func (r *RestoreReport) OK() bool {
	return len(r.Problems) == 0 && (r.Verify == nil || r.Verify.OK())
}

func (r *RestoreReport) addProblem(format string, args ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

func (r *RestoreReport) addWarning(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// RestoreBackup validates the backup and the target, and restores the database tables and object storage objects from the backup,
// then verifies the restored data
func RestoreBackup(c core.Context, config *settings.Config, filePath string, options *RestoreOptions) (*RestoreReport, error) {
	report := &RestoreReport{
		DryRun: options.DryRun,
	}

	// 1. validate the backup (checksums, manifest, json, compatibility) before any modification
	verifyReport, err := VerifyBackup(c, filePath, options.Passphrase)
	report.Verify = verifyReport

	if err != nil {
		return report, err
	}

	manifest := verifyReport.Manifest

	if manifest.ConfigIncluded && options.ConfigOutputPath != "" {
		if _, err := os.Stat(options.ConfigOutputPath); err == nil && !options.Force {
			return report, fmt.Errorf("configuration output file \"%s\" already exists", options.ConfigOutputPath)
		}
	} else if !manifest.ConfigIncluded && options.ConfigOutputPath != "" {
		report.addWarning("backup does not contain configuration file, nothing will be written to \"%s\"", options.ConfigOutputPath)
	}

	// 2. check whether the target is empty
	schemas, err := loadTableSchemas()

	if err != nil {
		return report, err
	}

	schemasByName := make(map[string]*tableSchema, len(schemas))
	report.TargetEmpty = true

	for _, schema := range schemas {
		schemasByName[schema.name] = schema
		rowCount, err := countTableRows(schema)

		if err != nil {
			return report, fmt.Errorf("cannot check table \"%s\" of target database: %w", schema.name, err)
		}

		if rowCount > 0 {
			report.TargetEmpty = false
			report.NonEmptyTables = append(report.NonEmptyTables, fmt.Sprintf("%s (%d rows)", schema.name, rowCount))
		}

		backupRows := int64(0)

		if table := manifest.GetTable(schema.name); table != nil {
			backupRows = table.Rows
		}

		report.Tables = append(report.Tables, &RestoreTableResult{
			Name:         schema.name,
			Description:  schema.registration.Description,
			BackupRows:   backupRows,
			ExistingRows: max(rowCount, 0),
		})
	}

	objectStorages, err := storage.NewAllObjectStorages(config)

	if err != nil {
		return report, fmt.Errorf("cannot initialize object storage: %w", err)
	}

	existingObjects := make(map[string]bool)

	for pathPrefix, objectStorage := range objectStorages {
		objectPaths, err := objectStorage.List(c)

		if err != nil {
			return report, fmt.Errorf("cannot list objects in \"%s\" object storage: %w", pathPrefix, err)
		}

		for _, objectPath := range objectPaths {
			existingObjects[getStorageEntryName(pathPrefix, objectPath)] = true
		}
	}

	for _, entry := range manifest.GetStorageEntries() {
		if existingObjects[entry.Name] {
			report.ConflictingObjects++
		}
	}

	if report.ConflictingObjects > 0 {
		report.TargetEmpty = false
	}

	report.StorageObjects = len(manifest.GetStorageEntries())

	if !report.TargetEmpty && !(options.Force && options.ConfirmOverwrite) {
		if options.DryRun {
			report.addWarning("target is not empty, a real restore requires --force and --yes-overwrite-existing-data")
			return report, nil
		}

		return report, ErrRestoreTargetNotEmpty
	}

	if options.DryRun {
		return report, nil
	}

	// 3. create / update the database structure
	for _, schema := range schemas {
		if err := schema.registration.Store().SyncStructs(schema.registration.NewBean()); err != nil {
			return report, fmt.Errorf("cannot update structure of table \"%s\": %w", schema.name, err)
		}
	}

	log.Infof(c, "[backup.RestoreBackup] database structure has been updated")

	// existing rows of tables which are not in the backup must also be cleared when overwriting
	if !report.TargetEmpty {
		for _, schema := range schemas {
			if manifest.GetTable(schema.name) == nil {
				if err := clearTable(c, schema); err != nil {
					return report, err
				}
			}
		}
	}

	// 4. restore database rows and storage objects
	restoredRows := make(map[string]int64)

	_, err = readArchive(filePath, options.Passphrase, func(name string, size int64, reader io.Reader) error {
		if strings.HasPrefix(name, databaseEntryPrefix) {
			tableName := strings.TrimSuffix(strings.TrimPrefix(name, databaseEntryPrefix), databaseEntrySuffix)
			schema := schemasByName[tableName]

			if schema == nil {
				return fmt.Errorf("%w: unknown table \"%s\"", ErrBackupIncompatible, tableName)
			}

			rowCount, err := restoreTable(c, schema, reader, !report.TargetEmpty)

			if err != nil {
				return fmt.Errorf("cannot restore table \"%s\" (tables restored before it have been committed, please clean the target database before retrying): %w", tableName, err)
			}

			restoredRows[tableName] = rowCount
			log.Infof(c, "[backup.RestoreBackup] table \"%s\" has been restored, %d rows", tableName, rowCount)
		} else if pathPrefix, objectPath, ok := parseStorageEntryName(name); ok {
			objectStorage := objectStorages[pathPrefix]

			if objectStorage == nil {
				return fmt.Errorf("%w: unknown object storage path prefix \"%s\"", ErrBackupIncompatible, pathPrefix)
			}

			data, err := io.ReadAll(reader)

			if err != nil {
				return err
			}

			expectedEntry := manifest.GetEntry(name)
			actualChecksum := sha256.Sum256(data)

			if expectedEntry == nil || expectedEntry.SHA256 != hex.EncodeToString(actualChecksum[:]) {
				return fmt.Errorf("%w: checksum of entry \"%s\" mismatches", ErrBackupCorrupted, name)
			}

			if err := objectStorage.Save(c, objectPath, newBytesObject(data)); err != nil {
				return fmt.Errorf("cannot save object \"%s/%s\": %w", pathPrefix, objectPath, err)
			}
		} else if name == configEntryName && options.ConfigOutputPath != "" {
			data, err := io.ReadAll(reader)

			if err != nil {
				return err
			}

			if err := writeFileAtomically(options.ConfigOutputPath, data, 0600); err != nil {
				return fmt.Errorf("cannot write configuration file: %w", err)
			}

			report.ConfigWrittenTo = options.ConfigOutputPath
		}

		return nil
	})

	if err != nil {
		return report, err
	}

	log.Infof(c, "[backup.RestoreBackup] %d storage objects have been restored", report.StorageObjects)

	// 5. verify the restored data
	verifyRestoredData(c, report, manifest, schemas, objectStorages, restoredRows)

	if !report.OK() {
		return report, fmt.Errorf("%w: %s", ErrRestoreVerificationFailed, report.Problems[0])
	}

	return report, nil
}

func clearTable(c core.Context, schema *tableSchema) error {
	return schema.database.DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Exec("DELETE FROM " + schema.engine().Quote(schema.fullName))
		return err
	})
}

// restoreTable inserts all the rows of a table in batches inside one transaction
func restoreTable(c core.Context, schema *tableSchema, reader io.Reader, deleteExistingRows bool) (int64, error) {
	rowCount := int64(0)

	err := schema.database.DoTransaction(c, func(sess datastore.DatabaseSession) error {
		if deleteExistingRows {
			if _, err := sess.Exec("DELETE FROM " + schema.engine().Quote(schema.fullName)); err != nil {
				return err
			}
		}

		batch := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(schema.beanType)), 0, restoreInsertBatchSize)

		flush := func() error {
			if batch.Len() < 1 {
				return nil
			}

			if _, err := sess.Insert(batch.Interface()); err != nil {
				return err
			}

			batch = batch.Slice(0, 0)
			return nil
		}

		_, err := scanJsonLines(reader, func(lineNumber int64, line []byte) error {
			values, err := decodeRow(schema, line)

			if err != nil {
				return fmt.Errorf("line %d: %w", lineNumber, err)
			}

			bean, err := newBeanFromValues(schema, values)

			if err != nil {
				return fmt.Errorf("line %d: %w", lineNumber, err)
			}

			batch = reflect.Append(batch, reflect.ValueOf(bean))
			rowCount++

			if batch.Len() >= restoreInsertBatchSize {
				return flush()
			}

			return nil
		})

		if err != nil {
			return err
		}

		return flush()
	})

	return rowCount, err
}

func verifyRestoredData(c core.Context, report *RestoreReport, manifest *Manifest, schemas []*tableSchema, objectStorages map[string]storage.ObjectStorage, restoredRows map[string]int64) {
	resultsByName := make(map[string]*RestoreTableResult, len(report.Tables))

	for _, result := range report.Tables {
		resultsByName[result.Name] = result
	}

	// table row counts and content digests
	for _, schema := range schemas {
		result := resultsByName[schema.name]
		backupTable := manifest.GetTable(schema.name)
		actualRows, err := countTableRows(schema)

		if err != nil {
			report.addProblem("cannot count rows of table \"%s\": %s", schema.name, err.Error())
			continue
		}

		result.RestoredRows = actualRows

		if actualRows != result.BackupRows {
			report.addProblem("row count of table \"%s\" mismatches (backup %d, restored %d)", schema.name, result.BackupRows, actualRows)
		}

		if backupTable == nil {
			result.DigestNote = "table not in backup"
			continue
		}

		if restored, exists := restoredRows[schema.name]; exists && restored != backupTable.Rows {
			report.addProblem("inserted row count of table \"%s\" mismatches (backup %d, inserted %d)", schema.name, backupTable.Rows, restored)
		}

		if strings.Join(backupTable.Columns, ",") != strings.Join(schema.columnNames(), ",") {
			result.DigestNote = "columns changed since backup, digest not compared"
			continue
		}

		hash := sha256.New()

		if _, err := dumpTable(c, schema.engine().DB().DB, schema, hash, nil); err != nil {
			report.addProblem("cannot read restored table \"%s\": %s", schema.name, err.Error())
			continue
		}

		entry := manifest.GetEntry(backupTable.EntryName)

		if entry != nil && entry.SHA256 == hex.EncodeToString(hash.Sum(nil)) {
			result.DigestMatch = true
		} else if manifest.DatabaseType != schema.accessor.DatabaseType() {
			result.DigestNote = "content digest differs (cross-database restore, values may be represented differently)"
			report.addWarning("content digest of table \"%s\" differs from backup (backup database %s, target database %s)", schema.name, manifest.DatabaseType, schema.accessor.DatabaseType())
		} else {
			result.DigestNote = "content digest differs"
			report.addProblem("content digest of table \"%s\" differs from backup", schema.name)
		}
	}

	// users can be loaded
	users := make([]*models.User, 0)

	if err := datastore.Container.UserStore.Query(c, 0).Find(&users); err != nil {
		report.addProblem("cannot load users: %s", err.Error())
	} else {
		report.UsersTotal = len(users)

		for _, user := range users {
			loadedUser, err := services.Users.GetUserById(c, user.Uid)

			if err != nil || loadedUser == nil || loadedUser.Username != user.Username {
				errorMessage := "user not found"

				if err != nil {
					errorMessage = err.Error()
				}

				report.addProblem("cannot load user \"%s\" (uid %d): %s", user.Username, user.Uid, errorMessage)
				continue
			}

			report.UsersLoaded++
		}
	}

	// highlights of important data
	for _, highlight := range []struct {
		bean  any
		label string
	}{
		{new(models.User), "users"},
		{new(models.Account), "accounts (including balances)"},
		{new(models.Transaction), "transactions"},
		{new(models.TransactionCategory), "transaction categories"},
		{new(models.TransactionTag), "transaction tags"},
		{new(models.TransactionTemplate), "transaction templates (including scheduled transactions)"},
		{new(models.UserApplicationCloudSetting), "user application settings"},
		{new(models.TransactionPictureInfo), "transaction pictures / attachments"},
		{new(models.UserCustomIcon), "user custom icons"},
	} {
		tableName := schemas[0].engine().TableName(highlight.bean)
		result := resultsByName[tableName]

		if result == nil {
			continue
		}

		status := "OK"

		if result.RestoredRows != result.BackupRows {
			status = "MISMATCH"
		} else if result.DigestMatch {
			status = "OK, content identical"
		}

		report.Highlights = append(report.Highlights, fmt.Sprintf("%s: %d restored / %d in backup (%s)", highlight.label, result.RestoredRows, result.BackupRows, status))
	}

	scheduledTemplates, err := datastore.Container.UserDataStore.Query(c, 0).Where("template_type=? AND deleted=?", models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE, false).Count(&models.TransactionTemplate{})

	if err == nil {
		report.Highlights = append(report.Highlights, fmt.Sprintf("scheduled transaction templates: %d", scheduledTemplates))
	}

	// storage objects can be read and checksums match
	for _, entry := range manifest.GetStorageEntries() {
		pathPrefix, objectPath, _ := parseStorageEntryName(entry.Name)
		object, exists, err := objectStorages[pathPrefix].Read(c, objectPath)

		if err != nil || !exists || object == nil {
			report.addProblem("cannot read restored object \"%s/%s\"", pathPrefix, objectPath)
			continue
		}

		hash := sha256.New()
		_, err = io.Copy(hash, object)
		_ = object.Close()

		if err != nil || hex.EncodeToString(hash.Sum(nil)) != entry.SHA256 {
			report.addProblem("checksum of restored object \"%s/%s\" mismatches", pathPrefix, objectPath)
			continue
		}

		report.StorageObjectsVerified++
	}

	report.Highlights = append(report.Highlights, fmt.Sprintf("storage objects: %d verified / %d in backup", report.StorageObjectsVerified, report.StorageObjects))

	for _, missingObject := range manifest.MissingStorageObjects {
		report.addWarning("object \"%s\" was already missing when the backup was created", missingObject)
	}
}

// bytesObject represents an in-memory object which can be saved into object storage
type bytesObject struct {
	*bytes.Reader
}

func (b *bytesObject) Close() error {
	return nil
}

func newBytesObject(data []byte) storage.ObjectInStorage {
	return &bytesObject{
		Reader: bytes.NewReader(data),
	}
}
