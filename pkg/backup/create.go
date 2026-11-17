package backup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/storage"
)

// CreateOptions represents the options of creating a backup
type CreateOptions struct {
	// OutputDir is the directory where the backup file is written to
	OutputDir string

	// IncludeConfig indicates whether to include the configuration file into the backup
	IncludeConfig bool

	// Encrypt indicates whether to encrypt the backup file
	Encrypt bool

	// Passphrase is the passphrase used to encrypt the backup file
	Passphrase string

	// ApplyRetention indicates whether to apply the retention policy after a successful backup
	ApplyRetention bool

	// RetentionCount is the maximum count of backups to keep (0 means unlimited)
	RetentionCount uint32

	// RetentionDays is the maximum age (in days) of backups to keep (0 means unlimited)
	RetentionDays uint32
}

// CreateResult represents the result of creating a backup
type CreateResult struct {
	FilePath        string
	Size            int64
	Manifest        *Manifest
	DeletedBackups  []string
	RetentionErrors []string
}

// NewCreateOptionsFromConfig returns the backup options according to the [backup] section of the config
func NewCreateOptionsFromConfig(config *settings.Config) *CreateOptions {
	return &CreateOptions{
		OutputDir:      config.BackupPath,
		IncludeConfig:  config.BackupIncludeConfig,
		Encrypt:        config.BackupEncrypt,
		Passphrase:     config.BackupPassphrase,
		ApplyRetention: true,
		RetentionCount: config.BackupRetentionCount,
		RetentionDays:  config.BackupRetentionDays,
	}
}

// CreateBackup creates a full-instance backup (database, object storage and optionally configuration file),
// prevents overlapping runs, updates the status file and applies the retention policy after a successful backup
func CreateBackup(c core.Context, config *settings.Config, options *CreateOptions) (*CreateResult, error) {
	if options.OutputDir == "" {
		return nil, fmt.Errorf("backup output directory is not set")
	}

	if options.Encrypt && options.Passphrase == "" {
		return nil, ErrBackupPassphraseRequired
	}

	outputDir, err := filepath.Abs(options.OutputDir)

	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return nil, err
	}

	lock, err := acquireBackupLock(outputDir)

	if err != nil {
		return nil, err
	}

	defer lock.release()

	startTime := time.Now()
	_ = updateBackupStatus(outputDir, func(status *BackupStatus) {
		status.LastRunStartedAt = timePointer(startTime)
		status.Running = true
	})

	result, err := createBackupFile(c, config, options, outputDir, startTime)

	if err != nil {
		log.Errorf(c, "[backup.CreateBackup] failed to create backup, because %s", err.Error())

		_ = updateBackupStatus(outputDir, func(status *BackupStatus) {
			status.Running = false
			status.LastRunFinishedAt = timePointer(time.Now())
			status.LastRunSuccess = false
			status.LastFailureAt = timePointer(time.Now())
			status.LastError = err.Error()
		})

		return nil, err
	}

	if options.ApplyRetention {
		deleted, retentionErrors := applyRetention(outputDir, result.FilePath, options.RetentionCount, options.RetentionDays, time.Now())
		result.DeletedBackups = deleted
		result.RetentionErrors = retentionErrors

		for _, deletedFile := range deleted {
			log.Infof(c, "[backup.CreateBackup] old backup \"%s\" has been removed by retention policy", deletedFile)
		}

		for _, retentionError := range retentionErrors {
			log.Warnf(c, "[backup.CreateBackup] failed to apply retention policy, because %s", retentionError)
		}
	}

	_ = updateBackupStatus(outputDir, func(status *BackupStatus) {
		status.Running = false
		status.LastRunFinishedAt = timePointer(time.Now())
		status.LastRunSuccess = true
		status.LastSuccessAt = timePointer(time.Now())
		status.LastSuccessFile = filepath.Base(result.FilePath)
		status.LastSuccessSize = result.Size
		status.LastSuccessWarnings = len(result.Manifest.Warnings) + len(result.Manifest.MissingStorageObjects)
		status.LastRetentionDeleted = result.DeletedBackups
	})

	log.Infof(c, "[backup.CreateBackup] backup \"%s\" (%d bytes, %d tables, %d storage objects) has been created successfully, cost %dms",
		result.FilePath, result.Size, len(result.Manifest.Tables), result.Manifest.StorageObjectCount, time.Since(startTime).Milliseconds())

	return result, nil
}

