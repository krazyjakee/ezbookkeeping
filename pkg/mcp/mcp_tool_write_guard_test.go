package mcp

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// mcpTestIdempotencyStore is an in-memory idempotency store for tests
type mcpTestIdempotencyStore struct {
	records        map[string]*models.MCPIdempotencyRecord
	nextRecordId   int64
	deletedExpired int
	now            time.Time
}

func newMCPTestIdempotencyStore(now time.Time) *mcpTestIdempotencyStore {
	return &mcpTestIdempotencyStore{
		records:      make(map[string]*models.MCPIdempotencyRecord),
		nextRecordId: 1,
		now:          now,
	}
}

func (s *mcpTestIdempotencyStore) key(uid int64, idempotencyKey string) string {
	return strconv.FormatInt(uid, 10) + "\n" + idempotencyKey
}

func (s *mcpTestIdempotencyStore) GetRecordByKey(c core.Context, uid int64, idempotencyKey string) (*models.MCPIdempotencyRecord, error) {
	record, exists := s.records[s.key(uid, idempotencyKey)]

	if !exists {
		return nil, nil
	}

	recordCopy := *record
	return &recordCopy, nil
}

func (s *mcpTestIdempotencyStore) TryCreatePendingRecord(c core.Context, record *models.MCPIdempotencyRecord) (bool, error) {
	if _, exists := s.records[s.key(record.Uid, record.IdempotencyKey)]; exists {
		return false, nil
	}

	record.RecordId = s.nextRecordId
	record.Completed = false
	record.CreatedUnixTime = s.now.Unix()
	s.nextRecordId++

	recordCopy := *record
	s.records[s.key(record.Uid, record.IdempotencyKey)] = &recordCopy
	return true, nil
}

func (s *mcpTestIdempotencyStore) CompleteRecord(c core.Context, uid int64, recordId int64, response string) error {
	for _, record := range s.records {
		if record.Uid == uid && record.RecordId == recordId {
			record.Completed = true
			record.Response = response
			return nil
		}
	}

	return errors.New("record not found")
}

func (s *mcpTestIdempotencyStore) DeleteRecord(c core.Context, uid int64, recordId int64) error {
	for key, record := range s.records {
		if record.Uid == uid && record.RecordId == recordId {
			delete(s.records, key)
			return nil
		}
	}

	return nil
}

func (s *mcpTestIdempotencyStore) DeleteExpiredRecords(c core.Context, uid int64, createdBeforeUnixTime int64) error {
	for key, record := range s.records {
		if record.Uid == uid && record.CreatedUnixTime < createdBeforeUnixTime {
			delete(s.records, key)
			s.deletedExpired++
		}
	}

	return nil
}

func TestMCPComputeWriteRequestHash_IgnoresOrderWhitespaceAndControlArguments(t *testing.T) {
	hash1, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{"type":"expense","amount":"120.00","tags":["a","b"],"geo_location":{"latitude":1,"longitude":2}}`))
	assert.Nil(t, err)

	hash2, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{ "geo_location": {"longitude":2, "latitude":1}, "tags":["a","b"], "amount":"120.00", "type":"expense", "idempotency_key":"k1", "operation_id":"op-1", "dry_run":false }`))
	assert.Nil(t, err)
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, 64, len(hash1))
}

