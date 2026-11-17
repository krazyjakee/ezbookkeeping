package datastore

import (
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// TableRegistration represents a database table which is managed by ezBookkeeping
type TableRegistration struct {
	// Description is the human-readable table description used in logs
	Description string

	// NewBean returns a new empty model instance of the table
	NewBean func() any

	// Store returns the data store which the table belongs to
	Store func() *DataStore
}

func userStore() *DataStore {
	return Container.UserStore
}

func tokenStore() *DataStore {
	return Container.TokenStore
}

func userDataStore() *DataStore {
	return Container.UserDataStore
}

// allTables is the single source of truth of all the database tables.
// Every new table MUST be registered here, so that it is created / updated by "database update"
// and included in full-instance backups and restores.
var allTables = []*TableRegistration{
	{Description: "user", NewBean: func() any { return new(models.User) }, Store: userStore},
	{Description: "two-factor", NewBean: func() any { return new(models.TwoFactor) }, Store: userStore},
	{Description: "two-factor recovery code", NewBean: func() any { return new(models.TwoFactorRecoveryCode) }, Store: userStore},
	{Description: "token record", NewBean: func() any { return new(models.TokenRecord) }, Store: tokenStore},
	{Description: "account", NewBean: func() any { return new(models.Account) }, Store: userDataStore},
	{Description: "transaction", NewBean: func() any { return new(models.Transaction) }, Store: userDataStore},
	{Description: "transaction category", NewBean: func() any { return new(models.TransactionCategory) }, Store: userDataStore},
	{Description: "transaction tag group", NewBean: func() any { return new(models.TransactionTagGroup) }, Store: userDataStore},
	{Description: "transaction tag", NewBean: func() any { return new(models.TransactionTag) }, Store: userDataStore},
	{Description: "transaction tag index", NewBean: func() any { return new(models.TransactionTagIndex) }, Store: userDataStore},
	{Description: "transaction template", NewBean: func() any { return new(models.TransactionTemplate) }, Store: userDataStore},
	{Description: "transaction picture", NewBean: func() any { return new(models.TransactionPictureInfo) }, Store: userDataStore},
	{Description: "user custom icon", NewBean: func() any { return new(models.UserCustomIcon) }, Store: userDataStore},
	{Description: "user import preset", NewBean: func() any { return new(models.ImportPreset) }, Store: userDataStore},
	{Description: "user custom exchange rate", NewBean: func() any { return new(models.UserCustomExchangeRate) }, Store: userDataStore},
	{Description: "user application cloud settings", NewBean: func() any { return new(models.UserApplicationCloudSetting) }, Store: userDataStore},
	{Description: "user external auth", NewBean: func() any { return new(models.UserExternalAuth) }, Store: userDataStore},
	{Description: "insights explorer", NewBean: func() any { return new(models.InsightsExplorer) }, Store: userDataStore},
	{Description: "transaction reference", NewBean: func() any { return new(models.TransactionReference) }, Store: userDataStore},
	{Description: "mcp idempotency record", NewBean: func() any { return new(models.MCPIdempotencyRecord) }, Store: userDataStore},
	{Description: "mcp audit log", NewBean: func() any { return new(models.MCPAuditLog) }, Store: userDataStore},
}

// GetAllTables returns all the registered database tables in creation order
func GetAllTables() []*TableRegistration {
	tables := make([]*TableRegistration, len(allTables))
	copy(tables, allTables)
	return tables
}
