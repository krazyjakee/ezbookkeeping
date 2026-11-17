package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	backupLockFileName = ".ezrbookkeeping-backup.lock"

	// a lock file older than this is always considered stale (e.g. created on another host which has gone away)
	staleBackupLockMaxAge = 24 * time.Hour
)

// processBackupMutex prevents overlapping backups in the current process
var processBackupMutex sync.Mutex

type backupLockInfo struct {
	Pid       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	StartedAt time.Time `json:"startedAt"`
}

type backupLock struct {
	lockFilePath string
}

// acquireBackupLock acquires the in-process mutex and the lock file in the backup directory
func acquireBackupLock(backupDir string) (*backupLock, error) {
	if !processBackupMutex.TryLock() {
		return nil, ErrBackupAlreadyRunning
	}

	lockFilePath := filepath.Join(backupDir, backupLockFileName)
	err := tryCreateLockFile(lockFilePath, time.Now())

	if err != nil {
		processBackupMutex.Unlock()
		return nil, err
	}

	return &backupLock{
		lockFilePath: lockFilePath,
	}, nil
}

func (l *backupLock) release() {
	_ = os.Remove(l.lockFilePath)
	processBackupMutex.Unlock()
}

// tryCreateLockFile creates the lock file exclusively, a stale lock file is removed and the creation is retried once.
// It must be called while holding processBackupMutex.
func tryCreateLockFile(lockFilePath string, now time.Time) error {
	hostname, _ := os.Hostname()
	info := &backupLockInfo{
		Pid:       os.Getpid(),
		Hostname:  hostname,
		StartedAt: now.UTC(),
	}

	data, err := json.Marshal(info)

	if err != nil {
		return err
	}

	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

		if err == nil {
			_, writeErr := file.Write(data)
			closeErr := file.Close()

			if writeErr != nil || closeErr != nil {
				_ = os.Remove(lockFilePath)
				return errors.Join(writeErr, closeErr)
			}

			return nil
		}

		if !os.IsExist(err) {
			return err
		}

		stale, existingInfo := isStaleLockFile(lockFilePath, hostname, now)

		if !stale {
			if existingInfo != nil {
				return fmt.Errorf("%w (pid %d on %s, started at %s, lock file %s)", ErrBackupAlreadyRunning, existingInfo.Pid, existingInfo.Hostname, existingInfo.StartedAt.Format(time.RFC3339), lockFilePath)
			}

			return fmt.Errorf("%w (lock file %s)", ErrBackupAlreadyRunning, lockFilePath)
		}

		if err := os.Remove(lockFilePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return fmt.Errorf("%w (lock file %s)", ErrBackupAlreadyRunning, lockFilePath)
}

// isStaleLockFile returns whether the existing lock file is left by a process which has exited
func isStaleLockFile(lockFilePath string, currentHostname string, now time.Time) (bool, *backupLockInfo) {
	fileInfo, err := os.Stat(lockFilePath)

	if err != nil {
		return os.IsNotExist(err), nil
	}

	data, err := os.ReadFile(lockFilePath)

	if err != nil {
		return false, nil
	}

	info := &backupLockInfo{}

	if err := json.Unmarshal(data, info); err != nil || info.Pid <= 0 {
		// the lock file may be being written by another process, only treat it as stale after a while
		return now.Sub(fileInfo.ModTime()) > time.Minute, nil
	}

	if now.Sub(info.StartedAt) > staleBackupLockMaxAge && now.Sub(fileInfo.ModTime()) > staleBackupLockMaxAge {
		return true, info
	}

	if info.Hostname == currentHostname {
		// the caller holds the in-process mutex, so a lock file with the current pid must be left by a previous process
		// with the same pid (e.g. pid 1 in a restarted container)
		if info.Pid == os.Getpid() {
			return true, info
		}

		return !isProcessRunning(info.Pid), info
	}

	return false, info
}
