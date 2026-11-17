package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// Names of the control arguments which are accepted by all MCP write tools
const (
	mcpIdempotencyKeyArgumentName = "idempotency_key"
	mcpOperationIdArgumentName    = "operation_id"
	mcpDryRunArgumentName         = "dry_run"
	mcpIdempotentReplayFieldName  = "idempotent_replay"
)

// Idempotency settings of MCP write tools
const (
	// mcpIdempotencyRecordRetention is how long an idempotency key (and its stored result) is kept, expired records are deleted lazily
	mcpIdempotencyRecordRetention = 30 * 24 * time.Hour

	// mcpIdempotencyPendingTimeout is how long a request with an idempotency key is considered to be still in progress
	mcpIdempotencyPendingTimeout = 10 * time.Minute

	// mcpIdempotencyMaxStoredResponseSize is the maximum size (in bytes) of the stored result, larger results are stored as a summary
	mcpIdempotencyMaxStoredResponseSize = 1024 * 1024

	// mcpOperationIdMaxLength is the maximum length (in characters) of the client operation / reference id
	mcpOperationIdMaxLength = 128
)

// mcpWriteControlArguments represents the control arguments which are accepted by all MCP write tools
type mcpWriteControlArguments struct {
	IdempotencyKey string
	OperationId    string
	DryRun         *bool
}

// mcpDefaultDryRunToolHandler is implemented by MCP tools whose dry_run argument defaults to true
type mcpDefaultDryRunToolHandler interface {
	DefaultDryRun() bool
}

// mcpIdempotencyStore represents the storage of idempotency records
type mcpIdempotencyStore interface {
	GetRecordByKey(c core.Context, uid int64, idempotencyKey string) (*models.MCPIdempotencyRecord, error)
	TryCreatePendingRecord(c core.Context, record *models.MCPIdempotencyRecord) (bool, error)
	CompleteRecord(c core.Context, uid int64, recordId int64, response string) error
	DeleteRecord(c core.Context, uid int64, recordId int64) error
	DeleteExpiredRecords(c core.Context, uid int64, createdBeforeUnixTime int64) error
}

// mcpWriteToolCall represents the state of a call of an MCP write tool, which is used to apply idempotency keys and to record audit logs
type mcpWriteToolCall struct {
	toolName       string
	uid            int64
	dryRun         bool
	idempotencyKey string
	operationId    string
	store          mcpIdempotencyStore
	pendingRecord  *models.MCPIdempotencyRecord
	replayResponse []byte
}

// mcpToolInputSupportsDryRun returns whether the input type of the MCP tool has the dry_run argument
func mcpToolInputSupportsDryRun(inputType reflect.Type) bool {
	if inputType == nil {
		return false
	}

	for inputType.Kind() == reflect.Pointer {
		inputType = inputType.Elem()
	}

	if inputType.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < inputType.NumField(); i++ {
		name := strings.Split(inputType.Field(i).Tag.Get("json"), ",")[0]

		if name == mcpDryRunArgumentName {
			return true
		}
	}

	return false
}

// mcpParseWriteControlArguments parses the control arguments (idempotency_key, operation_id and dry_run) of an MCP write tool call.
// Arguments which are not a JSON object are ignored here (the tool reports them)
func mcpParseWriteControlArguments(arguments jsontext.Value) (*mcpWriteControlArguments, error) {
	controlArguments := &mcpWriteControlArguments{}

	if len(arguments) == 0 {
		return controlArguments, nil
	}

	var fields map[string]jsontext.Value

	if err := json.Unmarshal(arguments, &fields); err != nil {
		return controlArguments, nil
	}

	if value, exists := fields[mcpIdempotencyKeyArgumentName]; exists && !mcpIsJSONNull(value) {
		if err := json.Unmarshal(value, &controlArguments.IdempotencyKey); err != nil {
			return nil, NewMCPInvalidFieldError(mcpIdempotencyKeyArgumentName, "", "\""+mcpIdempotencyKeyArgumentName+"\" must be a string")
		}

		if err := mcpCheckClientProvidedId(mcpIdempotencyKeyArgumentName, controlArguments.IdempotencyKey, models.MCPIdempotencyKeyMaxLength); err != nil {
			return nil, err
		}
	}

	if value, exists := fields[mcpOperationIdArgumentName]; exists && !mcpIsJSONNull(value) {
		if err := json.Unmarshal(value, &controlArguments.OperationId); err != nil {
			return nil, NewMCPInvalidFieldError(mcpOperationIdArgumentName, "", "\""+mcpOperationIdArgumentName+"\" must be a string")
		}

		if err := mcpCheckClientProvidedId(mcpOperationIdArgumentName, controlArguments.OperationId, mcpOperationIdMaxLength); err != nil {
			return nil, err
		}
	}

	if value, exists := fields[mcpDryRunArgumentName]; exists && !mcpIsJSONNull(value) {
		var dryRun bool

		if err := json.Unmarshal(value, &dryRun); err == nil {
			controlArguments.DryRun = &dryRun
		}
	}

	return controlArguments, nil
}