func TestMCPComputeWriteRequestHash_DifferentRequests(t *testing.T) {
	hash1, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{"type":"expense","amount":"120.00"}`))
	assert.Nil(t, err)

	hash2, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{"type":"expense","amount":"121.00"}`))
	assert.Nil(t, err)
	assert.NotEqual(t, hash1, hash2)

	hash3, err := mcpComputeWriteRequestHash("update_transaction", jsontext.Value(`{"type":"expense","amount":"120.00"}`))
	assert.Nil(t, err)
	assert.NotEqual(t, hash1, hash3)

	hash4, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{"type":"expense","amount":"120.00","tags":["b","a"]}`))
	assert.Nil(t, err)

	hash5, err := mcpComputeWriteRequestHash("add_transaction", jsontext.Value(`{"type":"expense","amount":"120.00","tags":["a","b"]}`))
	assert.Nil(t, err)
	assert.NotEqual(t, hash4, hash5)
}

func TestMCPComputeWriteRequestHash_EmptyArguments(t *testing.T) {
	hash1, err := mcpComputeWriteRequestHash("hide_account", nil)
	assert.Nil(t, err)

	hash2, err := mcpComputeWriteRequestHash("hide_account", jsontext.Value(`{"idempotency_key":"k1"}`))
	assert.Nil(t, err)
	assert.Equal(t, hash1, hash2)

	_, err = mcpComputeWriteRequestHash("hide_account", jsontext.Value(`[1,2]`))
	assert.NotNil(t, err)
}

func TestMCPParseWriteControlArguments(t *testing.T) {
	arguments, err := mcpParseWriteControlArguments(jsontext.Value(`{"idempotency_key":"key-1","operation_id":"op-1","dry_run":true,"amount":"1"}`))
	assert.Nil(t, err)
	assert.Equal(t, "key-1", arguments.IdempotencyKey)
	assert.Equal(t, "op-1", arguments.OperationId)
	assert.True(t, *arguments.DryRun)

	arguments, err = mcpParseWriteControlArguments(jsontext.Value(`{"amount":"1"}`))
	assert.Nil(t, err)
	assert.Equal(t, "", arguments.IdempotencyKey)
	assert.Nil(t, arguments.DryRun)

	arguments, err = mcpParseWriteControlArguments(nil)
	assert.Nil(t, err)
	assert.Equal(t, "", arguments.IdempotencyKey)
}

func TestMCPParseWriteControlArguments_InvalidValues(t *testing.T) {
	_, err := mcpParseWriteControlArguments(jsontext.Value(`{"idempotency_key":123}`))
	assert.Equal(t, mcpIdempotencyKeyArgumentName, mcpTestGetToolError(t, err).Field)

	_, err = mcpParseWriteControlArguments(jsontext.Value(`{"idempotency_key":"   "}`))
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTestGetToolError(t, err).Code)

	_, err = mcpParseWriteControlArguments(jsontext.Value(`{"idempotency_key":"` + strings.Repeat("k", models.MCPIdempotencyKeyMaxLength+1) + `"}`))
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, int64(models.MCPIdempotencyKeyMaxLength), *toolErr.Limit)

	_, err = mcpParseWriteControlArguments(jsontext.Value(`{"operation_id":"op\u0000"}`))
	assert.Equal(t, mcpOperationIdArgumentName, mcpTestGetToolError(t, err).Field)
}

func TestMCPIdempotencyBegin_ReplaySameRequest(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	pendingRecord, replayResponse, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)
	assert.Nil(t, replayResponse)
	assert.NotNil(t, pendingRecord)

	assert.Nil(t, store.CompleteRecord(nil, 1, pendingRecord.RecordId, `{"success":true,"committed":true,"transaction_id":"100"}`))

	pendingRecord, replayResponse, err = mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now.Add(time.Hour))
	assert.Nil(t, err)
	assert.Nil(t, pendingRecord)

	var response map[string]any
	assert.Nil(t, json.Unmarshal(replayResponse, &response))
	assert.Equal(t, true, response[mcpIdempotentReplayFieldName])
	assert.Equal(t, "100", response["transaction_id"])
	assert.Equal(t, true, response["committed"])
	assert.True(t, strings.HasPrefix(string(replayResponse), `{"idempotent_replay":true,"success":true`))
}

func TestMCPIdempotencyBegin_ConflictDifferentRequest(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	pendingRecord, _, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)
	assert.Nil(t, store.CompleteRecord(nil, 1, pendingRecord.RecordId, `{"success":true}`))

	_, replayResponse, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-2", now)
	assert.Nil(t, replayResponse)
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeIdempotencyConflict, toolErr.Code)
	assert.Equal(t, mcpIdempotencyKeyArgumentName, toolErr.Field)
	assert.Equal(t, "key-1", toolErr.Value)

	_, _, err = mcpIdempotencyBegin(nil, store, 1, "key-1", "update_transaction", "hash-1", now)
	assert.Equal(t, MCPToolErrorCodeIdempotencyConflict, mcpTestGetToolError(t, err).Code)
}

func TestMCPIdempotencyBegin_KeysAreScopedByUser(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	pendingRecord, _, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)
	assert.Nil(t, store.CompleteRecord(nil, 1, pendingRecord.RecordId, `{"success":true}`))

	pendingRecord, replayResponse, err := mcpIdempotencyBegin(nil, store, 2, "key-1", "add_transaction", "hash-2", now)
	assert.Nil(t, err)
	assert.Nil(t, replayResponse)
	assert.NotNil(t, pendingRecord)
}

func TestMCPIdempotencyBegin_InProgress(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	_, _, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)

	_, replayResponse, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now.Add(time.Minute))
	assert.Nil(t, replayResponse)
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeIdempotencyInProgress, toolErr.Code)
	assert.True(t, strings.Contains(toolErr.Message, "still in progress"))

	_, _, err = mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now.Add(mcpIdempotencyPendingTimeout+time.Minute))
	toolErr = mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeIdempotencyInProgress, toolErr.Code)
	assert.True(t, strings.Contains(toolErr.Message, "did not complete"))
}

func TestMCPIdempotencyBegin_FailedWriteCanBeRetried(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	pendingRecord, _, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)

	writeCall := &mcpWriteToolCall{
		uid:            1,
		idempotencyKey: "key-1",
		store:          store,
		pendingRecord:  pendingRecord,
	}
	writeCall.abort(nil)
	assert.Nil(t, writeCall.pendingRecord)

	pendingRecord, replayResponse, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)
	assert.Nil(t, replayResponse)
	assert.NotNil(t, pendingRecord)
}

func TestMCPIdempotencyBegin_ExpiredRecord(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	pendingRecord, _, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-1", now)
	assert.Nil(t, err)
	assert.Nil(t, store.CompleteRecord(nil, 1, pendingRecord.RecordId, `{"success":true}`))

	later := now.Add(mcpIdempotencyRecordRetention + time.Hour)
	store.now = later
	pendingRecord, replayResponse, err := mcpIdempotencyBegin(nil, store, 1, "key-1", "add_transaction", "hash-2", later)
	assert.Nil(t, err)
	assert.Nil(t, replayResponse)
	assert.NotNil(t, pendingRecord)
	assert.Equal(t, "hash-2", pendingRecord.RequestHash)
}

func TestMCPIdempotencyBegin_LazyCleanup(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := newMCPTestIdempotencyStore(now)

	_, _, err := mcpIdempotencyBegin(nil, store, 1, "old-key", "add_transaction", "hash-1", now)
	assert.Nil(t, err)

	later := now.Add(mcpIdempotencyRecordRetention + time.Hour)
	store.now = later
	_, _, err = mcpIdempotencyBegin(nil, store, 1, "new-key", "add_transaction", "hash-1", later)
	assert.Nil(t, err)
	assert.Equal(t, 1, store.deletedExpired)

	record, err := store.GetRecordByKey(nil, 1, "old-key")
	assert.Nil(t, err)
	assert.Nil(t, record)
}

func TestMCPMarkIdempotentReplay(t *testing.T) {
	assert.Equal(t, `{"idempotent_replay":true,"success":true}`, string(mcpMarkIdempotentReplay([]byte(`{"success":true}`))))
	assert.Equal(t, `{"idempotent_replay":true}`, string(mcpMarkIdempotentReplay([]byte(`{}`))))
	assert.Equal(t, `{"idempotent_replay":true,"a":1}`, string(mcpMarkIdempotentReplay([]byte(`{"idempotent_replay":true,"a":1}`))))
	assert.Equal(t, `[1]`, string(mcpMarkIdempotentReplay([]byte(`[1]`))))
}

func TestMCPWriteResponseIsCommitted(t *testing.T) {
	assert.False(t, mcpWriteResponseIsCommitted(nil))
	assert.False(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "dry_run": true, "committed": true}))
	assert.True(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "committed": true}))
	assert.False(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "committed": false}))
	assert.True(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "status": MCPWriteStatusCreated}))
	assert.False(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "status": MCPWriteStatusExisting}))
	assert.False(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "status": MCPWriteStatusUnchanged}))
	assert.True(t, mcpWriteResponseIsCommitted(map[string]any{"success": true, "deleted_transaction_id": "1"}))
	assert.False(t, mcpWriteResponseIsCommitted(map[string]any{"success": false}))
}

func TestMCPToolInputSupportsDryRun(t *testing.T) {
	assert.True(t, mcpToolInputSupportsDryRun(reflect.TypeOf(&MCPAddTransactionRequest{})))
	assert.True(t, mcpToolInputSupportsDryRun(reflect.TypeOf(&MCPImportTransactionsRequest{})))
	assert.False(t, mcpToolInputSupportsDryRun(reflect.TypeOf(&MCPHideAccountRequest{})))
	assert.False(t, mcpToolInputSupportsDryRun(nil))
}

func TestMCPWriteToolsAcceptControlArguments(t *testing.T) {
	initializeMCPContainerTest(t)

	for _, tool := range Container.GetMCPTools() {
		if tool.InputSchema.Properties == nil {
			assert.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
			continue
		}

		_, hasIdempotencyKey := tool.InputSchema.Properties.Get(mcpIdempotencyKeyArgumentName)
		_, hasOperationId := tool.InputSchema.Properties.Get(mcpOperationIdArgumentName)

		if tool.Annotations.ReadOnlyHint {
			assert.False(t, hasIdempotencyKey, tool.Name)
			assert.False(t, hasOperationId, tool.Name)
		} else {
			assert.True(t, hasIdempotencyKey, tool.Name)
			assert.True(t, hasOperationId, tool.Name)
		}
	}
}

func TestMCPImportTransactionsDefaultsToDryRun(t *testing.T) {
	var handler any = MCPImportTransactionsToolHandler
	defaultDryRunHandler, ok := handler.(mcpDefaultDryRunToolHandler)
	assert.True(t, ok)
	assert.True(t, defaultDryRunHandler.DefaultDryRun())

	handler = MCPAddTransactionToolHandler
	_, ok = handler.(mcpDefaultDryRunToolHandler)
	assert.False(t, ok)
}