func createBackupFile(c core.Context, config *settings.Config, options *CreateOptions, outputDir string, createdAt time.Time) (*CreateResult, error) {
	createdAt = createdAt.UTC()
	finalFilePath := filepath.Join(outputDir, GetBackupFileName(createdAt, options.Encrypt))
	partialFilePath := finalFilePath + partialBackupFileSuffix

	tempDir, err := os.MkdirTemp(outputDir, ".tmp-backup-")

	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tempDir)

	manifest := &Manifest{
		FormatVersion:         ManifestFormatVersion,
		ApplicationName:       "ezrbookkeeping",
		AppVersion:            core.Version,
		AppCommitHash:         core.CommitHash,
		CreatedAt:             createdAt,
		DatabaseType:          config.DatabaseConfig.DatabaseType,
		StorageType:           config.StorageType,
		MissingStorageObjects: make([]string, 0),
		Warnings:              make([]string, 0),
	}

	// 1. dump all the database tables into temporary files within one snapshot transaction per database
	tableFiles, referencedObjects, err := dumpAllTables(c, manifest, tempDir)

	if err != nil {
		return nil, err
	}

	writer, err := newArchiveWriter(partialFilePath, options.Passphrase, options.Encrypt, createdAt)

	if err != nil {
		return nil, err
	}

	success := false

	defer func() {
		if !success {
			writer.abort()
			_ = os.Remove(partialFilePath)
		}
	}()

	for _, table := range manifest.Tables {
		if _, err := writer.addFile(table.EntryName, tableFiles[table.Name]); err != nil {
			return nil, err
		}

		_ = os.Remove(tableFiles[table.Name])
	}

	// 2. copy all the objects in object storage
	objectStorages, err := storage.NewAllObjectStorages(config)

	if err != nil {
		return nil, fmt.Errorf("cannot initialize object storage: %w", err)
	}

	storedObjects := make(map[string]bool)

	for _, pathPrefix := range storage.GetAllObjectStoragePathPrefixes() {
		objectStorage := objectStorages[pathPrefix]
		objectPaths, err := objectStorage.List(c)

		if err != nil {
			return nil, fmt.Errorf("cannot list objects in \"%s\" object storage: %w", pathPrefix, err)
		}

		for _, objectPath := range objectPaths {
			entryName := getStorageEntryName(pathPrefix, objectPath)

			if err := validateEntryName(entryName); err != nil {
				manifest.Warnings = append(manifest.Warnings, fmt.Sprintf("skipped object \"%s/%s\" with unsupported path", pathPrefix, objectPath))
				continue
			}

			object, exists, err := objectStorage.Read(c, objectPath)

			if err != nil {
				return nil, fmt.Errorf("cannot read object \"%s/%s\": %w", pathPrefix, objectPath, err)
			}

			if !exists || object == nil {
				// the object was deleted after listing
				continue
			}

			data, err := io.ReadAll(object)
			_ = object.Close()

			if err != nil {
				return nil, fmt.Errorf("cannot read object \"%s/%s\": %w", pathPrefix, objectPath, err)
			}

			if _, err := writer.addBytes(entryName, data); err != nil {
				return nil, err
			}

			storedObjects[pathPrefix+"/"+objectPath] = true
			manifest.StorageObjectCount++
		}
	}

	for _, objectKey := range referencedObjects {
		if !storedObjects[objectKey] {
			manifest.MissingStorageObjects = append(manifest.MissingStorageObjects, objectKey)
			log.Warnf(c, "[backup.createBackupFile] object \"%s\" is referenced by database but does not exist in object storage", objectKey)
		}
	}

	// 3. copy the configuration file
	if options.IncludeConfig {
		if config.ConfigFilePath == "" {
			manifest.Warnings = append(manifest.Warnings, "configuration file path is unknown, configuration file is not included")
		} else {
			configData, err := os.ReadFile(config.ConfigFilePath)

			if err != nil {
				return nil, fmt.Errorf("cannot read configuration file: %w", err)
			}

			if _, err := writer.addBytes(configEntryName, configData); err != nil {
				return nil, err
			}

			manifest.ConfigIncluded = true
		}
	}

	if err := writer.finish(manifest); err != nil {
		return nil, err
	}

	// 4. verify the written archive before marking it as completed
	if _, err := verifyArchive(partialFilePath, options.Passphrase, false); err != nil {
		return nil, fmt.Errorf("verification of the written backup failed: %w", err)
	}

	if err := os.Rename(partialFilePath, finalFilePath); err != nil {
		return nil, err
	}

	syncDirectory(outputDir)
	success = true

	fileInfo, err := os.Stat(finalFilePath)

	if err != nil {
		return nil, err
	}

	return &CreateResult{
		FilePath: finalFilePath,
		Size:     fileInfo.Size(),
		Manifest: manifest,
	}, nil
}