// mcpCheckClientProvidedId checks the client provided id (idempotency key or operation id) is not blank, has no control characters and does not exceed the maximum length
func mcpCheckClientProvidedId(field string, value string, maxLength int) error {
	if strings.TrimSpace(value) == "" {
		return NewMCPInvalidFieldError(field, value, "\""+field+"\" must not be blank")
	}

	if length := utf8.RuneCountInString(value); length > maxLength {
		return NewMCPFieldTooLongError(field, maxLength, length)
	}

	for _, r := range value {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return NewMCPInvalidFieldError(field, "", "\""+field+"\" must not contain control characters")
		}
	}

	return nil
}

// mcpIsJSONNull returns whether the json value is null
func mcpIsJSONNull(value jsontext.Value) bool {
	return strings.TrimSpace(string(value)) == "null"
}

// mcpComputeWriteRequestHash returns the hex encoded SHA-256 hash of the tool name and the canonical form (RFC 8785) of the arguments,
// the control arguments (idempotency_key, operation_id and dry_run) are excluded, so the hash does not depend on the member order or whitespace
func mcpComputeWriteRequestHash(toolName string, arguments jsontext.Value) (string, error) {
	fields := make(map[string]jsontext.Value)

	if len(arguments) > 0 && !mcpIsJSONNull(arguments) {
		if err := json.Unmarshal(arguments, &fields); err != nil {
			return "", NewMCPToolError(MCPToolErrorCodeInvalidArguments, "", "arguments are invalid: "+err.Error())
		}
	}

	delete(fields, mcpIdempotencyKeyArgumentName)
	delete(fields, mcpOperationIdArgumentName)
	delete(fields, mcpDryRunArgumentName)

	content, err := json.Marshal(fields)

	if err != nil {
		return "", err
	}

	canonicalContent := jsontext.Value(content)

	if err := canonicalContent.Canonicalize(); err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(toolName + "\n" + string(canonicalContent)))
	return hex.EncodeToString(hash[:]), nil
}

// mcpIdempotencyBegin checks the idempotency key before executing a write.
// If a completed record with the same key and the same request exists, the stored result (marked with idempotent_replay) is returned and the write must not be executed.
// If the key was used for a different request, or a request with the same key is still in progress, an error is returned.
// Otherwise a pending record is created and returned, which must be completed after the write succeeds or deleted if the write fails
func mcpIdempotencyBegin(c core.Context, store mcpIdempotencyStore, uid int64, idempotencyKey string, toolName string, requestHash string, now time.Time) (*models.MCPIdempotencyRecord, []byte, error) {
	expiredBeforeUnixTime := now.Add(-mcpIdempotencyRecordRetention).Unix()

	for attempt := 0; attempt < 2; attempt++ {
		record, err := store.GetRecordByKey(c, uid, idempotencyKey)

		if err != nil {
			return nil, nil, err
		}

		if record != nil && record.CreatedUnixTime < expiredBeforeUnixTime {
			if err := store.DeleteRecord(c, uid, record.RecordId); err != nil {
				return nil, nil, err
			}

			record = nil
		}

		if record != nil {
			replayResponse, err := mcpIdempotencyCheckExistingRecord(record, idempotencyKey, toolName, requestHash, now)
			return nil, replayResponse, err
		}

		pendingRecord := &models.MCPIdempotencyRecord{
			Uid:            uid,
			IdempotencyKey: idempotencyKey,
			ToolName:       toolName,
			RequestHash:    requestHash,
		}

		created, err := store.TryCreatePendingRecord(c, pendingRecord)

		if err != nil {
			return nil, nil, err
		}

		if created {
			if err := store.DeleteExpiredRecords(c, uid, expiredBeforeUnixTime); err != nil {
				log.Warnf(c, "[mcp_tool_write_guard.mcpIdempotencyBegin] failed to delete expired idempotency records for user \"uid:%d\", because %s", uid, err.Error())
			}

			return pendingRecord, nil, nil
		}
	}

	return nil, nil, NewMCPToolError(MCPToolErrorCodeIdempotencyInProgress, mcpIdempotencyKeyArgumentName, "a request with the same idempotency key is in progress, retry later")
}

