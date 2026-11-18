package backup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/storage"
)

const maxJsonLineSize = 512 * 1024 * 1024

// VerifyReport represents the result of verifying a backup archive
type VerifyReport struct {
	FilePath  string
	Encrypted bool
	Manifest  *Manifest
	TableRows map[string]int64
	Problems  []string
	Warnings  []string
}

// OK returns whether the backup passed all the checks
func (r *VerifyReport) OK() bool {
	return len(r.Problems) == 0
}

func (r *VerifyReport) addProblem(format string, args ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

func (r *VerifyReport) addWarning(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// VerifyBackup verifies the backup archive: decryptability, archive integrity, manifest, checksums,
// json validity of database rows, and the compatibility with the current application version and database tables
func VerifyBackup(c core.Context, filePath string, passphrase string) (*VerifyReport, error) {
	return verifyArchive(filePath, passphrase, true)
}

func verifyArchive(filePath string, passphrase string, checkCompatibility bool) (*VerifyReport, error) {
	report := &VerifyReport{
		FilePath:  filePath,
		TableRows: make(map[string]int64),
	}

	var schemasByName map[string]*tableSchema

	if checkCompatibility {
		schemas, err := loadTableSchemas()

		if err != nil {
			return nil, err
		}

		schemasByName = make(map[string]*tableSchema, len(schemas))

		for _, schema := range schemas {
			schemasByName[schema.name] = schema
		}
	}

	databaseEntryErrors := make(map[string]string)

	result, err := readArchive(filePath, passphrase, func(name string, size int64, reader io.Reader) error {
		if !strings.HasPrefix(name, databaseEntryPrefix) {
			return nil
		}

		tableName := strings.TrimSuffix(strings.TrimPrefix(name, databaseEntryPrefix), databaseEntrySuffix)
		var schema *tableSchema

		if checkCompatibility {
			schema = schemasByName[tableName]

			if schema == nil {
				databaseEntryErrors[name] = fmt.Sprintf("table \"%s\" does not exist in this version of ezrbookkeeping (the backup was probably created by a newer version)", tableName)
				return nil
			}
		}

		rowCount, err := scanJsonLines(reader, func(lineNumber int64, line []byte) error {
			if schema != nil {
				if _, err := decodeRow(schema, line); err != nil {
					return fmt.Errorf("line %d: %s", lineNumber, err.Error())
				}
			} else if !json.Valid(line) || len(line) < 2 || line[0] != '{' {
				return fmt.Errorf("line %d: invalid json object", lineNumber)
			}

			return nil
		})

		if err != nil {
			databaseEntryErrors[name] = err.Error()
		}

		report.TableRows[tableName] = rowCount
		return nil
	})

	if err != nil {
		return nil, err
	}

	report.Encrypted = result.Encrypted
	report.Manifest = result.Manifest
	manifest := result.Manifest

	if manifest.FormatVersion < 1 {
		report.addProblem("invalid backup format version %d", manifest.FormatVersion)
	} else if manifest.FormatVersion > ManifestFormatVersion {
		report.addProblem("backup format version %d is not supported by this version of ezrbookkeeping (supported version is %d)", manifest.FormatVersion, ManifestFormatVersion)
	}

	// checksums of all entries
	manifestEntries := make(map[string]bool, len(manifest.Entries))

	for _, entry := range manifest.Entries {
		manifestEntries[entry.Name] = true
		actualEntry := result.Entries[entry.Name]

		if actualEntry == nil {
			report.addProblem("entry \"%s\" is listed in manifest but missing in archive", entry.Name)
		} else if actualEntry.Size != entry.Size {
			report.addProblem("size of entry \"%s\" mismatches (expected %d bytes, actual %d bytes)", entry.Name, entry.Size, actualEntry.Size)
		} else if actualEntry.SHA256 != entry.SHA256 {
			report.addProblem("checksum of entry \"%s\" mismatches", entry.Name)
		}
	}

	storageEntryCount := 0
	knownStoragePathPrefixes := make(map[string]bool)

	for _, pathPrefix := range storage.GetAllObjectStoragePathPrefixes() {
		knownStoragePathPrefixes[pathPrefix] = true
	}

	for name := range result.Entries {
		if !manifestEntries[name] {
			report.addProblem("entry \"%s\" is not listed in manifest", name)
		}

		if pathPrefix, _, ok := parseStorageEntryName(name); ok {
			storageEntryCount++

			if !knownStoragePathPrefixes[pathPrefix] {
				report.addProblem("object storage \"%s\" of entry \"%s\" is not supported by this version of ezrbookkeeping", pathPrefix, name)
			}
		}
	}

	if storageEntryCount != manifest.StorageObjectCount {
		report.addProblem("storage object count mismatches (manifest %d, archive %d)", manifest.StorageObjectCount, storageEntryCount)
	}

	if manifest.ConfigIncluded != (result.Entries[configEntryName] != nil) {
		report.addProblem("configuration file inclusion mismatches manifest")
	}

	// database tables
	manifestTables := make(map[string]bool, len(manifest.Tables))

	for _, table := range manifest.Tables {
		manifestTables[table.EntryName] = true

		if table.EntryName != getDatabaseEntryName(table.Name) {
			report.addProblem("invalid entry name of table \"%s\"", table.Name)
			continue
		}

		if errorMessage, exists := databaseEntryErrors[table.EntryName]; exists {
			report.addProblem("table \"%s\": %s", table.Name, errorMessage)
			continue
		}

		if rows, exists := report.TableRows[table.Name]; !exists {
			report.addProblem("data of table \"%s\" is missing", table.Name)
		} else if rows != table.Rows {
			report.addProblem("row count of table \"%s\" mismatches (manifest %d, archive %d)", table.Name, table.Rows, rows)
		}
	}

	for name := range result.Entries {
		if strings.HasPrefix(name, databaseEntryPrefix) && !manifestTables[name] {
			report.addProblem("database entry \"%s\" is not listed in manifest tables", name)
		}
	}

	for _, missingObject := range manifest.MissingStorageObjects {
		report.addWarning("object \"%s\" was referenced by database but missing in object storage when the backup was created", missingObject)
	}

	for _, warning := range manifest.Warnings {
		report.addWarning("%s", warning)
	}

	if checkCompatibility {
		checkCompatibilityWithCurrentVersion(report, schemasByName)
	}

	if !report.OK() {
		return report, fmt.Errorf("%w: %s", ErrBackupVerificationFailed, report.Problems[0])
	}

	return report, nil
}

func checkCompatibilityWithCurrentVersion(report *VerifyReport, schemasByName map[string]*tableSchema) {
	manifest := report.Manifest
	comparison, ok := compareVersions(manifest.AppVersion, core.Version)

	if !ok {
		backupVersion := manifest.AppVersion
		currentVersion := core.Version

		if backupVersion == "" {
			backupVersion = "unknown"
		}

		if currentVersion == "" {
			currentVersion = "unknown"
		}

		if backupVersion != currentVersion || backupVersion == "unknown" {
			report.addWarning("cannot compare application versions (backup %s, current %s), compatibility is checked by database tables and columns only", backupVersion, currentVersion)
		}
	} else if comparison > 0 {
		report.addProblem("backup was created by a newer version of ezrbookkeeping (%s) than the current version (%s), please upgrade ezrbookkeeping before restoring", manifest.AppVersion, core.Version)
	}

	backupTables := make(map[string]*ManifestTable, len(manifest.Tables))

	for _, table := range manifest.Tables {
		backupTables[table.Name] = table

		if schemasByName[table.Name] == nil {
			// already reported as entry error, report again only if the table data entry was missing
			if _, exists := report.TableRows[table.Name]; !exists {
				report.addProblem("table \"%s\" does not exist in this version of ezrbookkeeping", table.Name)
			}
		}
	}

	for name := range schemasByName {
		if backupTables[name] == nil {
			report.addWarning("table \"%s\" does not exist in backup (the backup was probably created by an older version), it will be empty after restore", name)
		}
	}
}

// scanJsonLines calls the handler for every non-empty line, and returns the line count
func scanJsonLines(reader io.Reader, handler func(lineNumber int64, line []byte) error) (int64, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), maxJsonLineSize)
	lineCount := int64(0)

	for scanner.Scan() {
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		lineCount++

		if err := handler(lineCount, line); err != nil {
			return lineCount, err
		}
	}

	if err := scanner.Err(); err != nil {
		return lineCount, err
	}

	return lineCount, nil
}
