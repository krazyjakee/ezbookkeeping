package backup

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encryptForTest(t *testing.T, plaintext []byte, passphrase string, chunkSize uint32) []byte {
	buffer := &bytes.Buffer{}
	writer, err := newEncryptingWriterWithParams(buffer, passphrase, 10, chunkSize)
	require.NoError(t, err)

	// write in odd-sized pieces to exercise chunk boundaries
	for offset := 0; offset < len(plaintext); offset += 7 {
		end := min(offset+7, len(plaintext))
		_, err := writer.Write(plaintext[offset:end])
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func decryptForTest(ciphertext []byte, passphrase string) ([]byte, error) {
	reader, err := newDecryptingReader(bytes.NewReader(ciphertext), passphrase)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(reader)
}

func TestEncryption_RoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 63, 64, 65, 128, 1000} {
		plaintext := make([]byte, size)
		_, _ = rand.Read(plaintext)

		ciphertext := encryptForTest(t, plaintext, "correct horse battery staple", 64)
		decrypted, err := decryptForTest(ciphertext, "correct horse battery staple")

		require.NoError(t, err, "size %d", size)
		assert.Equal(t, plaintext, decrypted, "size %d", size)
	}
}

func TestEncryption_WrongPassphrase(t *testing.T) {
	ciphertext := encryptForTest(t, []byte("financial data"), "passphrase-1", 64)
	_, err := decryptForTest(ciphertext, "passphrase-2")

	assert.ErrorIs(t, err, ErrBackupWrongPassphrase)
}

func TestEncryption_EmptyPassphrase(t *testing.T) {
	_, err := newEncryptingWriter(&bytes.Buffer{}, "")
	assert.ErrorIs(t, err, ErrBackupPassphraseRequired)
}

func TestEncryption_TruncatedAndTampered(t *testing.T) {
	plaintext := bytes.Repeat([]byte("0123456789"), 50)
	ciphertext := encryptForTest(t, plaintext, "secret", 64)

	// truncated at a record boundary (final chunk removed) and in the middle of a record
	for _, cut := range []int{len(ciphertext) - 1, len(ciphertext) - 30, encryptionHeaderSize + 10} {
		_, err := decryptForTest(ciphertext[:cut], "secret")
		assert.ErrorIs(t, err, ErrBackupTruncated, "cut at %d", cut)
	}

	// remove the whole final record, so that the stream ends at a record boundary
	lastRecordStart := findLastRecordStart(t, ciphertext)
	_, err := decryptForTest(ciphertext[:lastRecordStart], "secret")
	assert.ErrorIs(t, err, ErrBackupTruncated)

	// flipped bit
	tampered := append([]byte{}, ciphertext...)
	tampered[len(tampered)-5] ^= 0x01
	_, err = decryptForTest(tampered, "secret")
	assert.ErrorIs(t, err, ErrBackupCorrupted)

	// appended data
	_, err = decryptForTest(append(append([]byte{}, ciphertext...), 0x00), "secret")
	assert.ErrorIs(t, err, ErrBackupCorrupted)
}

func findLastRecordStart(t *testing.T, ciphertext []byte) int {
	offset := encryptionHeaderSize
	lastStart := -1

	for offset < len(ciphertext) {
		lastStart = offset
		length := int(ciphertext[offset+1])<<24 | int(ciphertext[offset+2])<<16 | int(ciphertext[offset+3])<<8 | int(ciphertext[offset+4])
		offset += 5 + length
	}

	require.Equal(t, len(ciphertext), offset)
	return lastStart
}

func TestParseBackupFileName(t *testing.T) {
	createdAt := time.Date(2026, 11, 17, 3, 0, 0, 123000000, time.UTC)

	name := GetBackupFileName(createdAt, false)
	assert.Equal(t, "ezrbookkeeping-backup-20261117T030000.123Z.tar.gz", name)

	parsedTime, encrypted, err := ParseBackupFileName(name)
	require.NoError(t, err)
	assert.True(t, createdAt.Equal(parsedTime))
	assert.False(t, encrypted)

	parsedTime, encrypted, err = ParseBackupFileName(GetBackupFileName(createdAt, true))
	require.NoError(t, err)
	assert.True(t, createdAt.Equal(parsedTime))
	assert.True(t, encrypted)

	for _, invalidName := range []string{
		"ezrbookkeeping-backup-20261117T030000.123Z.tar.gz.partial",
		"other-20261117T030000.123Z.tar.gz",
		"ezrbookkeeping-backup-invalid.tar.gz",
		"backup-status.json",
	} {
		_, _, err = ParseBackupFileName(invalidName)
		assert.ErrorIs(t, err, ErrBackupInvalidFileName, invalidName)
	}
}

