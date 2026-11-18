package mcp

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strconv"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Limits and formats of exporting transactions through MCP
const (
	mcpExportDefaultMaxRows = 500
	mcpExportMaxRows        = 2000

	mcpExportFormatCSV  = "csv"
	mcpExportFormatJSON = "json"

	mcpExportMultiValueSeparator = "; "
)

// mcpExportCSVColumns contains the columns of exported CSV text
var mcpExportCSVColumns = []string{
	"transaction_id",
	"type",
	"time",
	"utc_offset_minutes",
	"amount",
	"currency",
	"account_id",
	"account_name",
	"destination_amount",
	"destination_currency",
	"destination_account_id",
	"destination_account_name",
	"category_id",
	"category_name",
	"primary_category_name",
	"tag_ids",
	"tags",
	"comment",
	"attachment_ids",
	"attachment_sha256",
	"attachment_file_names",
	"references",
}

// MCPExportTransactionsRequest represents all parameters of the export transactions request
type MCPExportTransactionsRequest struct {
	StartTime      string `json:"start_time" jsonschema:"format=date-time" jsonschema_description:"Start time in RFC 3339 format (e.g. 2024-01-01T00:00:00Z)"`
	EndTime        string `json:"end_time" jsonschema:"format=date-time" jsonschema_description:"End time in RFC 3339 format (e.g. 2024-12-31T23:59:59Z)"`
	Type           string `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer,enum=balance_modification" jsonschema_description:"Transaction type to filter by (optional)"`
	AccountId      string `json:"account_id,omitempty" jsonschema_description:"Account ID to filter by, sub-accounts of a parent account are included, transfers from or to the account are included (optional)"`
	CategoryId     string `json:"category_id,omitempty" jsonschema_description:"Primary or secondary category ID to filter by, sub-categories of a primary category are included (optional)"`
	ReferenceValue string `json:"reference_value,omitempty" jsonschema_description:"Only export transactions which have an evidence / source reference with exactly this value (optional)"`
	ReferenceType  string `json:"reference_type,omitempty" jsonschema:"enum=supplier,enum=invoice_number,enum=receipt_number,enum=external_source_id,enum=source_document,enum=source_row" jsonschema_description:"Reference type of reference_value (optional)"`
	Format         string `json:"format,omitempty" jsonschema:"enum=csv,enum=json" jsonschema_description:"Export format (optional, default: csv). csv returns RFC 4180 CSV text in the csv field (multiple values in one cell are separated by \"; \", references are formatted as type=value), json returns the full transactions in the transactions field"`
	MaxRows        int32  `json:"max_rows,omitempty" jsonschema:"default=500" jsonschema_description:"Maximum count of exported transactions (optional, default: 500, maximum is reported by get_server_capabilities); truncated is true if more transactions match"`
}

// MCPExportTransactionsResponse represents the response structure for export transactions
type MCPExportTransactionsResponse struct {
	Format       string                  `json:"format" jsonschema:"enum=csv,enum=json" jsonschema_description:"Export format"`
	RowCount     int                     `json:"row_count" jsonschema_description:"Count of exported transactions"`
	MaxRows      int32                   `json:"max_rows" jsonschema_description:"Applied maximum count of exported transactions"`
	Truncated    bool                    `json:"truncated" jsonschema_description:"Whether more transactions match the filters than were exported (narrow the time range to export the rest)"`
	Order        string                  `json:"order" jsonschema_description:"Order of the exported transactions (always time_desc)"`
	Columns      []string                `json:"columns,omitempty" jsonschema_description:"Columns of the CSV text (only for csv format)"`
	CSV          string                  `json:"csv,omitempty" jsonschema_description:"Exported CSV text with header row (only for csv format)"`
	Transactions []*MCPTransactionDetail `json:"transactions,omitempty" jsonschema_description:"Exported transactions (only for json format)"`
}

type mcpExportTransactionsToolHandler struct{}

var MCPExportTransactionsToolHandler = &mcpExportTransactionsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpExportTransactionsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_EXPORT_TRANSACTIONS
}

// Name returns the name of the MCP tool
func (h *mcpExportTransactionsToolHandler) Name() string {
	return "export_transactions"
}

