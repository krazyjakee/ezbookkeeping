package backup

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BackupFileInfo represents a completed backup file in a backup directory
type BackupFileInfo struct {
	Name      string
	Path      string
	CreatedAt time.Time
	Size      int64
	Encrypted bool
}

// ListBackupFiles returns all the completed backup files in the backup directory, newest first
func ListBackupFiles(backupDir string) ([]*BackupFileInfo, error) {
	entries, err := os.ReadDir(backupDir)

	if os.IsNotExist(err) {
		return []*BackupFileInfo{}, nil
	} else if err != nil {
		return nil, err
	}

	backups := make([]*BackupFileInfo, 0, len(entries))

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		createdAt, encrypted, err := ParseBackupFileName(entry.Name())

		if err != nil {
			continue
		}

		fileInfo, err := entry.Info()

		if err != nil {
			continue
		}

		backups = append(backups, &BackupFileInfo{
			Name:      entry.Name(),
			Path:      filepath.Join(backupDir, entry.Name()),
			CreatedAt: createdAt,
			Size:      fileInfo.Size(),
			Encrypted: encrypted,
		})
	}

	sortBackupFilesNewestFirst(backups)

	return backups, nil
}

func sortBackupFilesNewestFirst(backups []*BackupFileInfo) {
	sort.SliceStable(backups, func(i, j int) bool {
		if backups[i].CreatedAt.Equal(backups[j].CreatedAt) {
			return backups[i].Name > backups[j].Name
		}

		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
}

// SelectBackupsToDelete returns the backups which exceed the retention count or days.
// The newest backup is never selected. A retention value of 0 disables the related limit.
func SelectBackupsToDelete(backups []*BackupFileInfo, retentionCount uint32, retentionDays uint32, now time.Time) []*BackupFileInfo {
	sortedBackups := make([]*BackupFileInfo, len(backups))
	copy(sortedBackups, backups)
	sortBackupFilesNewestFirst(sortedBackups)

	toDelete := make([]*BackupFileInfo, 0)
	cutoffTime := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)

	for i, backup := range sortedBackups {
		if i == 0 {
			continue
		}

		if retentionCount > 0 && i >= int(retentionCount) {
			toDelete = append(toDelete, backup)
			continue
		}

		if retentionDays > 0 && backup.CreatedAt.Before(cutoffTime) {
			toDelete = append(toDelete, backup)
		}
	}

	return toDelete
}

// applyRetention removes old backups exceeding the retention policy, it does nothing unless the latest successful backup file exists
func applyRetention(backupDir string, latestBackupFilePath string, retentionCount uint32, retentionDays uint32, now time.Time) ([]string, []string) {
	if retentionCount == 0 && retentionDays == 0 {
		return nil, nil
	}

	if _, err := os.Stat(latestBackupFilePath); err != nil {
		return nil, []string{"latest backup file does not exist, retention policy is skipped"}
	}

	backups, err := ListBackupFiles(backupDir)

	if err != nil {
		return nil, []string{err.Error()}
	}

	if len(backups) < 1 || backups[0].Path != latestBackupFilePath {
		return nil, []string{"latest backup is not the newest backup file in backup directory, retention policy is skipped"}
	}

	deleted := make([]string, 0)
	errors := make([]string, 0)

	for _, backup := range SelectBackupsToDelete(backups, retentionCount, retentionDays, now) {
		if err := os.Remove(backup.Path); err != nil {
			errors = append(errors, err.Error())
		} else {
			deleted = append(deleted, backup.Name)
		}
	}

	return deleted, errors
}