func TestValidateEntryName(t *testing.T) {
	for _, validName := range []string{"manifest.json", "config/ezbookkeeping.ini", "database/user.jsonl", "storage/avatar/1.png", "storage/transaction/1/2.pdf"} {
		assert.NoError(t, validateEntryName(validName), validName)
	}

	for _, invalidName := range []string{"", "/etc/passwd", "../x", "storage/../../x", "storage/avatar/../../x", "database/a/b.jsonl", "storage/avatar", "other.txt", "storage\\avatar\\1.png", "storage/avatar//1.png"} {
		assert.Error(t, validateEntryName(invalidName), invalidName)
	}
}

func TestCompareVersions(t *testing.T) {
	comparison, ok := compareVersions("2.2.0", "2.10.0")
	assert.True(t, ok)
	assert.Equal(t, -1, comparison)

	comparison, ok = compareVersions("v2.3.0-beta", "2.2.9")
	assert.True(t, ok)
	assert.Equal(t, 1, comparison)

	comparison, ok = compareVersions("2.2", "2.2.0")
	assert.True(t, ok)
	assert.Equal(t, 0, comparison)

	_, ok = compareVersions("", "2.2.0")
	assert.False(t, ok)

	_, ok = compareVersions("dev", "2.2.0")
	assert.False(t, ok)
}

func newBackupFileInfos(now time.Time, ages ...time.Duration) []*BackupFileInfo {
	backups := make([]*BackupFileInfo, 0, len(ages))

	for _, age := range ages {
		createdAt := now.Add(-age)
		backups = append(backups, &BackupFileInfo{
			Name:      GetBackupFileName(createdAt, false),
			CreatedAt: createdAt,
		})
	}

	return backups
}

func backupNames(backups []*BackupFileInfo) []string {
	names := make([]string, len(backups))

	for i, backup := range backups {
		names[i] = backup.Name
	}

	return names
}

func TestSelectBackupsToDelete_Count(t *testing.T) {
	now := time.Now().UTC()
	day := 24 * time.Hour
	backups := newBackupFileInfos(now, 3*day, 0, 1*day, 4*day, 2*day)

	toDelete := SelectBackupsToDelete(backups, 3, 0, now)
	assert.ElementsMatch(t, backupNames(newBackupFileInfos(now, 3*day, 4*day)), backupNames(toDelete))

	assert.Empty(t, SelectBackupsToDelete(backups, 0, 0, now))
	assert.Empty(t, SelectBackupsToDelete(backups, 5, 0, now))
}

func TestSelectBackupsToDelete_Days(t *testing.T) {
	now := time.Now().UTC()
	day := 24 * time.Hour
	backups := newBackupFileInfos(now, 1*day, 10*day, 8*day, 2*day)

	toDelete := SelectBackupsToDelete(backups, 0, 7, now)
	assert.ElementsMatch(t, backupNames(newBackupFileInfos(now, 10*day, 8*day)), backupNames(toDelete))

	// count and days limits are combined
	toDelete = SelectBackupsToDelete(backups, 1, 7, now)
	assert.Len(t, toDelete, 3)
}

func TestSelectBackupsToDelete_NeverDeleteNewest(t *testing.T) {
	now := time.Now().UTC()
	day := 24 * time.Hour
	backups := newBackupFileInfos(now, 30*day, 40*day)

	toDelete := SelectBackupsToDelete(backups, 1, 1, now)
	assert.Equal(t, backupNames(newBackupFileInfos(now, 40*day)), backupNames(toDelete))

	assert.Empty(t, SelectBackupsToDelete(newBackupFileInfos(now, 30*day), 1, 1, now))
}

func TestApplyRetention_OnlyCompletedBackups(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	day := 24 * time.Hour
	var latest string

	for i, age := range []time.Duration{0, day, 2 * day, 3 * day} {
		path := filepath.Join(dir, GetBackupFileName(now.Add(-age), i%2 == 1))
		require.NoError(t, os.WriteFile(path, []byte("x"), 0600))

		if i == 0 {
			latest = path
		}
	}

	partialFile := filepath.Join(dir, GetBackupFileName(now.Add(-10*day), false)+partialBackupFileSuffix)
	otherFile := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(partialFile, []byte("x"), 0600))
	require.NoError(t, os.WriteFile(otherFile, []byte("x"), 0600))

	deleted, errs := applyRetention(dir, latest, 2, 0, now)
	assert.Empty(t, errs)
	assert.Len(t, deleted, 2)

	backups, err := ListBackupFiles(dir)
	require.NoError(t, err)
	assert.Len(t, backups, 2)
	assert.Equal(t, latest, backups[0].Path)
	assert.FileExists(t, partialFile)
	assert.FileExists(t, otherFile)

	// retention is skipped if the latest backup does not exist
	deleted, errs = applyRetention(dir, filepath.Join(dir, "missing.tar.gz"), 1, 0, now)
	assert.Empty(t, deleted)
	assert.NotEmpty(t, errs)
}

