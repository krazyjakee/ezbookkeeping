package datastore

import (
	"xorm.io/xorm"
)

// RawDatabaseAccessor represents a database which exposes the underlying xorm engine.
// It is only intended for maintenance tooling (e.g. full-instance backup and restore)
// which needs database-level operations such as snapshot transactions.
type RawDatabaseAccessor interface {
	// DatabaseType returns the database type (e.g. sqlite3, mysql or postgres)
	DatabaseType() string

	// MasterEngine returns the underlying master xorm engine
	MasterEngine() *xorm.Engine
}

// DatabaseType returns the database type (e.g. sqlite3, mysql or postgres)
func (db *XOrmDatabase) DatabaseType() string {
	return db.databaseType
}

// MasterEngine returns the underlying master xorm engine
func (db *XOrmDatabase) MasterEngine() *xorm.Engine {
	return db.engineGroup.Master()
}
