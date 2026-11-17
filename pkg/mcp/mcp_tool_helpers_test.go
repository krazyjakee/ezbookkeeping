package mcp

import (
	"encoding/json/v2"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

type mcpToolHelpersTestEntity struct {
	id   int64
	name string
}

type mcpToolHelpersTestModel struct {
	Name    string `binding:"required,notBlank,max=64"`
	Comment string `binding:"max=1024"`
	Other   string `binding:"required"`
}

func newMCPToolHelpersTestResolver() *mcpEntityResolver[*mcpToolHelpersTestEntity] {
	return newMCPEntityResolver("test entity",
		func(e *mcpToolHelpersTestEntity) int64 { return e.id },
		func(e *mcpToolHelpersTestEntity) string { return e.name },
		func(e *mcpToolHelpersTestEntity) string { return "desc " + e.name })
}

func TestGetModelBindingMaxValue(t *testing.T) {
	modelType := reflect.TypeOf(mcpToolHelpersTestModel{})
	assert.Equal(t, 64, getModelBindingMaxValue(modelType, "Name", 1))
	assert.Equal(t, 1024, getModelBindingMaxValue(modelType, "Comment", 1))
	assert.Equal(t, 7, getModelBindingMaxValue(modelType, "Other", 7))
	assert.Equal(t, 8, getModelBindingMaxValue(modelType, "NotExists", 8))
	assert.Equal(t, 64, getModelBindingMaxValue(reflect.TypeOf(&mcpToolHelpersTestModel{}), "Name", 1))
}

func TestGetModelBindingMaxValue_ConsistentWithModels(t *testing.T) {
	field, exists := reflect.TypeOf(models.TransactionCreateRequest{}).FieldByName("Comment")
	assert.True(t, exists)
	assert.Contains(t, field.Tag.Get("binding"), "max=")
	assert.True(t, strings.Contains(field.Tag.Get("binding"), "max="+strconv.Itoa(mcpTransactionCommentMaxLength)))

	field, exists = reflect.TypeOf(models.TransactionModifyRequest{}).FieldByName("Comment")
	assert.True(t, exists)
	assert.True(t, strings.Contains(field.Tag.Get("binding"), "max="+strconv.Itoa(mcpTransactionCommentMaxLength)))
}

func TestCheckMCPStringLength(t *testing.T) {
	assert.Nil(t, checkMCPStringLength("comment", "", false, 10))
	assert.Nil(t, checkMCPStringLength("comment", "1234567890", false, 10))
	assert.Nil(t, checkMCPStringLength("comment", "一二三四五六七八九十", false, 10))

	err := checkMCPStringLength("comment", "12345678901", false, 10)
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "comment", toolErr.Field)
	assert.Equal(t, int64(10), *toolErr.Limit)
	assert.Equal(t, int64(11), *toolErr.Actual)

	err = checkMCPStringLength("name", "  ", true, 10)
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "name", toolErr.Field)
}

func TestCheckMCPStringLength_TransactionCommentLimit(t *testing.T) {
	assert.Nil(t, checkMCPStringLength("comment", strings.Repeat("a", mcpTransactionCommentMaxLength), false, mcpTransactionCommentMaxLength))

	err := checkMCPStringLength("comment", strings.Repeat("a", mcpTransactionCommentMaxLength+1), false, mcpTransactionCommentMaxLength)
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, int64(mcpTransactionCommentMaxLength), *toolErr.Limit)
}

func TestParseMCPOptionalId(t *testing.T) {
	id, err := parseMCPOptionalId("id", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), id)

	id, err = parseMCPOptionalId("id", "123")
	assert.Nil(t, err)
	assert.Equal(t, int64(123), id)

	_, err = parseMCPOptionalId("id", "abc")
	assert.NotNil(t, err)

	_, err = parseMCPOptionalId("id", "-1")
	assert.NotNil(t, err)

	_, err = parseMCPRequiredId("id", "")
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
}

func TestParseMCPAmount(t *testing.T) {
	amount, err := parseMCPAmount("amount", "12.34", true)
	assert.Nil(t, err)
	assert.Equal(t, int64(1234), amount)

	_, err = parseMCPAmount("amount", "", true)
	assert.NotNil(t, err)

	amount, err = parseMCPAmount("amount", "", false)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), amount)

	_, err = parseMCPAmount("amount", "abc", true)
	assert.NotNil(t, err)

	_, err = parseMCPAmount("amount", "99999999999999999", true)
	assert.NotNil(t, err)
}

func TestMCPEntityResolver_ResolveById(t *testing.T) {
	resolver := newMCPToolHelpersTestResolver()
	items := []*mcpToolHelpersTestEntity{{id: 1, name: "Food"}, {id: 2, name: "Food"}}

	item, err := resolver.Resolve(items, "category_id", "2", "category_name", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(2), item.id)

	_, err = resolver.Resolve(items, "category_id", "3", "category_name", "")
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeNotFound, toolErr.Code)
	assert.Equal(t, "category_id", toolErr.Field)

	_, err = resolver.Resolve(items, "category_id", "1", "category_name", "Other")
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "category_name", toolErr.Field)
}

