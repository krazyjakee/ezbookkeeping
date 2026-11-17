package backup

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	// ManifestFormatVersion is the current version of the backup archive format
	ManifestFormatVersion = 1

	// BackupFileNamePrefix is the file name prefix of all backup archives
	BackupFileNamePrefix = "ezrbookkeeping-backup-"

	// BackupFileExtension is the file extension of plain (unencrypted) backup archives
	BackupFileExtension = ".tar.gz"

	// EncryptedBackupFileExtension is the file extension of encrypted backup archives
	EncryptedBackupFileExtension = ".tar.gz.enc"

	backupFileTimestampLayout = "20060102T150405.000Z"
	partialBackupFileSuffix   = ".partial"

	manifestEntryName   = "manifest.json"
	databaseEntryPrefix = "database/"
	databaseEntrySuffix = ".jsonl"
	storageEntryPrefix  = "storage/"
	configEntryName     = "config/ezbookkeeping.ini"

	maxManifestSize = 256 * 1024 * 1024
)

// Errors of backup and restore
var (
	ErrBackupAlreadyRunning        = errors.New("another backup is already running")
	ErrBackupPassphraseRequired    = errors.New("backup passphrase is required for encrypted backups (set \"backup_passphrase\" in [backup] section, EBK_BACKUP_BACKUP_PASSPHRASE / EBKCFP_BACKUP_BACKUP_PASSPHRASE environment variable, or --passphrase-file)")
	ErrBackupWrongPassphrase       = errors.New("cannot decrypt backup: wrong passphrase")
	ErrBackupCorrupted             = errors.New("backup archive is corrupted")
	ErrBackupTruncated             = errors.New("backup archive is truncated or incomplete")
	ErrBackupManifestMissing       = errors.New("backup manifest is missing (backup is incomplete or corrupted)")
	ErrBackupVerificationFailed    = errors.New("backup verification failed")
	ErrBackupIncompatible          = errors.New("backup is not compatible with this version of ezrbookkeeping")
	ErrRestoreTargetNotEmpty       = errors.New("target database or object storage is not empty, use --force together with --yes-overwrite-existing-data to overwrite existing data")
	ErrRestoreVerificationFailed   = errors.New("restored data verification failed")
	ErrBackupInvalidFileName       = errors.New("invalid backup file name")
	ErrBackupUnsupportedColumnType = errors.New("unsupported database column type for backup")
)

// Manifest represents the metadata of a backup archive
type Manifest struct {
	FormatVersion         int              `json:"formatVersion"`
	ApplicationName       string           `json:"applicationName"`
	AppVersion            string           `json:"appVersion"`
	AppCommitHash         string           `json:"appCommitHash"`
	CreatedAt             time.Time        `json:"createdAt"`
	DatabaseType          string           `json:"databaseType"`
	StorageType           string           `json:"storageType"`
	ConfigIncluded        bool             `json:"configIncluded"`
	Tables                []*ManifestTable `json:"tables"`
	Entries               []*ManifestEntry `json:"entries"`
	StorageObjectCount    int              `json:"storageObjectCount"`
	MissingStorageObjects []string         `json:"missingStorageObjects"`
	Warnings              []string         `json:"warnings"`
}

// ManifestTable represents a database table in the backup archive
type ManifestTable struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rows        int64    `json:"rows"`
	Columns     []string `json:"columns"`
	EntryName   string   `json:"entryName"`
}

// ManifestEntry represents a file entry in the backup archive
type ManifestEntry struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// GetEntry returns the manifest entry by name
func (m *Manifest) GetEntry(name string) *ManifestEntry {
	for _, entry := range m.Entries {
		if entry.Name == name {
			return entry
		}
	}

	return nil
}

// GetTable returns the manifest table by name
func (m *Manifest) GetTable(name string) *ManifestTable {
	for _, table := range m.Tables {
		if table.Name == name {
			return table
		}
	}

	return nil
}

// GetStorageEntries returns all the storage object entries
func (m *Manifest) GetStorageEntries() []*ManifestEntry {
	entries := make([]*ManifestEntry, 0)

	for _, entry := range m.Entries {
		if strings.HasPrefix(entry.Name, storageEntryPrefix) {
			entries = append(entries, entry)
		}
	}

	return entries
}

