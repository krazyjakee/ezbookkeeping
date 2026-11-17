package mcp

import (
	"encoding/json/v2"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCP write operation result statuses
const (
	MCPWriteStatusCreated        = "created"
	MCPWriteStatusUpdated        = "updated"
	MCPWriteStatusDeleted        = "deleted"
	MCPWriteStatusExisting       = "existing"
	MCPWriteStatusWouldBeCreated = "would_be_created"
	MCPWriteStatusWouldBeUpdated = "would_be_updated"
	MCPWriteStatusWouldBeDeleted = "would_be_deleted"
	MCPWriteStatusFailed         = "failed"
	MCPWriteStatusSkipped        = "skipped"
	MCPWriteStatusUnchanged      = "unchanged"
)

// Textual account categories used in MCP tools
const (
	mcpAccountCategoryCash                 = "cash"
	mcpAccountCategoryCheckingAccount      = "checking_account"
	mcpAccountCategorySavingsAccount       = "savings_account"
	mcpAccountCategoryCreditCard           = "credit_card"
	mcpAccountCategoryVirtual              = "virtual"
	mcpAccountCategoryDebt                 = "debt"
	mcpAccountCategoryReceivables          = "receivables"
	mcpAccountCategoryCertificateOfDeposit = "certificate_of_deposit"
	mcpAccountCategoryInvestment           = "investment"
)

var mcpAccountCategoryNameMap = map[models.AccountCategory]string{
	models.ACCOUNT_CATEGORY_CASH:                   mcpAccountCategoryCash,
	models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT:       mcpAccountCategoryCheckingAccount,
	models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT:        mcpAccountCategorySavingsAccount,
	models.ACCOUNT_CATEGORY_CREDIT_CARD:            mcpAccountCategoryCreditCard,
	models.ACCOUNT_CATEGORY_VIRTUAL:                mcpAccountCategoryVirtual,
	models.ACCOUNT_CATEGORY_DEBT:                   mcpAccountCategoryDebt,
	models.ACCOUNT_CATEGORY_RECEIVABLES:            mcpAccountCategoryReceivables,
	models.ACCOUNT_CATEGORY_CERTIFICATE_OF_DEPOSIT: mcpAccountCategoryCertificateOfDeposit,
	models.ACCOUNT_CATEGORY_INVESTMENT:             mcpAccountCategoryInvestment,
}

// Textual transaction category types used in MCP tools
const (
	mcpCategoryTypeIncome   = "income"
	mcpCategoryTypeExpense  = "expense"
	mcpCategoryTypeTransfer = "transfer"
)

// Maximum field lengths, which are read from the binding tags of request models in pkg/models, so that MCP tools
// always validate input (including dry run) with the same limits as the HTTP API and the database columns
var (
	mcpTransactionCommentMaxLength = getModelBindingMaxValue(reflect.TypeOf(models.TransactionCreateRequest{}), "Comment", 1024)
	mcpAccountNameMaxLength        = getModelBindingMaxValue(reflect.TypeOf(models.AccountCreateRequest{}), "Name", 64)
	mcpAccountCommentMaxLength     = getModelBindingMaxValue(reflect.TypeOf(models.AccountCreateRequest{}), "Comment", 255)
	mcpCategoryNameMaxLength       = getModelBindingMaxValue(reflect.TypeOf(models.TransactionCategoryCreateRequest{}), "Name", 64)
	mcpCategoryCommentMaxLength    = getModelBindingMaxValue(reflect.TypeOf(models.TransactionCategoryCreateRequest{}), "Comment", 255)
	mcpTagNameMaxLength            = getModelBindingMaxValue(reflect.TypeOf(models.TransactionTagCreateRequest{}), "Name", 64)
	mcpTagGroupNameMaxLength       = getModelBindingMaxValue(reflect.TypeOf(models.TransactionTagGroupCreateRequest{}), "Name", 64)
	mcpTemplateNameMaxLength       = getModelBindingMaxValue(reflect.TypeOf(models.TransactionTemplateCreateRequest{}), "Name", 64)
	mcpTemplateCommentMaxLength    = getModelBindingMaxValue(reflect.TypeOf(models.TransactionTemplateCreateRequest{}), "Comment", 1024)
)

// mcpMaximumBatchItemsCount is the maximum count of items in one batch MCP tool call
const mcpMaximumBatchItemsCount = 200

// getModelBindingMaxValue returns the "max" value in the binding tag of the specified field of the model type, or returns the default value if not found
func getModelBindingMaxValue(modelType reflect.Type, fieldName string, defaultValue int) int {
	for modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		return defaultValue
	}

	field, exists := modelType.FieldByName(fieldName)

	if !exists {
		return defaultValue
	}

	bindingTag := field.Tag.Get("binding")

	for _, rule := range strings.Split(bindingTag, ",") {
		rule = strings.TrimSpace(rule)

		if !strings.HasPrefix(rule, "max=") {
			continue
		}

		value, err := strconv.Atoi(strings.TrimPrefix(rule, "max="))

		if err != nil || value <= 0 {
			return defaultValue
		}

		return value
	}

	return defaultValue
}