func TestMCPEntityResolver_ResolveByName(t *testing.T) {
	resolver := newMCPToolHelpersTestResolver()
	items := []*mcpToolHelpersTestEntity{{id: 1, name: "Food"}, {id: 2, name: "food"}, {id: 3, name: "Travel"}}

	item, err := resolver.Resolve(items, "category_id", "", "category_name", "Food")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), item.id)

	item, err = resolver.Resolve(items, "category_id", "", "category_name", " travel ")
	assert.Nil(t, err)
	assert.Equal(t, int64(3), item.id)

	_, err = resolver.Resolve(items, "category_id", "", "category_name", "FOOD")
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, "category_name", toolErr.Field)
	assert.Equal(t, 2, len(toolErr.Candidates))
	assert.Equal(t, "1", toolErr.Candidates[0].Id)
	assert.Equal(t, "desc Food", toolErr.Candidates[0].Description)
	assert.Equal(t, "2", toolErr.Candidates[1].Id)

	_, err = resolver.Resolve(items, "category_id", "", "category_name", "Unknown")
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeNotFound, toolErr.Code)

	_, err = resolver.Resolve(items, "category_id", "", "category_name", "")
	assert.True(t, errors.As(err, &toolErr))
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
}

func TestCreateMCPToolErrorResponse(t *testing.T) {
	result, err := createMCPToolErrorResponse(NewMCPFieldTooLongError("comment", 1024, 1025))
	assert.Nil(t, err)

	response, ok := result.(MCPCallToolResponse[MCPTextContent])
	assert.True(t, ok)
	assert.True(t, response.IsError)
	assert.Nil(t, response.StructuredContent)
	assert.Equal(t, 1, len(response.Content))

	var errorResult MCPToolErrorResult
	assert.Nil(t, json.Unmarshal([]byte(response.Content[0].Text), &errorResult))
	assert.False(t, errorResult.Success)
	assert.Equal(t, MCPToolErrorCodeTooLong, errorResult.Error.Code)
	assert.Equal(t, "comment", errorResult.Error.Field)
	assert.Equal(t, int64(1024), *errorResult.Error.Limit)
	assert.Equal(t, int64(1025), *errorResult.Error.Actual)
}

func TestGetServerCapabilities(t *testing.T) {
	config := &settings.Config{
		EnableTransactionPictures:     true,
		MaxTransactionPictureFileSize: 1024,
		EnableScheduledTransaction:    true,
	}
	user := &models.User{
		DefaultCurrency:      "GBP",
		TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL,
	}

	response := MCPGetServerCapabilitiesToolHandler.createNewMCPGetServerCapabilitiesResponse(user, config)

	assert.Equal(t, mcpTransactionCommentMaxLength, response.Limits.TransactionDescriptionMaxLength)
	assert.Equal(t, models.MaximumTagsCountOfTransaction, response.Limits.TransactionTagsMaxCount)
	assert.Equal(t, mcpMaximumBatchItemsCount, response.Limits.BatchMaxItemsCount)
	assert.True(t, response.Attachments.Enabled)
	assert.Equal(t, uint32(1024), response.Attachments.MaxFileSizeBytes)
	assert.False(t, response.Attachments.UploadViaMCP)
	assert.True(t, len(response.Attachments.SupportedTypes) > 0)
	assert.True(t, response.ScheduledTransactions.Enabled)
	assert.Equal(t, "GBP", response.CurrentUser.DefaultCurrency)
	assert.Equal(t, "all", response.CurrentUser.TransactionEditScope)
	assert.Contains(t, response.SupportedCurrencies, "GBP")
	assert.Contains(t, response.SupportedCurrencies, "USD")

	for i := 1; i < len(response.SupportedCurrencies); i++ {
		assert.True(t, response.SupportedCurrencies[i-1] < response.SupportedCurrencies[i])
	}
}

func TestMCPContainerGetMCPTools_Annotations(t *testing.T) {
	initializeMCPContainerTest(t)
	toolNames := make(map[string]bool)

	for _, tool := range Container.GetMCPTools() {
		assert.False(t, toolNames[tool.Name], "duplicate tool name "+tool.Name)
		toolNames[tool.Name] = true

		assert.NotNil(t, tool.Annotations, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
		assert.NotNil(t, tool.InputSchema, tool.Name)
		assert.NotNil(t, tool.OutputSchema, tool.Name)

		if tool.Annotations.ReadOnlyHint {
			assert.False(t, tool.Annotations.DestructiveHint, tool.Name)
		}

		if strings.HasPrefix(tool.Name, "delete_") {
			assert.False(t, tool.Annotations.ReadOnlyHint, tool.Name)
			assert.True(t, tool.Annotations.DestructiveHint, tool.Name)
		}

		if strings.HasPrefix(tool.Name, "get_") || strings.HasPrefix(tool.Name, "list_") || strings.HasPrefix(tool.Name, "query_") {
			assert.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
			assert.True(t, tool.Annotations.IdempotentHint, tool.Name)
		}

		if strings.HasPrefix(tool.Name, "create_") || strings.HasPrefix(tool.Name, "add_") || strings.HasPrefix(tool.Name, "batch_create_") {
			assert.False(t, tool.Annotations.ReadOnlyHint, tool.Name)
			assert.False(t, tool.Annotations.DestructiveHint, tool.Name)
		}
	}

	assert.True(t, toolNames["get_server_capabilities"])
	assert.True(t, toolNames["get_transaction"])
	assert.True(t, toolNames["update_transaction"])
}
