package models

// MCPIdempotencyKeyMaxLength is the maximum length (in characters) of an idempotency key of MCP write tools
const MCPIdempotencyKeyMaxLength = 128

// MCPIdempotencyRecord represents the stored result of an MCP write tool call with an idempotency key.
// A record is created as pending (not completed) before the write is executed, and completed with the response after the write succeeds,
// so a retry with the same key and the same request returns the stored response instead of executing the write again
type MCPIdempotencyRecord struct {
	RecordId        int64  `xorm:"PK"`
	Uid             int64  `xorm:"UNIQUE(UQE_mcp_idempotency_record_uid_key) INDEX(IDX_mcp_idempotency_record_uid_created_time) NOT NULL"`
	IdempotencyKey  string `xorm:"VARCHAR(128) UNIQUE(UQE_mcp_idempotency_record_uid_key) NOT NULL"`
	ToolName        string `xorm:"VARCHAR(64) NOT NULL"`
	RequestHash     string `xorm:"VARCHAR(64) NOT NULL"`
	Completed       bool   `xorm:"NOT NULL"`
	Response        string `xorm:"MEDIUMBLOB"`
	CreatedUnixTime int64  `xorm:"INDEX(IDX_mcp_idempotency_record_uid_created_time)"`
	UpdatedUnixTime int64
}
