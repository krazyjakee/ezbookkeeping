package mcp

import (
	"encoding/json/v2"
	"reflect"
	"strconv"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Pagination limits of querying audit logs
const (
	mcpAuditLogDefaultCount = 50
	mcpAuditLogMaxCount     = 200
)

// MCPQueryAuditLogRequest represents all parameters of the query audit log request
type MCPQueryAuditLogRequest struct {
	StartTime   string `json:"start_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Only return changes made at or after this time in RFC 3339 format (optional)"`
	EndTime     string `json:"end_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Only return changes made at or before this time in RFC 3339 format (optional)"`
	ToolName    string `json:"tool_name,omitempty" jsonschema_description:"Only return changes made by this MCP tool, e.g. add_transaction (optional)"`
	Action      string `json:"action,omitempty" jsonschema:"enum=create,enum=update,enum=delete,enum=hide,enum=reorder,enum=upload,enum=detach,enum=import,enum=reconcile" jsonschema_description:"Only return changes of this action (optional)"`
	EntityType  string `json:"entity_type,omitempty" jsonschema:"enum=transaction,enum=account,enum=transaction_category,enum=transaction_tag,enum=transaction_tag_group,enum=transaction_template,enum=transaction_attachment" jsonschema_description:"Only return changes of this entity type (optional)"`
	EntityId    string `json:"entity_id,omitempty" jsonschema_description:"Only return changes which affected the entity with this ID (optional)"`
	OperationId string `json:"filter_operation_id,omitempty" jsonschema_description:"Only return changes made with this client provided operation_id (optional)"`
	Count       int32  `json:"count,omitempty" jsonschema:"default=50" jsonschema_description:"Maximum number of results to return (optional, default: 50, maximum: 200)"`
	Page        int32  `json:"page,omitempty" jsonschema:"default=1" jsonschema_description:"Page number for pagination (optional, default: 1)"`
}

// MCPAuditLogInfo represents an audit history record of a change made through MCP tools
type MCPAuditLogInfo struct {
	Id             string         `json:"id" jsonschema_description:"Audit log ID"`
	Time           string         `json:"time" jsonschema_description:"Time of the change in RFC 3339 format (UTC)"`
	Source         string         `json:"source" jsonschema_description:"Source of the change (always mcp)"`
	ToolName       string         `json:"tool_name" jsonschema_description:"MCP tool which made the change"`
	Action         string         `json:"action" jsonschema_description:"Action of the change (create, update, delete, hide, reorder, upload, detach, import, reconcile)"`
	EntityType     string         `json:"entity_type" jsonschema_description:"Entity type of the change"`
	EntityIds      []string       `json:"entity_ids" jsonschema_description:"IDs of the affected entities"`
	OperationId    string         `json:"operation_id,omitempty" jsonschema_description:"Client provided operation / reference ID"`
	IdempotencyKey string         `json:"idempotency_key,omitempty" jsonschema_description:"Idempotency key of the request"`
	TokenId        string         `json:"token_id,omitempty" jsonschema_description:"ID of the token used for the change"`
	ClientIp       string         `json:"client_ip,omitempty" jsonschema_description:"Client IP address"`
	Details        map[string]any `json:"details,omitempty" jsonschema_description:"Small summary of the result (status, changed fields, counts)"`
}

// MCPQueryAuditLogResponse represents the response structure for querying audit logs
type MCPQueryAuditLogResponse struct {
	TotalCount  int64              `json:"total_count" jsonschema_description:"Total number of audit logs matching the query"`
	CurrentPage int32              `json:"current_page" jsonschema_description:"Current page number"`
	TotalPage   int32              `json:"total_page" jsonschema_description:"Total number of pages"`
	Order       string             `json:"order" jsonschema_description:"Order of the results (always time_desc)"`
	Logs        []*MCPAuditLogInfo `json:"logs" jsonschema_description:"Audit logs matching the query"`
}

type mcpQueryAuditLogToolHandler struct{}

var MCPQueryAuditLogToolHandler = &mcpQueryAuditLogToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpQueryAuditLogToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_QUERY_AUDIT_LOG
}

