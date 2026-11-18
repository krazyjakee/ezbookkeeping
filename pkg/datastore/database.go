package datastore

import (
	"fmt"
	"strings"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// Database represents a database instance
type Database interface {
	// NewSession starts a new session with the specified context
	NewSession(c core.Context) DatabaseSession

	// DoTransaction runs a new database transaction
	DoTransaction(c core.Context, fn func(sess DatabaseSession) error) error

	// SetSavePoint sets a save point in the current transaction for Postgres
	SetSavePoint(sess DatabaseSession, savePointName string) error

	// RollbackToSavePoint rolls back to the specified save point in the current transaction for Postgres
	RollbackToSavePoint(sess DatabaseSession, savePointName string) error

	// SyncStructs updates database structs by database models
	SyncStructs(beans ...any) error
}

// XOrmDatabase represents a database instance using xorm
type XOrmDatabase struct {
	databaseType string
	engineGroup  *xorm.EngineGroup
}

// NewSession starts a new session with the specified context
func (db *XOrmDatabase) NewSession(c core.Context) DatabaseSession {
	return &XOrmSession{
		Session: db.engineGroup.Context(NewXOrmContextAdapter(c)),
	}
}

// DoTransaction runs a new database transaction
func (db *XOrmDatabase) DoTransaction(c core.Context, fn func(sess DatabaseSession) error) (err error) {
	sess := db.engineGroup.NewSession()

	if c != nil {
		sess.Context(NewXOrmContextAdapter(c))
	}

	defer sess.Close()

	if err = sess.Begin(); err != nil {
		return err
	}

	xormSession := &XOrmSession{
		Session: sess,
	}

	if err = fn(xormSession); err != nil {
		_ = sess.Rollback()
		return err
	}

	if err = sess.Commit(); err != nil {
		return err
	}

	return nil
}

// SetSavePoint sets a save point in the current transaction for Postgres
func (db *XOrmDatabase) SetSavePoint(sess DatabaseSession, savePointName string) error {
	if db.databaseType == settings.PostgresDbType {
		_, err := sess.Exec("SAVEPOINT " + savePointName)
		return err
	}

	return nil
}

// RollbackToSavePoint rolls back to the specified save point in the current transaction for Postgres
func (db *XOrmDatabase) RollbackToSavePoint(sess DatabaseSession, savePointName string) error {
	if db.databaseType == settings.PostgresDbType {
		_, err := sess.Exec("ROLLBACK TO SAVEPOINT " + savePointName)
		return err
	}

	return nil
}

// SyncStructs updates database structs by database models
func (db *XOrmDatabase) SyncStructs(beans ...any) error {
	err := db.engineGroup.Sync2(beans...)

	if err != nil {
		return err
	}

	return db.widenVarcharColumns(beans...)
}

// widenVarcharColumns enlarges VARCHAR columns which are shorter in the database than in the models.
// xorm treats VARCHAR(m) and VARCHAR(n) as the same type and never widens them on MySQL or Postgres,
// so without this an increased model length is only enforced by validation and writes fail in the database.
// Columns are never shrunk, and SQLite does not enforce VARCHAR lengths, so it is skipped.
func (db *XOrmDatabase) widenVarcharColumns(beans ...any) error {
	if db.databaseType != settings.MySqlDbType && db.databaseType != settings.PostgresDbType {
		return nil
	}

	tables, err := db.engineGroup.DBMetas()

	if err != nil {
		return err
	}

	existingTables := make(map[string]*schemas.Table, len(tables))

	for _, table := range tables {
		existingTables[strings.ToLower(table.Name)] = table
	}

	for _, bean := range beans {
		table, err := db.engineGroup.TableInfo(bean)

		if err != nil {
			return err
		}

		tableName := db.engineGroup.TableName(bean)
		existingTable, exists := existingTables[strings.ToLower(tableName)]

		if !exists {
			continue
		}

		for _, column := range table.Columns() {
			if !isVarcharColumn(column) {
				continue
			}

			existingColumn := existingTable.GetColumn(column.Name)

			if existingColumn == nil || !isVarcharColumn(existingColumn) || existingColumn.Length >= column.Length {
				continue
			}

			_, err = db.engineGroup.Exec(db.engineGroup.Dialect().ModifyColumnSQL(db.engineGroup.TableName(bean, true), column))

			if err != nil {
				return fmt.Errorf("failed to widen column %s.%s from varchar(%d) to varchar(%d): %w", tableName, column.Name, existingColumn.Length, column.Length, err)
			}
		}
	}

	return nil
}

func isVarcharColumn(column *schemas.Column) bool {
	return column.Length > 0 && (strings.EqualFold(column.SQLType.Name, schemas.Varchar) || strings.EqualFold(column.SQLType.Name, schemas.NVarchar))
}
