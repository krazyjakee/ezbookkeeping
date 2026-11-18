package datastore

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

type widenVarcharTestRowV1 struct {
	Id      int64  `xorm:"PK"`
	Comment string `xorm:"VARCHAR(255) NOT NULL"`
}

func (widenVarcharTestRowV1) TableName() string { return "widen_varchar_test" }

type widenVarcharTestRowV2 struct {
	Id      int64  `xorm:"PK"`
	Comment string `xorm:"VARCHAR(1024) NOT NULL"`
}

func (widenVarcharTestRowV2) TableName() string { return "widen_varchar_test" }

type widenVarcharTestRowShorter struct {
	Id      int64  `xorm:"PK"`
	Comment string `xorm:"VARCHAR(100) NOT NULL"`
}

func (widenVarcharTestRowShorter) TableName() string { return "widen_varchar_test" }

func widenVarcharTestColumnLength(t *testing.T, db *XOrmDatabase) int64 {
	tables, err := db.engineGroup.DBMetas()
	assert.Nil(t, err)

	for _, table := range tables {
		if table.Name == "widen_varchar_test" {
			return table.GetColumn("comment").Length
		}
	}

	t.Fatal("table widen_varchar_test not found")
	return 0
}

func newWidenVarcharTestDatabase(t *testing.T, databaseType string, driverName string, dsn string) *XOrmDatabase {
	engineGroup, err := xorm.NewEngineGroup(driverName, []string{dsn})
	assert.Nil(t, err)
	t.Cleanup(func() {
		_ = engineGroup.DropTables("widen_varchar_test")
		_ = engineGroup.Close()
	})
	_ = engineGroup.DropTables("widen_varchar_test")

	return &XOrmDatabase{databaseType: databaseType, engineGroup: engineGroup}
}

// TestSyncStructs_WidensVarcharColumnOnPostgres needs a disposable Postgres database, e.g.
// EBK_TEST_POSTGRES_DSN="postgres://user:pass@127.0.0.1:5432/db?sslmode=disable"
func TestSyncStructs_WidensVarcharColumnOnPostgres(t *testing.T) {
	dsn := os.Getenv("EBK_TEST_POSTGRES_DSN")

	if dsn == "" {
		t.Skip("EBK_TEST_POSTGRES_DSN is not set")
	}

	db := newWidenVarcharTestDatabase(t, settings.PostgresDbType, "postgres", dsn)

	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowV1)))
	assert.Equal(t, int64(255), widenVarcharTestColumnLength(t, db))

	_, err := db.engineGroup.Insert(&widenVarcharTestRowV1{Id: 1, Comment: "existing"})
	assert.Nil(t, err)

	_, err = db.engineGroup.Insert(&widenVarcharTestRowV2{Id: 2, Comment: strings.Repeat("a", 500)})
	assert.NotNil(t, err, "a 500 character value must not fit before widening")

	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowV2)))
	assert.Equal(t, int64(1024), widenVarcharTestColumnLength(t, db))

	_, err = db.engineGroup.Insert(&widenVarcharTestRowV2{Id: 2, Comment: strings.Repeat("a", 1024)})
	assert.Nil(t, err)

	existing := &widenVarcharTestRowV2{}
	has, err := db.engineGroup.ID(1).Get(existing)
	assert.Nil(t, err)
	assert.True(t, has)
	assert.Equal(t, "existing", existing.Comment)

	// Running the sync again is a no-op, and a shorter model never shrinks the column
	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowV2)))
	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowShorter)))
	assert.Equal(t, int64(1024), widenVarcharTestColumnLength(t, db))
}

func TestSyncStructs_SkipsWideningOnSqlite(t *testing.T) {
	db := newWidenVarcharTestDatabase(t, settings.Sqlite3DbType, "sqlite3", "file:"+t.TempDir()+"/widen.db")

	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowV1)))
	assert.Nil(t, db.SyncStructs(new(widenVarcharTestRowV2)))

	_, err := db.engineGroup.Insert(&widenVarcharTestRowV2{Id: 1, Comment: strings.Repeat("a", 1024)})
	assert.Nil(t, err)
}
