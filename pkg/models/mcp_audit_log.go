package models

// MCPAuditLogSourceMCP represents the audit log source of changes made through MCP tools
const MCPAuditLogSourceMCP = "mcp"

// MCPAuditLog represents an audit history record of a committed change made through MCP tools stored in database
type MCPAuditLog struct {
	LogId           int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_mcp_audit_log_uid_created_time) INDEX(IDX_mcp_audit_log_uid_entity_type_created_time) NOT NULL"`
	Source          string `xorm:"VARCHAR(16) NOT NULL"`
	ToolName        string `xorm:"VARCHAR(64) NOT NULL"`
	Action          string `xorm:"VARCHAR(32) NOT NULL"`
	EntityType      string `xorm:"VARCHAR(32) INDEX(IDX_mcp_audit_log_uid_entity_type_created_time) NOT NULL"`
	EntityIds       string `xorm:"TEXT"`
	OperationId     string `xorm:"VARCHAR(128)"`
	IdempotencyKey  string `xorm:"VARCHAR(128)"`
	TokenId         string `xorm:"VARCHAR(32)"`
	ClientIp        string `xorm:"VARCHAR(39)"`
	Details         string `xorm:"BLOB"`
	CreatedUnixTime int64  `xorm:"INDEX(IDX_mcp_audit_log_uid_created_time) INDEX(IDX_mcp_audit_log_uid_entity_type_created_time)"`
}

// MCPAuditLogFilter represents the filter of querying MCP audit logs
type MCPAuditLogFilter struct {
	MinCreatedUnixTime int64
	MaxCreatedUnixTime int64
	ToolName           string
	Action             string
	EntityType         string
	EntityId           int64
	OperationId        string
}
