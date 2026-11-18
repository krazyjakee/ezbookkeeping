package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const backupStatusFileName = "backup-status.json"

var statusFileMutex sync.Mutex

// BackupStatus represents the status of the latest backup runs in a backup directory
type BackupStatus struct {
	Running              bool       `json:"running"`
	LastRunStartedAt     *time.Time `json:"lastRunStartedAt,omitempty"`
	LastRunFinishedAt    *time.Time `json:"lastRunFinishedAt,omitempty"`
	LastRunSuccess       bool       `json:"lastRunSuccess"`
	LastSuccessAt        *time.Time `json:"lastSuccessAt,omitempty"`
	LastSuccessFile      string     `json:"lastSuccessFile,omitempty"`
	LastSuccessSize      int64      `json:"lastSuccessSize,omitempty"`
	LastSuccessWarnings  int        `json:"lastSuccessWarnings,omitempty"`
	LastFailureAt        *time.Time `json:"lastFailureAt,omitempty"`
	LastError            string     `json:"lastError,omitempty"`
	LastRetentionDeleted []string   `json:"lastRetentionDeleted,omitempty"`
}

// ReadBackupStatus returns the backup status of the specified backup directory, or nil if there is no status file
func ReadBackupStatus(backupDir string) (*BackupStatus, error) {
	data, err := os.ReadFile(filepath.Join(backupDir, backupStatusFileName))

	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	status := &BackupStatus{}

	if err := json.Unmarshal(data, status); err != nil {
		return nil, err
	}

	return status, nil
}

func updateBackupStatus(backupDir string, update func(status *BackupStatus)) error {
	statusFileMutex.Lock()
	defer statusFileMutex.Unlock()

	status, err := ReadBackupStatus(backupDir)

	if err != nil || status == nil {
		status = &BackupStatus{}
	}

	update(status)

	data, err := json.MarshalIndent(status, "", "  ")

	if err != nil {
		return err
	}

	return writeFileAtomically(filepath.Join(backupDir, backupStatusFileName), data, 0600)
}

func writeFileAtomically(filePath string, data []byte, perm os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), "."+filepath.Base(filePath)+".tmp-")

	if err != nil {
		return err
	}

	tempFilePath := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
		return err
	}

	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
		return err
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempFilePath)
		return err
	}

	if err := os.Rename(tempFilePath, filePath); err != nil {
		_ = os.Remove(tempFilePath)
		return err
	}

	return nil
}