// mcpIdempotencyCheckExistingRecord returns the stored result (marked with idempotent_replay) if the existing record is completed and belongs to the same request, otherwise returns an error
func mcpIdempotencyCheckExistingRecord(record *models.MCPIdempotencyRecord, idempotencyKey string, toolName string, requestHash string, now time.Time) ([]byte, error) {
	if record.ToolName != toolName || record.RequestHash != requestHash {
		err := NewMCPToolError(MCPToolErrorCodeIdempotencyConflict, mcpIdempotencyKeyArgumentName, "idempotency key \""+idempotencyKey+"\" has already been used for a different request (tool \""+record.ToolName+"\" at "+formatMCPUnixTime(record.CreatedUnixTime)+"), nothing has been changed; use a new idempotency key for a different request")
		err.Value = idempotencyKey
		return nil, err
	}

	if !record.Completed {
		message := "a request with the same idempotency key (started at " + formatMCPUnixTime(record.CreatedUnixTime) + ") is still in progress, retry later with the same idempotency key"

		if record.CreatedUnixTime < now.Add(-mcpIdempotencyPendingTimeout).Unix() {
			message = "a request with the same idempotency key (started at " + formatMCPUnixTime(record.CreatedUnixTime) + ") did not complete, its changes may or may not have been saved; check the current state (e.g. with query_audit_log or query_transactions) before retrying with a new idempotency key"
		}

		err := NewMCPToolError(MCPToolErrorCodeIdempotencyInProgress, mcpIdempotencyKeyArgumentName, message)
		err.Value = idempotencyKey
		return nil, err
	}

	return mcpMarkIdempotentReplay([]byte(record.Response)), nil
}

// mcpMarkIdempotentReplay returns the stored JSON object with an additional "idempotent_replay": true member (keeping the original member order)
func mcpMarkIdempotentReplay(response []byte) []byte {
	trimmedResponse := strings.TrimSpace(string(response))

	if !strings.HasPrefix(trimmedResponse, "{") || !strings.HasSuffix(trimmedResponse, "}") {
		return []byte(trimmedResponse)
	}

	var fields map[string]jsontext.Value

	if err := json.Unmarshal([]byte(trimmedResponse), &fields); err != nil {
		return []byte(trimmedResponse)
	}

	if _, exists := fields[mcpIdempotentReplayFieldName]; exists {
		return []byte(trimmedResponse)
	}

	if len(fields) == 0 {
		return []byte("{\"" + mcpIdempotentReplayFieldName + "\":true}")
	}

	return []byte("{\"" + mcpIdempotentReplayFieldName + "\":true," + strings.TrimSpace(trimmedResponse[1:]))
}

// mcpBeginWriteToolCall parses the control arguments of the MCP write tool call and checks the idempotency key (if set and not a dry run)
func mcpBeginWriteToolCall(c *core.WebContext, toolName string, supportsDryRun bool, defaultDryRun bool, callToolReq *MCPCallToolRequest, user *models.User, services MCPAvailableServices) (*mcpWriteToolCall, error) {
	var arguments jsontext.Value

	if callToolReq != nil {
		arguments = callToolReq.Arguments
	}

	controlArguments, err := mcpParseWriteControlArguments(arguments)

	if err != nil {
		return nil, err
	}

	writeCall := &mcpWriteToolCall{
		toolName:       toolName,
		uid:            user.Uid,
		idempotencyKey: controlArguments.IdempotencyKey,
		operationId:    controlArguments.OperationId,
	}

	if supportsDryRun {
		writeCall.dryRun = defaultDryRun

		if controlArguments.DryRun != nil {
			writeCall.dryRun = *controlArguments.DryRun
		}
	}

	if writeCall.idempotencyKey == "" || writeCall.dryRun {
		return writeCall, nil
	}

	requestHash, err := mcpComputeWriteRequestHash(toolName, arguments)

	if err != nil {
		return nil, err
	}

	writeCall.store = mcpGetIdempotencyRecordService(services)
	pendingRecord, replayResponse, err := mcpIdempotencyBegin(c, writeCall.store, user.Uid, writeCall.idempotencyKey, toolName, requestHash, time.Now())

	if err != nil {
		log.Warnf(c, "[mcp_tool_write_guard.mcpBeginWriteToolCall] cannot use idempotency key \"%s\" of tool \"%s\" for user \"uid:%d\", because %s", writeCall.idempotencyKey, toolName, user.Uid, err.Error())
		return nil, err
	}

	if replayResponse != nil {
		log.Infof(c, "[mcp_tool_write_guard.mcpBeginWriteToolCall] replay the stored result of idempotency key \"%s\" of tool \"%s\" for user \"uid:%d\"", writeCall.idempotencyKey, toolName, user.Uid)
	}

	writeCall.pendingRecord = pendingRecord
	writeCall.replayResponse = replayResponse

	return writeCall, nil
}

