package mcp

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPAuditGetEntityTypeAndAction(t *testing.T) {
	testCases := []struct {
		toolName   string
		entityType string
		action     string
	}{
		{"add_transaction", "transaction", "create"},
		{"update_transaction", "transaction", "update"},
		{"delete_transaction", "transaction", "delete"},
		{"import_transactions", "transaction", "import"},
		{"create_account", "account", "create"},
		{"hide_account", "account", "hide"},
		{"set_account_reconciled_time", "account", "reconcile"},
		{"batch_create_transaction_categories", "transaction_category", "create"},
		{"reorder_transaction_categories", "transaction_category", "reorder"},
		{"create_transaction_tags", "transaction_tag", "create"},
		{"reorder_transaction_tag_groups", "transaction_tag_group", "reorder"},
		{"upload_transaction_attachment", "transaction_attachment", "upload"},
		{"detach_transaction_attachment", "transaction_attachment", "detach"},
		{"delete_transaction_template", "transaction_template", "delete"},
	}

	for _, testCase := range testCases {
		entityType, action := mcpAuditGetEntityTypeAndAction(testCase.toolName)
		assert.Equal(t, testCase.entityType, entityType, testCase.toolName)
		assert.Equal(t, testCase.action, action, testCase.toolName)
	}
}

func TestMCPAuditGetEntityTypeAndAction_AllWriteTools(t *testing.T) {
	initializeMCPContainerTest(t)

	for _, tool := range Container.GetMCPTools() {
		if tool.Annotations.ReadOnlyHint {
			continue
		}

		entityType, action := mcpAuditGetEntityTypeAndAction(tool.Name)
		assert.NotEqual(t, "", entityType, tool.Name)
		assert.NotEqual(t, "write", action, tool.Name)
		assert.LessOrEqual(t, len(entityType), 32, tool.Name)
	}
}

func parseMCPAuditTestResponse(t *testing.T, content string) map[string]any {
	t.Helper()
	var response map[string]any

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		t.Fatal(err)
	}

	return response
}

func TestMCPAuditExtractEntityIds_AddTransaction(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"committed":true,"transaction_id":"100","transaction":{"id":"100","account_id":"1","category_id":"2","tags":[{"id":"7"}]}}`)
	assert.Equal(t, []string{"100"}, mcpAuditExtractEntityIds(response, "transaction"))
}

func TestMCPAuditExtractEntityIds_UpdateTransaction(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"committed":true,"transaction":{"id":"100"},"account_balances":[{"account_id":"1"},{"account_id":"2"}]}`)
	assert.Equal(t, []string{"100"}, mcpAuditExtractEntityIds(response, "transaction"))
}

func TestMCPAuditExtractEntityIds_DeleteTransaction(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"deleted_transaction_id":"100","permanent":true}`)
	assert.Equal(t, []string{"100"}, mcpAuditExtractEntityIds(response, "transaction"))
}

func TestMCPAuditExtractEntityIds_ImportResults(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"committed":true,"results":[{"row":1,"transaction_id":"100","mapping":{"account_id":"1"}},{"row":2,"status":"skipped","duplicate_candidates":[{"transaction_id":"90"}]},{"row":3,"transaction_id":"101"}]}`)
	assert.ElementsMatch(t, []string{"100", "101"}, mcpAuditExtractEntityIds(response, "transaction"))
}

func TestMCPAuditExtractEntityIds_Categories(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"category_ids":["10","11"],"categories":[{"id":"12","parent_category_id":"10"}],"transaction_id":"5"}`)
	assert.ElementsMatch(t, []string{"10", "11", "12"}, mcpAuditExtractEntityIds(response, "transaction_category"))
}

func TestMCPAuditExtractEntityIds_IgnoresNonNumericIds(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"transaction_id":"abc","transaction":{"id":""}}`)
	assert.Equal(t, 0, len(mcpAuditExtractEntityIds(response, "transaction")))
	assert.Equal(t, 0, len(mcpAuditExtractEntityIds(nil, "transaction")))
}

func TestMCPAuditFormatAndParseEntityIds(t *testing.T) {
	assert.Equal(t, ",1,22,333,", mcpAuditFormatEntityIds([]string{"1", "22", "333"}))
	assert.Equal(t, "", mcpAuditFormatEntityIds(nil))
	assert.Equal(t, []string{"1", "22", "333"}, mcpAuditParseEntityIds(",1,22,333,"))
	assert.Equal(t, []string{}, mcpAuditParseEntityIds(""))
}

func TestMCPAuditBuildDetails(t *testing.T) {
	response := parseMCPAuditTestResponse(t, `{"success":true,"status":"updated","changed_fields":["comment"],"transaction":{"id":"1"}}`)
	details := parseMCPAuditTestResponse(t, mcpAuditBuildDetails(response, true))
	assert.Equal(t, "updated", details["status"])
	assert.Equal(t, []any{"comment"}, details["changed_fields"])
	assert.Equal(t, true, details["entity_ids_truncated"])
	_, exists := details["transaction"]
	assert.False(t, exists)
}