// parseMCPToolArguments parses the arguments of the MCP call tool request into the specified type
func parseMCPToolArguments[T any](callToolReq *MCPCallToolRequest, allowEmpty bool) (*T, error) {
	var request T

	if callToolReq == nil || callToolReq.Arguments == nil || len(callToolReq.Arguments) == 0 {
		if allowEmpty {
			return &request, nil
		}

		return nil, errs.ErrIncompleteOrIncorrectSubmission
	}

	if err := json.Unmarshal(callToolReq.Arguments, &request); err != nil {
		return nil, NewMCPToolError(MCPToolErrorCodeInvalidArguments, "", "arguments are invalid: "+err.Error())
	}

	return &request, nil
}

// newMCPJSONTextResult returns the structured response and the text content (serialized structured response) of the MCP tool
func newMCPJSONTextResult(response any) (any, []*MCPTextContent, error) {
	content, err := json.Marshal(response)

	if err != nil {
		return nil, nil, err
	}

	return response, []*MCPTextContent{
		NewMCPTextContent(string(content)),
	}, nil
}

// parseMCPOptionalId parses the textual id, returns 0 if the textual id is empty
func parseMCPOptionalId(field string, value string) (int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return 0, nil
	}

	id, err := utils.StringToInt64(value)

	if err != nil || id <= 0 {
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a positive integer ID in string format")
	}

	return id, nil
}

// parseMCPRequiredId parses the textual id, returns error if the textual id is empty
func parseMCPRequiredId(field string, value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, NewMCPRequiredFieldError(field)
	}

	return parseMCPOptionalId(field, value)
}

// checkMCPStringLength checks whether the text is not blank (if required) and does not exceed the maximum length (counted in characters as the validator of HTTP API does)
func checkMCPStringLength(field string, value string, required bool, maxLength int) error {
	if required && strings.TrimSpace(value) == "" {
		return NewMCPRequiredFieldError(field)
	}

	length := utf8.RuneCountInString(value)

	if maxLength > 0 && length > maxLength {
		return NewMCPFieldTooLongError(field, maxLength, length)
	}

	return nil
}

// parseMCPAmount parses the textual decimal amount (e.g. 12.34) and checks whether it is in the supported range
func parseMCPAmount(field string, value string, required bool) (int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		if required {
			return 0, NewMCPRequiredFieldError(field)
		}

		return 0, nil
	}

	amount, err := utils.ParseAmount(value)

	if err != nil {
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a decimal number with at most 2 decimal places (e.g. 12.34)")
	}

	if amount < models.MinimumTransactionAmount || amount > models.MaximumTransactionAmount {
		err := NewMCPToolError(MCPToolErrorCodeOutOfRange, field, "\""+field+"\" is out of the supported range")
		err.Value = value
		return 0, err
	}

	return amount, nil
}