// abort deletes the pending idempotency record after the write failed, so that the request can be retried with the same idempotency key
func (w *mcpWriteToolCall) abort(c *core.WebContext) {
	if w == nil || w.pendingRecord == nil || w.store == nil {
		return
	}

	if err := w.store.DeleteRecord(c, w.uid, w.pendingRecord.RecordId); err != nil {
		log.Warnf(c, "[mcp_tool_write_guard.abort] failed to delete pending idempotency record of key \"%s\" for user \"uid:%d\", because %s", w.idempotencyKey, w.uid, err.Error())
	}

	w.pendingRecord = nil
}

// finish stores the result for the idempotency key (if set) and records the audit log if the write has been committed
func (w *mcpWriteToolCall) finish(c *core.WebContext, services MCPAvailableServices, structuredResponse any) {
	if w == nil {
		return
	}

	content, err := json.Marshal(structuredResponse)

	if err != nil {
		log.Errorf(c, "[mcp_tool_write_guard.finish] failed to serialize the result of tool \"%s\" for user \"uid:%d\", because %s", w.toolName, w.uid, err.Error())
		w.abort(c)
		return
	}

	var response map[string]any

	if err := json.Unmarshal(content, &response); err != nil {
		response = nil
	}

	entityType, action := mcpAuditGetEntityTypeAndAction(w.toolName)
	entityIds := mcpAuditExtractEntityIds(response, entityType)

	if w.pendingRecord != nil && w.store != nil {
		storedResponse := string(content)

		if len(content) > mcpIdempotencyMaxStoredResponseSize {
			storedResponse = mcpIdempotencyBuildResponseSummary(response, entityIds)
		}

		if err := w.store.CompleteRecord(c, w.uid, w.pendingRecord.RecordId, storedResponse); err != nil {
			log.Errorf(c, "[mcp_tool_write_guard.finish] failed to save the result of idempotency key \"%s\" for user \"uid:%d\", because %s", w.idempotencyKey, w.uid, err.Error())
		}
	}

	if w.dryRun || !mcpWriteResponseIsCommitted(response) {
		return
	}

	auditLog := mcpAuditBuildLog(c, w, entityType, action, entityIds, response)

	if err := mcpGetAuditLogService(services).CreateAuditLog(c, auditLog); err != nil {
		log.Errorf(c, "[mcp_tool_write_guard.finish] failed to save audit log of tool \"%s\" for user \"uid:%d\", because %s", w.toolName, w.uid, err.Error())
	}
}

// mcpIdempotencyBuildResponseSummary returns a small summary of the result which is stored instead of a result exceeding the maximum stored size
func mcpIdempotencyBuildResponseSummary(response map[string]any, entityIds []string) string {
	summary := map[string]any{
		"success":                   true,
		"stored_response_truncated": true,
		"entity_ids":                entityIds,
	}

	for _, key := range []string{"success", "dry_run", "committed", "status"} {
		if value, exists := response[key]; exists {
			summary[key] = value
		}
	}

	content, err := json.Marshal(summary)

	if err != nil {
		return "{\"success\":true,\"stored_response_truncated\":true}"
	}

	return string(content)
}

// mcpWriteResponseIsCommitted returns whether the result of the write tool indicates that changes have been saved:
// dry runs are never committed, "committed" is used if present, otherwise the result status must not be a non-committing status
func mcpWriteResponseIsCommitted(response map[string]any) bool {
	if response == nil {
		return false
	}

	if dryRun, ok := response["dry_run"].(bool); ok && dryRun {
		return false
	}

	if committed, ok := response["committed"].(bool); ok {
		return committed
	}

	if status, ok := response["status"].(string); ok {
		switch status {
		case MCPWriteStatusExisting, MCPWriteStatusUnchanged, MCPWriteStatusSkipped, MCPWriteStatusFailed, MCPWriteStatusWouldBeCreated, MCPWriteStatusWouldBeUpdated, MCPWriteStatusWouldBeDeleted:
			return false
		}
	}

	if success, ok := response["success"].(bool); ok && !success {
		return false
	}

	return true
}

// createMCPToolReplayResponse returns the tool result of a replayed idempotent request
func createMCPToolReplayResponse(c *core.WebContext, response []byte) any {
	callToolResp := MCPCallToolResponse[MCPTextContent]{
		Content: []*MCPTextContent{
			NewMCPTextContent(string(response)),
		},
		IsError: false,
	}

	if c.GetHeader(MCPProtocolVersionHeaderName) >= string(ToolResultStructuredContentMinVersion) {
		callToolResp.StructuredContent = jsontext.Value(response)
	}

	return callToolResp
}