// GetBackupFileName returns the backup file name of the specified creation time
func GetBackupFileName(createdAt time.Time, encrypted bool) string {
	extension := BackupFileExtension

	if encrypted {
		extension = EncryptedBackupFileExtension
	}

	return BackupFileNamePrefix + createdAt.UTC().Format(backupFileTimestampLayout) + extension
}

// ParseBackupFileName returns the creation time and whether encrypted of the specified backup file name
func ParseBackupFileName(fileName string) (time.Time, bool, error) {
	if !strings.HasPrefix(fileName, BackupFileNamePrefix) {
		return time.Time{}, false, ErrBackupInvalidFileName
	}

	encrypted := false
	timestamp := fileName[len(BackupFileNamePrefix):]

	if strings.HasSuffix(timestamp, EncryptedBackupFileExtension) {
		encrypted = true
		timestamp = timestamp[:len(timestamp)-len(EncryptedBackupFileExtension)]
	} else if strings.HasSuffix(timestamp, BackupFileExtension) {
		timestamp = timestamp[:len(timestamp)-len(BackupFileExtension)]
	} else {
		return time.Time{}, false, ErrBackupInvalidFileName
	}

	createdAt, err := time.Parse(backupFileTimestampLayout, timestamp)

	if err != nil {
		return time.Time{}, false, ErrBackupInvalidFileName
	}

	return createdAt.UTC(), encrypted, nil
}

func getDatabaseEntryName(tableName string) string {
	return databaseEntryPrefix + tableName + databaseEntrySuffix
}

func getStorageEntryName(pathPrefix string, objectPath string) string {
	return storageEntryPrefix + pathPrefix + "/" + objectPath
}

// parseStorageEntryName returns the storage path prefix and the object path of the storage entry name
func parseStorageEntryName(entryName string) (string, string, bool) {
	if !strings.HasPrefix(entryName, storageEntryPrefix) {
		return "", "", false
	}

	parts := strings.SplitN(entryName[len(storageEntryPrefix):], "/", 2)

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// validateEntryName returns an error if the archive entry name is unsafe or unexpected
func validateEntryName(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) {
		return fmt.Errorf("%w: unsafe entry name \"%s\"", ErrBackupCorrupted, name)
	}

	if path.Clean(name) != name {
		return fmt.Errorf("%w: unsafe entry name \"%s\"", ErrBackupCorrupted, name)
	}

	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." || part == "" {
			return fmt.Errorf("%w: unsafe entry name \"%s\"", ErrBackupCorrupted, name)
		}
	}

	if name == manifestEntryName || name == configEntryName {
		return nil
	}

	if strings.HasPrefix(name, databaseEntryPrefix) && strings.HasSuffix(name, databaseEntrySuffix) && !strings.Contains(name[len(databaseEntryPrefix):], "/") {
		return nil
	}

	if _, _, ok := parseStorageEntryName(name); ok {
		return nil
	}

	return fmt.Errorf("%w: unexpected entry \"%s\"", ErrBackupCorrupted, name)
}

// compareVersions compares two application versions (e.g. "1.2.3", "v1.2.3-beta"), returns -1, 0, 1, and false if any version cannot be parsed
func compareVersions(a string, b string) (int, bool) {
	partsA, okA := parseVersion(a)
	partsB, okB := parseVersion(b)

	if !okA || !okB {
		return 0, false
	}

	for i := 0; i < 3; i++ {
		if partsA[i] < partsB[i] {
			return -1, true
		} else if partsA[i] > partsB[i] {
			return 1, true
		}
	}

	return 0, true
}

func parseVersion(version string) ([3]int, bool) {
	result := [3]int{}
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")

	if idx := strings.IndexAny(version, "-+ "); idx >= 0 {
		version = version[:idx]
	}

	if version == "" {
		return result, false
	}

	parts := strings.Split(version, ".")

	if len(parts) > 3 {
		return result, false
	}

	for i, part := range parts {
		value, err := strconv.Atoi(part)

		if err != nil || value < 0 {
			return result, false
		}

		result[i] = value
	}

	return result, true
}
