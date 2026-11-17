package backup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/storage"
)

const testPassphrase = "test backup passphrase"

var testPictureData = []byte("%PDF-1.4 fake receipt attachment \x00\x01\x02 binary content")
var testAvatarData = []byte("\x89PNG fake avatar")
var testIconData = []byte("\x89PNG fake custom icon")

func newTestConfig(t *testing.T, dir string, name string) *settings.Config {
	storagePath := filepath.Join(dir, name+"-storage")
	require.NoError(t, os.MkdirAll(storagePath, 0700))

	configFilePath := filepath.Join(dir, name+".ini")
	require.NoError(t, os.WriteFile(configFilePath, []byte("[security]\nsecret_key = test-secret-"+name+"\n"), 0600))

	return &settings.Config{
		ConfigFilePath: configFilePath,
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:          settings.Sqlite3DbType,
			DatabasePath:          filepath.Join(dir, name+".db"),
			MaxIdleConnection:     2,
			ConnectionMaxLifeTime: 14400,
		},
		StorageType:         settings.LocalFileSystemObjectStorageType,
		LocalFileSystemPath: storagePath,
		BackupPath:          filepath.Join(dir, name+"-backups"),
		SecretKey:           "test-secret",
	}
}

func initializeTestInstance(t *testing.T, config *settings.Config) {
	settings.SetCurrentConfig(config)
	require.NoError(t, datastore.InitializeDataStore(config))

	for _, table := range datastore.GetAllTables() {
		require.NoError(t, table.Store().SyncStructs(table.NewBean()))
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}

func createTestData(t *testing.T, c core.Context, config *settings.Config) {
	now := time.Now().Unix()
	uid := int64(1234567890123456789)
	limit := int64(500000)

	user := &models.User{
		Uid:              uid,
		Username:         "alice",
		Email:            "alice@example.com",
		Nickname:         "Alice <\"quoted\" & unicode 中文>",
		Password:         "hashed-password",
		Salt:             "salt123456",
		CustomAvatarType: "png",
		DefaultCurrency:  "USD",
		EmailVerified:    true,
		CreatedUnixTime:  now,
		UpdatedUnixTime:  now,
	}

	accounts := []*models.Account{
		{AccountId: 1001, Uid: uid, Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "Wallet", Currency: "USD", Balance: 12345, Extend: &models.AccountExtend{}, CreatedUnixTime: now},
		{AccountId: 1002, Uid: uid, Category: models.ACCOUNT_CATEGORY_CREDIT_CARD, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "Card", Currency: "EUR", Balance: -6789, Hidden: true, Extend: &models.AccountExtend{CreditCardLimit: &limit, LastReconciledTime: int64Pointer(now)}, CreatedUnixTime: now},
	}

	categories := []*models.TransactionCategory{
		{CategoryId: 2001, Uid: uid, Type: models.CATEGORY_TYPE_EXPENSE, Name: "Food", Color: "ff0000", CreatedUnixTime: now},
		{CategoryId: 2002, Uid: uid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 2001, Name: "Restaurant", CreatedUnixTime: now},
	}

	tags := []*models.TransactionTag{
		{TagId: 3001, Uid: uid, Name: "business", CreatedUnixTime: now},
	}

	transactions := []*models.Transaction{
		{TransactionId: 4001, Uid: uid, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2002, AccountId: 1001, TransactionTime: now * 1000, Amount: 1500, Comment: "lunch\nwith newline", GeoLongitude: 116.397128, GeoLatitude: 39.916527, CreatedUnixTime: now},
		{TransactionId: 4002, Uid: uid, Deleted: true, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 2001, AccountId: 1002, TransactionTime: now*1000 + 1, Amount: 99, CreatedUnixTime: now},
	}

	tagIndexes := []*models.TransactionTagIndex{
		{TagIndexId: 5001, Uid: uid, TransactionTime: now * 1000, TagId: 3001, TransactionId: 4001, CreatedUnixTime: now},
	}

	templates := []*models.TransactionTemplate{
		{TemplateId: 6001, Uid: uid, TemplateType: models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE, Name: "Rent", Type: models.TRANSACTION_TYPE_EXPENSE, CategoryId: 2001, AccountId: 1001, ScheduledFrequencyType: models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_MONTHLY, ScheduledFrequency: "1", ScheduledStartTime: int64Pointer(now), Amount: 100000, TagIds: "3001", CreatedUnixTime: now},
		{TemplateId: 6002, Uid: uid, TemplateType: models.TRANSACTION_TEMPLATE_TYPE_NORMAL, Name: "Coffee", Type: models.TRANSACTION_TYPE_EXPENSE, CategoryId: 2001, AccountId: 1001, Amount: 300, CreatedUnixTime: now},
	}

	picture := &models.TransactionPictureInfo{Uid: uid, TransactionId: 4001, PictureId: 7001, PictureExtension: "pdf", OriginalFileName: "receipt.pdf", FileSize: int64(len(testPictureData)), CreatedUnixTime: now}
	icon := &models.UserCustomIcon{IconId: 8001, Uid: uid, CreatedUnixTime: now}
	cloudSetting := &models.UserApplicationCloudSetting{Uid: uid, Settings: models.ApplicationCloudSettingSlice{{SettingKey: "theme", SettingValue: "dark"}}, UpdatedUnixTime: now}

	err := datastore.Container.UserDataStore.DoTransaction(uid, c, func(sess datastore.DatabaseSession) error {
		for _, bean := range []any{user, &accounts, &categories, &tags, &transactions, &tagIndexes, &templates, picture, icon, cloudSetting} {
			if _, err := sess.Insert(bean); err != nil {
				return err
			}
		}

		return nil
	})
	require.NoError(t, err)

	objectStorages, err := storage.NewAllObjectStorages(config)
	require.NoError(t, err)
	require.NoError(t, objectStorages["transaction"].Save(c, "1234567890123456789/7001.pdf", newBytesObject(testPictureData)))
	require.NoError(t, objectStorages["avatar"].Save(c, "1234567890123456789.png", newBytesObject(testAvatarData)))
	// the custom icon object is intentionally missing to test the missing object warning
}

func readTestObject(t *testing.T, c core.Context, config *settings.Config, pathPrefix string, path string) []byte {
	objectStorages, err := storage.NewAllObjectStorages(config)
	require.NoError(t, err)

	object, exists, err := objectStorages[pathPrefix].Read(c, path)
	require.NoError(t, err)
	require.True(t, exists)
	defer object.Close()

	data, err := io.ReadAll(object)
	require.NoError(t, err)
	return data
}

type testSnapshot struct {
	users        []*models.User
	accounts     []*models.Account
	transactions []*models.Transaction
	categories   []*models.TransactionCategory
	tags         []*models.TransactionTag
	tagIndexes   []*models.TransactionTagIndex
	templates    []*models.TransactionTemplate
	pictures     []*models.TransactionPictureInfo
	icons        []*models.UserCustomIcon
	settings     []*models.UserApplicationCloudSetting
}

func loadTestSnapshot(t *testing.T, c core.Context) *testSnapshot {
	snapshot := &testSnapshot{}
	for _, target := range []any{&snapshot.users, &snapshot.accounts, &snapshot.transactions, &snapshot.categories, &snapshot.tags, &snapshot.tagIndexes, &snapshot.templates, &snapshot.pictures, &snapshot.icons, &snapshot.settings} {
		require.NoError(t, datastore.Container.UserDataStore.Query(c, 0).Find(target))
	}

	return snapshot
}

func TestBackupAndRestore_EndToEnd(t *testing.T) {
	c := core.NewNullContext()
	dir := t.TempDir()

	// source instance
	sourceConfig := newTestConfig(t, dir, "source")
	initializeTestInstance(t, sourceConfig)
	createTestData(t, c, sourceConfig)
	sourceSnapshot := loadTestSnapshot(t, c)

	require.Len(t, sourceSnapshot.users, 1)
	require.Len(t, sourceSnapshot.accounts, 2)
	require.NotNil(t, sourceSnapshot.accounts[1].Extend.CreditCardLimit)

	options := NewCreateOptionsFromConfig(sourceConfig)
	options.Encrypt = true
	options.Passphrase = testPassphrase
	options.IncludeConfig = true

	result, err := CreateBackup(c, sourceConfig, options)
	require.NoError(t, err)
	assert.FileExists(t, result.FilePath)
	assert.Equal(t, EncryptedBackupFileExtension, result.FilePath[len(result.FilePath)-len(EncryptedBackupFileExtension):])
	assert.Equal(t, 2, result.Manifest.StorageObjectCount)
	assert.Equal(t, []string{"icon/1234567890123456789/8001.png"}, result.Manifest.MissingStorageObjects)
	assert.True(t, result.Manifest.ConfigIncluded)

	fileInfo, err := os.Stat(result.FilePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	status, err := ReadBackupStatus(sourceConfig.BackupPath)
	require.NoError(t, err)
	assert.True(t, status.LastRunSuccess)
	assert.Equal(t, filepath.Base(result.FilePath), status.LastSuccessFile)

	backups, err := ListBackupFiles(sourceConfig.BackupPath)
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.True(t, backups[0].Encrypted)

	// verify
	verifyReport, err := VerifyBackup(c, result.FilePath, testPassphrase)
	require.NoError(t, err)
	assert.True(t, verifyReport.OK())
	assert.Equal(t, int64(2), verifyReport.TableRows["transaction"])

	_, err = VerifyBackup(c, result.FilePath, "wrong passphrase")
	assert.ErrorIs(t, err, ErrBackupWrongPassphrase)

	_, err = VerifyBackup(c, result.FilePath, "")
	assert.ErrorIs(t, err, ErrBackupPassphraseRequired)

	// unencrypted backup without config
	plainOptions := NewCreateOptionsFromConfig(sourceConfig)
	plainOptions.IncludeConfig = false
	plainOptions.OutputDir = filepath.Join(dir, "plain-backups")
	plainResult, err := CreateBackup(c, sourceConfig, plainOptions)
	require.NoError(t, err)
	assert.False(t, plainResult.Manifest.ConfigIncluded)

	plainVerifyReport, err := VerifyBackup(c, plainResult.FilePath, "")
	require.NoError(t, err)
	assert.False(t, plainVerifyReport.Encrypted)

	// target instance
	targetConfig := newTestConfig(t, dir, "target")
	initializeTestInstance(t, targetConfig)

	dryRunReport, err := RestoreBackup(c, targetConfig, result.FilePath, &RestoreOptions{Passphrase: testPassphrase, DryRun: true})
	require.NoError(t, err)
	assert.True(t, dryRunReport.TargetEmpty)
	assert.Empty(t, loadTestSnapshot(t, c).users, "dry run must not modify the target")

	configOutputPath := filepath.Join(dir, "restored.ini")
	restoreReport, err := RestoreBackup(c, targetConfig, result.FilePath, &RestoreOptions{Passphrase: testPassphrase, ConfigOutputPath: configOutputPath})
	require.NoError(t, err, "%v", restoreReport)
	assert.True(t, restoreReport.OK())
	assert.Equal(t, 1, restoreReport.UsersLoaded)
	assert.Equal(t, 2, restoreReport.StorageObjectsVerified)

	for _, table := range restoreReport.Tables {
		assert.True(t, table.DigestMatch, "table %s: %s", table.Name, table.DigestNote)
	}

	targetSnapshot := loadTestSnapshot(t, c)
	assert.Equal(t, sourceSnapshot, targetSnapshot)
	assert.Equal(t, testPictureData, readTestObject(t, c, targetConfig, "transaction", "1234567890123456789/7001.pdf"))
	assert.Equal(t, testAvatarData, readTestObject(t, c, targetConfig, "avatar", "1234567890123456789.png"))

	sourceConfigData, err := os.ReadFile(sourceConfig.ConfigFilePath)
	require.NoError(t, err)
	restoredConfigData, err := os.ReadFile(configOutputPath)
	require.NoError(t, err)
	assert.Equal(t, sourceConfigData, restoredConfigData)

	// restoring into a non-empty target requires explicit confirmation
	_, err = RestoreBackup(c, targetConfig, plainResult.FilePath, &RestoreOptions{})
	assert.ErrorIs(t, err, ErrRestoreTargetNotEmpty)

	_, err = RestoreBackup(c, targetConfig, plainResult.FilePath, &RestoreOptions{Force: true})
	assert.ErrorIs(t, err, ErrRestoreTargetNotEmpty)

	// modify target, then overwrite it with the backup
	_, err = datastore.Container.UserDataStore.Query(c, 0).Exec("UPDATE account SET balance = 1")
	require.NoError(t, err)
	_, err = datastore.Container.UserDataStore.Query(c, 0).Insert(&models.TransactionTag{TagId: 3999, Uid: 1, Name: "extra"})
	require.NoError(t, err)

	restoreReport, err = RestoreBackup(c, targetConfig, plainResult.FilePath, &RestoreOptions{Force: true, ConfirmOverwrite: true})
	require.NoError(t, err, "%v", restoreReport)
	assert.Equal(t, sourceSnapshot, loadTestSnapshot(t, c))
}

func TestBackup_DetectsCorruptedAndTruncatedArchives(t *testing.T) {
	c := core.NewNullContext()
	dir := t.TempDir()

	config := newTestConfig(t, dir, "corrupt")
	initializeTestInstance(t, config)
	createTestData(t, c, config)

	for _, encrypted := range []bool{false, true} {
		options := NewCreateOptionsFromConfig(config)
		options.Encrypt = encrypted
		options.Passphrase = testPassphrase
		options.OutputDir = filepath.Join(dir, "backups", map[bool]string{false: "plain", true: "encrypted"}[encrypted])

		result, err := CreateBackup(c, config, options)
		require.NoError(t, err)

		data, err := os.ReadFile(result.FilePath)
		require.NoError(t, err)

		writeVariant := func(name string, content []byte) string {
			path := filepath.Join(dir, name)
			require.NoError(t, os.WriteFile(path, content, 0600))
			return path
		}

		// truncated archives
		for _, size := range []int{len(data) / 3, len(data) / 2, len(data) - 1, len(data) - 8} {
			path := writeVariant("truncated", data[:size])
			_, err := VerifyBackup(c, path, testPassphrase)
			assert.Error(t, err, "encrypted %t, truncated to %d of %d bytes", encrypted, size, len(data))
		}

		// flipped bytes
		for _, offset := range []int{len(data) / 4, len(data) / 2, len(data) - 20} {
			corrupted := append([]byte{}, data...)
			corrupted[offset] ^= 0xff
			path := writeVariant("corrupted", corrupted)
			_, err := VerifyBackup(c, path, testPassphrase)
			assert.Error(t, err, "encrypted %t, corrupted at %d of %d bytes", encrypted, offset, len(data))
		}

		// appended garbage
		path := writeVariant("appended", append(append([]byte{}, data...), bytes.Repeat([]byte{0x42}, 64)...))
		_, err = VerifyBackup(c, path, testPassphrase)
		assert.Error(t, err, "encrypted %t, appended data", encrypted)

		// restore must refuse a corrupted backup before touching the target
		targetConfig := newTestConfig(t, dir, "corrupt-target")
		initializeTestInstance(t, targetConfig)

		corrupted := append([]byte{}, data...)
		corrupted[len(data)/2] ^= 0xff
		_, err = RestoreBackup(c, targetConfig, writeVariant("corrupted-restore", corrupted), &RestoreOptions{Passphrase: testPassphrase})
		assert.Error(t, err)
		assert.Empty(t, loadTestSnapshot(t, c).users)

		initializeTestInstance(t, config)
	}
}

func TestBackup_ChecksumMismatchDetected(t *testing.T) {
	c := core.NewNullContext()
	dir := t.TempDir()

	config := newTestConfig(t, dir, "checksum")
	initializeTestInstance(t, config)
	createTestData(t, c, config)

	// write a structurally valid archive whose manifest checksum does not match the entry
	path := filepath.Join(dir, "mismatch.tar.gz")
	writer, err := newArchiveWriter(path, "", false, time.Now())
	require.NoError(t, err)
	entry, err := writer.addBytes("storage/avatar/1.png", []byte("data"))
	require.NoError(t, err)
	entry.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	require.NoError(t, writer.finish(&Manifest{FormatVersion: ManifestFormatVersion, StorageObjectCount: 1}))

	report, err := VerifyBackup(c, path, "")
	assert.ErrorIs(t, err, ErrBackupVerificationFailed)
	assert.NotEmpty(t, report.Problems)

	// newer format / application version is refused
	path = filepath.Join(dir, "newer.tar.gz")
	writer, err = newArchiveWriter(path, "", false, time.Now())
	require.NoError(t, err)
	require.NoError(t, writer.finish(&Manifest{FormatVersion: ManifestFormatVersion + 1}))

	_, err = VerifyBackup(c, path, "")
	assert.ErrorIs(t, err, ErrBackupVerificationFailed)

	originalVersion := core.Version
	core.Version = "1.0.0"
	defer func() { core.Version = originalVersion }()

	path = filepath.Join(dir, "newer-app.tar.gz")
	writer, err = newArchiveWriter(path, "", false, time.Now())
	require.NoError(t, err)
	require.NoError(t, writer.finish(&Manifest{FormatVersion: ManifestFormatVersion, AppVersion: "9.0.0"}))

	report, err = VerifyBackup(c, path, "")
	assert.ErrorIs(t, err, ErrBackupVerificationFailed)
	assert.Contains(t, report.Problems[0], "newer version")

	// unknown column in a table (backup from a newer schema)
	path = filepath.Join(dir, "unknown-column.tar.gz")
	writer, err = newArchiveWriter(path, "", false, time.Now())
	require.NoError(t, err)
	_, err = writer.addBytes("database/user.jsonl", []byte("{\"uid\":1,\"new_column\":2}\n"))
	require.NoError(t, err)
	require.NoError(t, writer.finish(&Manifest{FormatVersion: ManifestFormatVersion, Tables: []*ManifestTable{{Name: "user", Rows: 1, EntryName: "database/user.jsonl"}}}))

	report, err = VerifyBackup(c, path, "")
	assert.ErrorIs(t, err, ErrBackupVerificationFailed)
	assert.Contains(t, report.Problems[0], "new_column")
}

func TestBackup_OverlappingRunIsRejected(t *testing.T) {
	c := core.NewNullContext()
	dir := t.TempDir()

	config := newTestConfig(t, dir, "overlap")
	initializeTestInstance(t, config)

	require.NoError(t, os.MkdirAll(config.BackupPath, 0700))
	lock, err := acquireBackupLock(config.BackupPath)
	require.NoError(t, err)

	_, err = CreateBackup(c, config, NewCreateOptionsFromConfig(config))
	assert.ErrorIs(t, err, ErrBackupAlreadyRunning)

	lock.release()

	_, err = CreateBackup(c, config, NewCreateOptionsFromConfig(config))
	assert.NoError(t, err)
}

func TestBackup_RetentionAppliedAfterSuccess(t *testing.T) {
	c := core.NewNullContext()
	dir := t.TempDir()

	config := newTestConfig(t, dir, "retention")
	initializeTestInstance(t, config)
	require.NoError(t, os.MkdirAll(config.BackupPath, 0700))

	now := time.Now().UTC()

	for i := 1; i <= 4; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(config.BackupPath, GetBackupFileName(now.Add(-time.Duration(i)*time.Hour), false)), []byte("old"), 0600))
	}

	options := NewCreateOptionsFromConfig(config)
	options.RetentionCount = 2
	options.IncludeConfig = true

	// a failed backup must not remove anything
	options.Encrypt = true
	options.Passphrase = ""
	_, err := CreateBackup(c, config, options)
	assert.ErrorIs(t, err, ErrBackupPassphraseRequired)

	backups, err := ListBackupFiles(config.BackupPath)
	require.NoError(t, err)
	assert.Len(t, backups, 4)

	brokenConfig := *config
	brokenConfig.ConfigFilePath = filepath.Join(dir, "missing.ini")
	options.Encrypt = false
	_, err = CreateBackup(c, &brokenConfig, options)
	assert.Error(t, err)

	backups, err = ListBackupFiles(config.BackupPath)
	require.NoError(t, err)
	assert.Len(t, backups, 4)

	status, err := ReadBackupStatus(config.BackupPath)
	require.NoError(t, err)
	assert.False(t, status.LastRunSuccess)
	assert.NotEmpty(t, status.LastError)

	// a successful backup applies the retention policy
	result, err := CreateBackup(c, config, options)
	require.NoError(t, err)
	assert.Len(t, result.DeletedBackups, 3)

	backups, err = ListBackupFiles(config.BackupPath)
	require.NoError(t, err)
	require.Len(t, backups, 2)
	assert.Equal(t, result.FilePath, backups[0].Path)

	entries, err := os.ReadDir(config.BackupPath)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), partialBackupFileSuffix)
		assert.NotContains(t, entry.Name(), ".tmp-backup-")
	}
}
