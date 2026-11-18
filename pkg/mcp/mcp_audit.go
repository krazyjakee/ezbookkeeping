package mcp

import (
	"encoding/json/v2"
	"strings"
	"unicode/utf8"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Limits of the audit logs of MCP write tools
const (
	mcpAuditMaxEntityIdsCount = 500
	mcpAuditMaxDetailsSize    = 4096
	mcpAuditMaxClientIpLength = 39
	mcpAuditMaxTokenIdLength  = 32
)

// mcpAuditToolActionAndEntityType represents the action and entity type of an MCP write tool recorded in audit logs
type mcpAuditToolActionAndEntityType struct {
	action     string
	entityType string
}

// mcpAuditToolExceptions contains the actions and entity types of MCP write tools which cannot be derived from the tool name
var mcpAuditToolExceptions = map[string]*mcpAuditToolActionAndEntityType{
	"set_account_reconciled_time":   {action: "reconcile", entityType: "account"},
	"import_transactions":           {action: "import", entityType: "transaction"},
	"add_transaction_from_document": {action: "create", entityType: "transaction"},
}

// mcpAuditToolNameVerb represents a verb (tool name prefix) of MCP write tools and its audit action
type mcpAuditToolNameVerb struct {
	prefix string
	action string
}

// mcpAuditToolNameVerbs contains the verbs (tool name prefixes) of MCP write tools and their audit actions, longer prefixes first
var mcpAuditToolNameVerbs = []*mcpAuditToolNameVerb{
	{prefix: "batch_create_", action: "create"},
	{prefix: "add_", action: "create"},
	{prefix: "create_", action: "create"},
	{prefix: "update_", action: "update"},
	{prefix: "delete_", action: "delete"},
	{prefix: "hide_", action: "hide"},
	{prefix: "reorder_", action: "reorder"},
	{prefix: "upload_", action: "upload"},
	{prefix: "detach_", action: "detach"},
	{prefix: "import_", action: "import"},
	{prefix: "set_", action: "update"},
}

// mcpAuditDetailFieldNames contains the result fields which are copied to the details of audit logs
var mcpAuditDetailFieldNames = []string{"status", "changed_fields", "created_count", "updated_count", "existing_count", "skipped_count", "failed_count", "permanent", "attached", "verified", "references_error"}

// mcpAuditGetEntityTypeAndAction returns the entity type (e.g. transaction, account, transaction_category) and the action (e.g. create, update, delete) of the MCP write tool
func mcpAuditGetEntityTypeAndAction(toolName string) (string, string) {
	if exception, exists := mcpAuditToolExceptions[toolName]; exists {
		return exception.entityType, exception.action
	}

	action := "write"
	entity := toolName

	for i := 0; i < len(mcpAuditToolNameVerbs); i++ {
		prefix := mcpAuditToolNameVerbs[i].prefix

		if strings.HasPrefix(toolName, prefix) {
			action = mcpAuditToolNameVerbs[i].action
			entity = toolName[len(prefix):]
			break
		}
	}

	if strings.HasSuffix(entity, "ies") {
		entity = entity[:len(entity)-3] + "y"
	} else if strings.HasSuffix(entity, "s") {
		entity = entity[:len(entity)-1]
	}

	return entity, action
}

// mcpAuditGetEntityIdKeyStem returns the stem of the id field names of the entity type in tool results (e.g. "category" for transaction_category)
func mcpAuditGetEntityIdKeyStem(entityType string) string {
	if entityType == "transaction" {
		return entityType
	}

	return strings.TrimPrefix(entityType, "transaction_")
}

// mcpAuditExtractEntityIds returns the ids of the affected entities in the result of an MCP write tool:
// "id", "<entity>_id" and "<entity>_ids" members of the result, the "id" of objects in the result, and "id" / "<entity>_id" of items of arrays in the result
func mcpAuditExtractEntityIds(response map[string]any, entityType string) []string {
	ids := make([]string, 0)

	if response == nil {
		return ids
	}

	stem := mcpAuditGetEntityIdKeyStem(entityType)
	idExists := make(map[string]bool)

	addId := func(value any) {
		id, ok := value.(string)

		if !ok || id == "" || idExists[id] || len(ids) >= mcpAuditMaxEntityIdsCount {
			return
		}

		if _, err := utils.StringToInt64(id); err != nil {
			return
		}

		idExists[id] = true
		ids = append(ids, id)
	}

	isEntityIdKey := func(key string) bool {
		if stem == "" {
			return key == "id" || strings.HasSuffix(key, "_id")
		}

		return key == "id" || key == stem+"_id" || strings.HasSuffix(key, "_"+stem+"_id")
	}

	isEntityIdsKey := func(key string) bool {
		if stem == "" {
			return strings.HasSuffix(key, "_ids")
		}

		return key == stem+"_ids" || strings.HasSuffix(key, "_"+stem+"_ids")
	}

	for key, value := range response {
		switch typedValue := value.(type) {
		case string:
			if isEntityIdKey(key) {
				addId(typedValue)
			}
		case []any:
			for i := 0; i < len(typedValue); i++ {
				if isEntityIdsKey(key) {
					addId(typedValue[i])
				} else if item, ok := typedValue[i].(map[string]any); ok {
					for itemKey, itemValue := range item {
						if isEntityIdKey(itemKey) {
							addId(itemValue)
						}
					}
				}
			}
		case map[string]any:
			addId(typedValue["id"])
		}
	}

	return ids
}

// mcpAuditFormatEntityIds returns the textual entity ids stored in audit logs (",id1,id2,", so that one id can be matched exactly)
func mcpAuditFormatEntityIds(ids []string) string {
	if len(ids) < 1 {
		return ""
	}

	return "," + strings.Join(ids, ",") + ","
}

// mcpAuditParseEntityIds returns the entity ids of the textual entity ids stored in audit logs
func mcpAuditParseEntityIds(entityIds string) []string {
	ids := make([]string, 0)

	for _, id := range strings.Split(entityIds, ",") {
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids
}

// mcpAuditBuildDetails returns the small JSON details of the audit log which contain the result status and counts
func mcpAuditBuildDetails(response map[string]any, entityIdsTruncated bool) string {
	details := make(map[string]any)

	for i := 0; i < len(mcpAuditDetailFieldNames); i++ {
		if value, exists := response[mcpAuditDetailFieldNames[i]]; exists && value != nil {
			details[mcpAuditDetailFieldNames[i]] = value
		}
	}

	if entityIdsTruncated {
		details["entity_ids_truncated"] = true
	}

	content, err := json.Marshal(details)

	if err != nil {
		return ""
	}

	if len(content) > mcpAuditMaxDetailsSize {
		delete(details, "changed_fields")
		content, err = json.Marshal(details)

		if err != nil || len(content) > mcpAuditMaxDetailsSize {
			return ""
		}
	}

	return string(content)
}

// mcpAuditTruncate returns the text truncated to the maximum length in characters
func mcpAuditTruncate(text string, maxLength int) string {
	if utf8.RuneCountInString(text) <= maxLength {
		return text
	}

	return string([]rune(text)[:maxLength])
}

// mcpAuditBuildLog returns the audit log model of the committed write tool call
func mcpAuditBuildLog(c *core.WebContext, w *mcpWriteToolCall, entityType string, action string, entityIds []string, response map[string]any) *models.MCPAuditLog {
	auditLog := &models.MCPAuditLog{
		Uid:            w.uid,
		Source:         models.MCPAuditLogSourceMCP,
		ToolName:       w.toolName,
		Action:         action,
		EntityType:     entityType,
		EntityIds:      mcpAuditFormatEntityIds(entityIds),
		OperationId:    w.operationId,
		IdempotencyKey: w.idempotencyKey,
		Details:        mcpAuditBuildDetails(response, len(entityIds) >= mcpAuditMaxEntityIdsCount),
	}

	if c != nil {
		auditLog.ClientIp = mcpAuditTruncate(c.ClientIP(), mcpAuditMaxClientIpLength)

		if claims := c.GetTokenClaims(); claims != nil {
			auditLog.TokenId = mcpAuditTruncate(claims.UserTokenId, mcpAuditMaxTokenIdLength)
		}
	}

	return auditLog
}