// Description returns the description of the MCP tool
func (h *mcpExportTransactionsToolHandler) Description() string {
	return "Export filtered transactions as machine-readable CSV (default) or JSON text for reconciliation and reporting, including transaction IDs, accounts, categories, tags, descriptions, attachment IDs with their SHA-256 checksums recorded at upload and original file names, and evidence references. Bounded by max_rows; truncated reports whether more transactions match."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpExportTransactionsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpExportTransactionsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpExportTransactionsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpExportTransactionsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPExportTransactionsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpExportTransactionsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPExportTransactionsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpExportTransactionsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPExportTransactionsRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	format := strings.TrimSpace(request.Format)

	if format == "" {
		format = mcpExportFormatCSV
	} else if format != mcpExportFormatCSV && format != mcpExportFormatJSON {
		return nil, nil, NewMCPInvalidFieldError("format", request.Format, "\"format\" must be one of csv, json")
	}

	maxRows := request.MaxRows

	if maxRows <= 0 {
		maxRows = mcpExportDefaultMaxRows
	} else if maxRows > mcpExportMaxRows {
		err := NewMCPToolError(MCPToolErrorCodeOutOfRange, "max_rows", "\"max_rows\" must not exceed "+strconv.Itoa(mcpExportMaxRows))
		limit := int64(mcpExportMaxRows)
		actual := int64(maxRows)
		err.Limit = &limit
		err.Actual = &actual
		return nil, nil, err
	}

	maxTime, err := parseMCPTime("end_time", request.EndTime, true)

	if err != nil {
		return nil, nil, err
	}

	minTime, err := parseMCPTime("start_time", request.StartTime, true)

	if err != nil {
		return nil, nil, err
	}

	transactionType, err := mcpTxnParseQueryTransactionType("type", request.Type)

	if err != nil {
		return nil, nil, err
	}

	referenceType := strings.TrimSpace(request.ReferenceType)
	referenceValue := strings.TrimSpace(request.ReferenceValue)

	if referenceType != "" && !models.IsValidTransactionReferenceType(referenceType) {
		return nil, nil, NewMCPInvalidFieldError("reference_type", request.ReferenceType, "\"reference_type\" must be one of "+mcpReferenceTypesDescription)
	}

	if referenceType != "" && referenceValue == "" {
		return nil, nil, NewMCPRequiredFieldError("reference_value")
	}

	uid := user.Uid
	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(maxTime.Unix())
	minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(minTime.Unix())

	var filterAccountIds []int64
	var filterCategoryIds []int64

	if request.AccountId != "" {
		allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

		if err != nil {
			log.Errorf(c, "[export_transactions_tool_handler.Handle] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		filterAccountIds, err = mcpTxnGetAccountAndSubAccountIds(allAccounts, "account_id", request.AccountId)

		if err != nil {
			return nil, nil, err
		}
	}

	if request.CategoryId != "" {
		allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

		if err != nil {
			log.Errorf(c, "[export_transactions_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		filterCategoryIds, err = mcpTxnGetCategoryAndSubCategoryIds(allCategories, "category_id", request.CategoryId)

		if err != nil {
			return nil, nil, err
		}
	}

	var transactions []*models.Transaction

	if referenceValue != "" {
		_, transactions, err = mcpTxnQueryTransactionsByReference(c, uid, referenceType, referenceValue, maxTransactionTime, minTransactionTime, transactionType, filterAccountIds, filterCategoryIds, 1, maxRows+1, services)

		if err != nil {
			return nil, nil, err
		}
	} else {
		transactions, err = services.GetTransactionService().GetTransactionsByMaxTimeUpToCount(c, uid, maxTransactionTime, minTransactionTime, transactionType, filterCategoryIds, filterAccountIds, nil, false, "", "", core.MATCH_MODE_DEFAULT, false, 1, maxRows, pageCountForLoadTransactions, true, true)

		if err != nil {
			log.Errorf(c, "[export_transactions_tool_handler.Handle] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		transactions, err = mcpExportReplaceIncomingTransfers(c, uid, transactions, services)

		if err != nil {
			return nil, nil, err
		}
	}

	truncated := len(transactions) > int(maxRows)

	if truncated {
		transactions = transactions[:maxRows]
	}

	detailData, err := mcpTxnLoadDetailData(c, uid, transactions, currentConfig, services)

	if err != nil {
		return nil, nil, err
	}

	details := make([]*MCPTransactionDetail, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		details = append(details, mcpTxnBuildDetail(transactions[i], detailData, nil, true))
	}

	response := &MCPExportTransactionsResponse{
		Format:    format,
		RowCount:  len(details),
		MaxRows:   maxRows,
		Truncated: truncated,
		Order:     "time_desc",
	}

	if format == mcpExportFormatJSON {
		response.Transactions = details
	} else {
		csvText, err := mcpExportBuildCSV(details)

		if err != nil {
			log.Errorf(c, "[export_transactions_tool_handler.Handle] failed to build csv for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		response.Columns = mcpExportCSVColumns
		response.CSV = csvText
	}

	return newMCPJSONTextResult(response)
}

// mcpTxnParseQueryTransactionType returns the transaction type filter of the textual transaction type (0 if not set)
func mcpTxnParseQueryTransactionType(field string, value string) (models.TransactionType, error) {
	switch value {
	case "":
		return 0, nil
	case transactionTypeExpense:
		return models.TRANSACTION_TYPE_EXPENSE, nil
	case transactionTypeIncome:
		return models.TRANSACTION_TYPE_INCOME, nil
	case transactionTypeTransfer:
		return models.TRANSACTION_TYPE_TRANSFER, nil
	case transactionTypeModifyBalance:
		return models.TRANSACTION_TYPE_MODIFY_BALANCE, nil
	default:
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be one of income, expense, transfer, balance_modification")
	}
}

// mcpExportReplaceIncomingTransfers replaces the incoming sides of transfers with their outgoing sides (which hold the transfer details), duplicated transfers are removed and the order is kept
func mcpExportReplaceIncomingTransfers(c *core.WebContext, uid int64, transactions []*models.Transaction, services MCPAvailableServices) ([]*models.Transaction, error) {
	relatedIds := make([]int64, 0)

	for i := 0; i < len(transactions); i++ {
		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			relatedIds = append(relatedIds, transactions[i].RelatedId)
		}
	}

	if len(relatedIds) < 1 {
		return transactions, nil
	}

	relatedTransactions, err := services.GetTransactionService().GetTransactionsByTransactionIds(c, uid, relatedIds)

	if err != nil {
		log.Errorf(c, "[export_transactions_tool_handler.mcpExportReplaceIncomingTransfers] failed to get related transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, err
	}

	relatedTransactionMap := services.GetTransactionService().GetTransactionMapByList(relatedTransactions)
	result := make([]*models.Transaction, 0, len(transactions))
	transactionIdExists := make(map[int64]bool, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			transaction = relatedTransactionMap[transaction.RelatedId]

			if transaction == nil {
				continue
			}
		}

		if transactionIdExists[transaction.TransactionId] {
			continue
		}

		transactionIdExists[transaction.TransactionId] = true
		result = append(result, transaction)
	}

	return result, nil
}

// mcpExportBuildCSV returns the RFC 4180 CSV text (with header row) of the transaction details
func mcpExportBuildCSV(details []*MCPTransactionDetail) (string, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write(mcpExportCSVColumns); err != nil {
		return "", err
	}

	for i := 0; i < len(details); i++ {
		if err := writer.Write(mcpExportBuildCSVRecord(details[i])); err != nil {
			return "", err
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// mcpExportBuildCSVRecord returns the CSV record of the transaction detail, the order of values is the same as mcpExportCSVColumns
func mcpExportBuildCSVRecord(detail *MCPTransactionDetail) []string {
	tagIds := make([]string, 0, len(detail.Tags))
	tagNames := make([]string, 0, len(detail.Tags))

	for i := 0; i < len(detail.Tags); i++ {
		tagIds = append(tagIds, detail.Tags[i].Id)
		tagNames = append(tagNames, detail.Tags[i].Name)
	}

	attachmentIds := make([]string, 0, len(detail.Attachments))
	attachmentChecksums := make([]string, 0, len(detail.Attachments))
	attachmentFileNames := make([]string, 0, len(detail.Attachments))

	for i := 0; i < len(detail.Attachments); i++ {
		attachmentIds = append(attachmentIds, detail.Attachments[i].Id)
		attachmentChecksums = append(attachmentChecksums, detail.Attachments[i].SHA256)
		attachmentFileNames = append(attachmentFileNames, detail.Attachments[i].OriginalFileName)
	}

	references := make([]string, 0, len(detail.References))

	for i := 0; i < len(detail.References); i++ {
		references = append(references, detail.References[i].Type+"="+detail.References[i].Value)
	}

	return []string{
		detail.Id,
		detail.Type,
		detail.Time,
		strconv.Itoa(int(detail.UtcOffsetMinutes)),
		detail.Amount,
		detail.Currency,
		detail.AccountId,
		mcpExportSanitizeCSVTextCell(detail.AccountName),
		detail.DestinationAmount,
		detail.DestinationCurrency,
		detail.DestinationAccountId,
		mcpExportSanitizeCSVTextCell(detail.DestinationAccountName),
		detail.CategoryId,
		mcpExportSanitizeCSVTextCell(detail.CategoryName),
		mcpExportSanitizeCSVTextCell(detail.PrimaryCategoryName),
		strings.Join(tagIds, mcpExportMultiValueSeparator),
		mcpExportSanitizeCSVTextCell(strings.Join(tagNames, mcpExportMultiValueSeparator)),
		mcpExportSanitizeCSVTextCell(detail.Comment),
		strings.Join(attachmentIds, mcpExportMultiValueSeparator),
		strings.Join(attachmentChecksums, mcpExportMultiValueSeparator),
		mcpExportSanitizeCSVTextCell(strings.Join(attachmentFileNames, mcpExportMultiValueSeparator)),
		mcpExportSanitizeCSVTextCell(strings.Join(references, mcpExportMultiValueSeparator)),
	}
}

// mcpExportSanitizeCSVTextCell prefixes a single quote to free text which would be interpreted as a formula by spreadsheet applications (CSV injection),
// quoting of commas, quotes and line breaks is done by the CSV writer
func mcpExportSanitizeCSVTextCell(value string) string {
	if value == "" {
		return value
	}

	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}

	return value
}