func TestBackupLock_PreventsOverlappingRuns(t *testing.T) {
	dir := t.TempDir()

	lock, err := acquireBackupLock(dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, backupLockFileName))

	// in-process overlapping run
	_, err = acquireBackupLock(dir)
	assert.ErrorIs(t, err, ErrBackupAlreadyRunning)

	lock.release()
	assert.NoFileExists(t, filepath.Join(dir, backupLockFileName))

	lock, err = acquireBackupLock(dir)
	require.NoError(t, err)
	lock.release()
}

func writeLockFileForTest(t *testing.T, path string, info *backupLockInfo) {
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
}

func TestBackupLock_LockFileOfRunningProcess(t *testing.T) {
	dir := t.TempDir()
	lockFilePath := filepath.Join(dir, backupLockFileName)
	hostname, _ := os.Hostname()

	// the parent process (go test runner) is running
	writeLockFileForTest(t, lockFilePath, &backupLockInfo{Pid: os.Getppid(), Hostname: hostname, StartedAt: time.Now().UTC()})

	_, err := acquireBackupLock(dir)
	assert.ErrorIs(t, err, ErrBackupAlreadyRunning)

	// lock file from another host is respected while it is not too old
	writeLockFileForTest(t, lockFilePath, &backupLockInfo{Pid: 1, Hostname: hostname + "-other", StartedAt: time.Now().UTC()})

	_, err = acquireBackupLock(dir)
	assert.ErrorIs(t, err, ErrBackupAlreadyRunning)
}

func TestBackupLock_StaleLockFile(t *testing.T) {
	dir := t.TempDir()
	lockFilePath := filepath.Join(dir, backupLockFileName)
	hostname, _ := os.Hostname()

	// a process which has exited
	command := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, command.Run())
	exitedPid := command.Process.Pid

	writeLockFileForTest(t, lockFilePath, &backupLockInfo{Pid: exitedPid, Hostname: hostname, StartedAt: time.Now().UTC()})

	lock, err := acquireBackupLock(dir)
	require.NoError(t, err)
	lock.release()

	// lock file left by a previous process with the same pid (e.g. pid 1 in a restarted container)
	writeLockFileForTest(t, lockFilePath, &backupLockInfo{Pid: os.Getpid(), Hostname: hostname, StartedAt: time.Now().UTC()})

	lock, err = acquireBackupLock(dir)
	require.NoError(t, err)
	lock.release()

	// very old lock file from another host
	oldTime := time.Now().Add(-48 * time.Hour)
	writeLockFileForTest(t, lockFilePath, &backupLockInfo{Pid: 1, Hostname: hostname + "-other", StartedAt: oldTime.UTC()})
	require.NoError(t, os.Chtimes(lockFilePath, oldTime, oldTime))

	lock, err = acquireBackupLock(dir)
	require.NoError(t, err)
	lock.release()
}

func TestBackupStatus_Update(t *testing.T) {
	dir := t.TempDir()

	status, err := ReadBackupStatus(dir)
	require.NoError(t, err)
	assert.Nil(t, status)

	now := time.Now()
	require.NoError(t, updateBackupStatus(dir, func(status *BackupStatus) {
		status.LastFailureAt = timePointer(now)
		status.LastError = "boom"
	}))
	require.NoError(t, updateBackupStatus(dir, func(status *BackupStatus) {
		status.LastSuccessFile = "a.tar.gz"
	}))

	status, err = ReadBackupStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "boom", status.LastError)
	assert.Equal(t, "a.tar.gz", status.LastSuccessFile)

	fileInfo, err := os.Stat(filepath.Join(dir, backupStatusFileName))
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
	}
}

func TestReadArchive_DetectsMissingManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tar.gz")

	writer, err := newArchiveWriter(path, "", false, time.Now())
	require.NoError(t, err)
	_, err = writer.addBytes("config/ezbookkeeping.ini", []byte("[global]"))
	require.NoError(t, err)
	require.NoError(t, writer.tarWriter.Close())
	require.NoError(t, writer.gzipWriter.Close())
	require.NoError(t, writer.bufferedFile.Flush())
	require.NoError(t, writer.file.Close())

	_, err = readArchive(path, "", nil)
	assert.True(t, errors.Is(err, ErrBackupManifestMissing), "%v", err)
}