// dumpAllTables dumps all the registered tables into temporary files, and returns the file paths and the object keys referenced by database rows
func dumpAllTables(c core.Context, manifest *Manifest, tempDir string) (map[string]string, []string, error) {
	schemas, err := loadTableSchemas()

	if err != nil {
		return nil, nil, err
	}

	tableFiles := make(map[string]string)
	referencedObjects := make([]string, 0)

	// group tables by database instance, so that all the tables of one database are read in one snapshot transaction
	databaseOrder := make([]any, 0)
	tablesByDatabase := make(map[any][]*tableSchema)

	for _, schema := range schemas {
		if _, exists := tablesByDatabase[schema.database]; !exists {
			databaseOrder = append(databaseOrder, schema.database)
		}

		tablesByDatabase[schema.database] = append(tablesByDatabase[schema.database], schema)
	}

	for _, database := range databaseOrder {
		databaseTables := tablesByDatabase[database]
		accessor := databaseTables[0].accessor
		tx, err := beginSnapshotTransaction(c, accessor)

		if err != nil {
			return nil, nil, fmt.Errorf("cannot start snapshot transaction: %w", err)
		}

		for _, schema := range databaseTables {
			filePath := filepath.Join(tempDir, schema.name+databaseEntrySuffix)
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

			if err != nil {
				_ = tx.Rollback()
				return nil, nil, err
			}

			bufferedWriter := bufio.NewWriterSize(file, 256*1024)
			collector := newReferencedObjectCollector(schema)
			rowCount, err := dumpTable(c, tx, schema, bufferedWriter, collector.handleRow)

			if err == nil {
				err = bufferedWriter.Flush()
			}

			closeErr := file.Close()

			if err == nil {
				err = closeErr
			}

			if err != nil {
				_ = tx.Rollback()
				return nil, nil, fmt.Errorf("cannot dump table \"%s\" (if the database structure is outdated, run \"ezbookkeeping database update\" first): %w", schema.name, err)
			}

			referencedObjects = append(referencedObjects, collector.objectKeys...)
			tableFiles[schema.name] = filePath
			manifest.Tables = append(manifest.Tables, &ManifestTable{
				Name:        schema.name,
				Description: schema.registration.Description,
				Rows:        rowCount,
				Columns:     schema.columnNames(),
				EntryName:   getDatabaseEntryName(schema.name),
			})

			log.Debugf(c, "[backup.dumpAllTables] table \"%s\" has been dumped, %d rows", schema.name, rowCount)
		}

		if err := tx.Rollback(); err != nil {
			log.Warnf(c, "[backup.dumpAllTables] cannot end snapshot transaction, because %s", err.Error())
		}
	}

	sort.Strings(referencedObjects)

	return tableFiles, referencedObjects, nil
}

// referencedObjectCollector collects the object storage keys which are referenced by database rows
type referencedObjectCollector struct {
	schema     *tableSchema
	objectKeys []string
}

func newReferencedObjectCollector(schema *tableSchema) *referencedObjectCollector {
	return &referencedObjectCollector{
		schema: schema,
	}
}

func (c *referencedObjectCollector) handleRow(values []any) error {
	bean := c.schema.registration.NewBean()

	switch bean.(type) {
	case *models.User, *models.TransactionPictureInfo, *models.UserCustomIcon:
	default:
		return nil
	}

	valuesByName := make(map[string]any, len(values))

	for i, column := range c.schema.columns {
		valuesByName[column.name] = values[i]
	}

	filledBean, err := newBeanFromValues(c.schema, valuesByName)

	if err != nil {
		return err
	}

	if objectKey := getReferencedObjectKey(filledBean); objectKey != "" {
		c.objectKeys = append(c.objectKeys, objectKey)
	}

	return nil
}

// getReferencedObjectKey returns the object storage key ("<path prefix>/<object path>") referenced by the model, or empty if not referenced.
// The object paths must be kept consistent with the paths used in services.ServiceUsingStorage
func getReferencedObjectKey(bean any) string {
	switch model := bean.(type) {
	case *models.User:
		if model.CustomAvatarType != "" && !model.Deleted {
			return fmt.Sprintf("avatar/%d.%s", model.Uid, model.CustomAvatarType)
		}
	case *models.TransactionPictureInfo:
		if !model.Deleted {
			return fmt.Sprintf("transaction/%d/%d.%s", model.Uid, model.PictureId, model.PictureExtension)
		}
	case *models.UserCustomIcon:
		if !model.Deleted {
			return fmt.Sprintf("icon/%d/%d.%s", model.Uid, model.IconId, models.UserCustomIconFileExtension)
		}
	}

	return ""
}

func syncDirectory(dir string) {
	dirFile, err := os.Open(dir)

	if err != nil {
		return
	}

	_ = dirFile.Sync()
	_ = dirFile.Close()
}

func timePointer(t time.Time) *time.Time {
	utcTime := t.UTC()
	return &utcTime
}