// parseMCPTime parses the time in RFC 3339 format
func parseMCPTime(field string, value string, required bool) (*time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		if required {
			return nil, NewMCPRequiredFieldError(field)
		}

		return nil, nil
	}

	t, err := utils.ParseFromLongDateTimeWithTimezoneRFC3339Format(value)

	if err != nil {
		return nil, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a time in RFC 3339 format (e.g. 2023-01-01T12:00:00Z or 2023-01-01T12:00:00+08:00)")
	}

	return &t, nil
}

// formatMCPTransactionTime returns the transaction time in RFC 3339 format in the transaction timezone
func formatMCPTransactionTime(transactionTime int64, timezoneUtcOffset int16) string {
	transactionUnixTime := utils.GetUnixTimeFromTransactionTime(transactionTime)
	transactionTimeZone := time.FixedZone("Transaction Timezone", int(timezoneUtcOffset)*60)
	return utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(transactionUnixTime, transactionTimeZone)
}

// formatMCPUnixTime returns the unix time in RFC 3339 format in UTC
func formatMCPUnixTime(unixTime int64) string {
	return utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(unixTime, time.UTC)
}

// getMCPAccountCategoryName returns the textual account category
func getMCPAccountCategoryName(category models.AccountCategory) string {
	if name, exists := mcpAccountCategoryNameMap[category]; exists {
		return name
	}

	return ""
}

// parseMCPAccountCategory returns the account category by the textual account category
func parseMCPAccountCategory(field string, value string) (models.AccountCategory, error) {
	for category, name := range mcpAccountCategoryNameMap {
		if name == value {
			return category, nil
		}
	}

	return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of cash, checking_account, savings_account, credit_card, virtual, debt, receivables, certificate_of_deposit, investment")
}

// getMCPCategoryTypeName returns the textual transaction category type
func getMCPCategoryTypeName(categoryType models.TransactionCategoryType) string {
	switch categoryType {
	case models.CATEGORY_TYPE_INCOME:
		return mcpCategoryTypeIncome
	case models.CATEGORY_TYPE_EXPENSE:
		return mcpCategoryTypeExpense
	case models.CATEGORY_TYPE_TRANSFER:
		return mcpCategoryTypeTransfer
	default:
		return ""
	}
}

// parseMCPCategoryType returns the transaction category type by the textual transaction category type
func parseMCPCategoryType(field string, value string) (models.TransactionCategoryType, error) {
	switch value {
	case mcpCategoryTypeIncome:
		return models.CATEGORY_TYPE_INCOME, nil
	case mcpCategoryTypeExpense:
		return models.CATEGORY_TYPE_EXPENSE, nil
	case mcpCategoryTypeTransfer:
		return models.CATEGORY_TYPE_TRANSFER, nil
	default:
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of income, expense, transfer")
	}
}

// getMCPTransactionTypeName returns the textual transaction type by the transaction type stored in database
func getMCPTransactionTypeName(transactionDbType models.TransactionDbType) string {
	switch transactionDbType {
	case models.TRANSACTION_DB_TYPE_INCOME:
		return transactionTypeIncome
	case models.TRANSACTION_DB_TYPE_EXPENSE:
		return transactionTypeExpense
	case models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.TRANSACTION_DB_TYPE_TRANSFER_IN:
		return transactionTypeTransfer
	case models.TRANSACTION_DB_TYPE_MODIFY_BALANCE:
		return transactionTypeModifyBalance
	default:
		return ""
	}
}

// getMCPCategoryTypeByTransactionType returns the transaction category type which can be used by the textual transaction type
func getMCPCategoryTypeByTransactionType(transactionType string) models.TransactionCategoryType {
	switch transactionType {
	case transactionTypeIncome:
		return models.CATEGORY_TYPE_INCOME
	case transactionTypeExpense:
		return models.CATEGORY_TYPE_EXPENSE
	case transactionTypeTransfer:
		return models.CATEGORY_TYPE_TRANSFER
	default:
		return 0
	}
}