// Name returns the name of the MCP tool
func (h *mcpQueryAuditLogToolHandler) Name() string {
	return "query_audit_log"
}

// Description returns the description of the MCP tool
func (h *mcpQueryAuditLogToolHandler) Description() string {
	return "Query the audit history of changes committed through MCP write tools (dry runs and reads are not recorded): tool, action, entity type and affected IDs, client operation_id, idempotency key, token ID, client IP and time. Supports filters by time range, tool, action, entity and operation ID (filter_operation_id), with pagination (newest first)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpQueryAuditLogToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpQueryAuditLogToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpQueryAuditLogToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpQueryAuditLogToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryAuditLogRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpQueryAuditLogToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryAuditLogResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpQueryAuditLogToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPQueryAuditLogRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	filter := &models.MCPAuditLogFilter{
		ToolName:    strings.TrimSpace(request.ToolName),
		Action:      strings.TrimSpace(request.Action),
		EntityType:  strings.TrimSpace(request.EntityType),
		OperationId: request.OperationId,
	}

	if startTime, err := parseMCPTime("start_time", request.StartTime, false); err != nil {
		return nil, nil, err
	} else if startTime != nil {
		filter.MinCreatedUnixTime = startTime.Unix()
	}

	if endTime, err := parseMCPTime("end_time", request.EndTime, false); err != nil {
		return nil, nil, err
	} else if endTime != nil {
		filter.MaxCreatedUnixTime = endTime.Unix()
	}

	if filter.EntityId, err = parseMCPOptionalId("entity_id", request.EntityId); err != nil {
		return nil, nil, err
	}

	if request.Count <= 0 {
		request.Count = mcpAuditLogDefaultCount
	} else if request.Count > mcpAuditLogMaxCount {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeOutOfRange, "count", "\"count\" must not exceed "+strconv.Itoa(mcpAuditLogMaxCount))
	}

	if request.Page <= 0 {
		request.Page = 1
	}

	uid := user.Uid
	auditLogs, totalCount, err := mcpGetAuditLogService(services).GetAuditLogs(c, uid, filter, request.Page, request.Count)

	if err != nil {
		log.Errorf(c, "[query_audit_log_tool_handler.Handle] failed to get audit logs for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	response := &MCPQueryAuditLogResponse{
		TotalCount:  totalCount,
		CurrentPage: request.Page,
		TotalPage:   int32((totalCount + int64(request.Count) - 1) / int64(request.Count)),
		Order:       "time_desc",
		Logs:        make([]*MCPAuditLogInfo, 0, len(auditLogs)),
	}

	for i := 0; i < len(auditLogs); i++ {
		response.Logs = append(response.Logs, mcpAuditBuildLogInfo(auditLogs[i]))
	}

	return newMCPJSONTextResult(response)
}

// mcpAuditBuildLogInfo returns the audit log information of the audit log model
func mcpAuditBuildLogInfo(auditLog *models.MCPAuditLog) *MCPAuditLogInfo {
	info := &MCPAuditLogInfo{
		Id:             utils.Int64ToString(auditLog.LogId),
		Time:           formatMCPUnixTime(auditLog.CreatedUnixTime),
		Source:         auditLog.Source,
		ToolName:       auditLog.ToolName,
		Action:         auditLog.Action,
		EntityType:     auditLog.EntityType,
		EntityIds:      mcpAuditParseEntityIds(auditLog.EntityIds),
		OperationId:    auditLog.OperationId,
		IdempotencyKey: auditLog.IdempotencyKey,
		TokenId:        auditLog.TokenId,
		ClientIp:       auditLog.ClientIp,
	}

	if auditLog.Details != "" {
		var details map[string]any

		if err := json.Unmarshal([]byte(auditLog.Details), &details); err == nil && len(details) > 0 {
			info.Details = details
		}
	}

	return info
}
