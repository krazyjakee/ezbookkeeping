package mcp

import (
	"encoding/json/v2"
	"reflect"
	"sort"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const pageCountForLoadTransactions = 1000

// MCPQueryTransactionsRequest represents all parameters of the query transactions request
type MCPQueryTransactionsRequest struct {
	StartTime             string   `json:"start_time" jsonschema:"format=date-time" jsonschema_description:"Start time for the query in RFC 3339 format (e.g. 2023-01-01T12:00:00Z)"`
	EndTime               string   `json:"end_time" jsonschema:"format=date-time" jsonschema_description:"End time for the query in RFC 3339 format or (e.g. 2023-01-01T12:00:00Z)"`
	Type                  string   `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer,enum=balance_modification" jsonschema_description:"Transaction type to filter by (income, expense, transfer, balance_modification) (optional)"`
	SecondaryCategoryName string   `json:"category_name,omitempty" jsonschema_description:"Primary or secondary category name to filter transactions by, all categories with this name (and sub-categories of primary categories) are included (optional)"`
	CategoryId            string   `json:"category_id,omitempty" jsonschema_description:"Primary or secondary category ID to filter transactions by, sub-categories of a primary category are included (optional, takes precedence over category_name)"`
	AccountName           string   `json:"account_name,omitempty" jsonschema_description:"Account name to filter transactions by, all accounts with this name (and sub-accounts of parent accounts) are included (optional)"`
	AccountId             string   `json:"account_id,omitempty" jsonschema_description:"Account ID to filter transactions by, sub-accounts of a parent account are included (optional, takes precedence over account_name)"`
	TagIds                []string `json:"tag_ids,omitempty" jsonschema_description:"Tag IDs to filter transactions by (optional)"`
	TagNames              []string `json:"tag_names,omitempty" jsonschema_description:"Tag names to filter transactions by, ambiguous or unknown names are rejected (optional)"`
	TagFilterMode         string   `json:"tag_filter_mode,omitempty" jsonschema:"enum=has_any,enum=has_all,enum=not_has_any,enum=not_has_all" jsonschema_description:"How to match tag_ids / tag_names (optional, default: has_any)"`
	NoTags                bool     `json:"no_tags,omitempty" jsonschema_description:"If true, only transactions without any tag are returned (optional, cannot be used with tag_ids / tag_names)"`
	AmountFilter          string   `json:"amount_filter,omitempty" jsonschema_description:"Amount filter in source account currency (optional): gt:<amount>, lt:<amount>, eq:<amount>, ne:<amount>, bt:<min>:<max> (between), nb:<min>:<max> (not between), e.g. bt:10:20.50"`
	HasAttachments        bool     `json:"has_attachments,omitempty" jsonschema_description:"If true, only transactions with at least one attachment are returned (optional)"`
	Keyword               string   `json:"keyword,omitempty" jsonschema_description:"Keyword to search in transaction description (optional)"`
	MatchMode             string   `json:"match_mode,omitempty" jsonschema:"enum=default,enum=ignore_case" jsonschema_description:"Match mode for keyword search (optional, leave empty for database default setting, ignore_case for case-insensitive search)"`
	Count                 int32    `json:"count,omitempty" jsonschema:"default=100" jsonschema_description:"Maximum number of results to return (default: 100)"`
	Page                  int32    `json:"page,omitempty" jsonschema:"default=1" jsonschema_description:"Page number for pagination (default: 1). Results are always ordered by transaction time descending (newest first), which is the only order supported by ezBookkeeping"`
	ReferenceValue        string   `json:"reference_value,omitempty" jsonschema_description:"Only return transactions which have an evidence / source reference with exactly this value, e.g. an invoice number or external source ID (optional). Can be combined with the time range, type, account and category filters, but not with tag, amount, keyword or attachment filters"`
	ReferenceType         string   `json:"reference_type,omitempty" jsonschema:"enum=supplier,enum=invoice_number,enum=receipt_number,enum=external_source_id,enum=source_document,enum=source_row" jsonschema_description:"Reference type of reference_value (optional, any type if not set)"`
	ResponseFields        string   `json:"response_fields,omitempty" jsonschema_description:"Comma-separated list of optional fields to include in the response (optional, leave empty for all fields; id, type, amount, account_id, category_id and attachment_count are always included; available fields: time, currency, category_name, account_name, comment, tags, geo_location, references)"`
}

// MCPQueryTransactionsResponse represents the response structure for querying transactions
type MCPQueryTransactionsResponse struct {
	TotalCount   int64                 `json:"total_count" jsonschema_description:"Total number of transactions matching the query"`
	CurrentPage  int32                 `json:"current_page" jsonschema_description:"Current page number of the results"`
	TotalPage    int32                 `json:"total_page" jsonschema_description:"Total number of pages available for the query, calculated based on total_count and count"`
	Order        string                `json:"order" jsonschema_description:"Order of the results (always time_desc)"`
	Transactions []*MCPTransactionInfo `json:"transactions" jsonschema_description:"List of transactions matching the query"`
}

// MCPTransactionInfo defines the structure of transaction information
type MCPTransactionInfo struct {
	Id                     string                           `json:"id" jsonschema_description:"Transaction ID"`
	Time                   string                           `json:"time,omitempty" jsonschema_description:"Time of the transaction in RFC 3339 format (e.g. 2023-01-01T12:00:00Z)"`
	Type                   string                           `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer,enum=balance_modification" jsonschema_description:"Transaction type (income, expense, transfer, balance_modification)"`
	Amount                 string                           `json:"amount" jsonschema_description:"Amount of the transaction in the specified currency"`
	Currency               string                           `json:"currency,omitempty" jsonschema_description:"Currency code of the transaction (e.g. USD, EUR)"`
	CategoryId             string                           `json:"category_id,omitempty" jsonschema_description:"Secondary category ID of the transaction"`
	SecondaryCategoryName  string                           `json:"category_name,omitempty" jsonschema_description:"Secondary category name for the transaction"`
	AccountId              string                           `json:"account_id" jsonschema_description:"Account ID of the transaction"`
	AccountName            string                           `json:"account_name,omitempty" jsonschema_description:"Account name for the transaction"`
	DestinationAmount      string                           `json:"destination_amount,omitempty" jsonschema_description:"Destination amount for transfer transactions (optional)"`
	DestinationCurrency    string                           `json:"destination_currency,omitempty" jsonschema_description:"Currency code of the destination amount for transfer transactions (optional)"`
	DestinationAccountId   string                           `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID for transfer transactions (optional)"`
	DestinationAccountName string                           `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name for transfer transactions (optional)"`
	Tags                   []*MCPTransactionDetailTag       `json:"tags,omitempty" jsonschema_description:"Tags of the transaction"`
	Comment                string                           `json:"comment,omitempty" jsonschema_description:"Description of the transaction"`
	GeoLocation            *MCPTransactionDetailGeoLocation `json:"geo_location,omitempty" jsonschema_description:"Geographic location of the transaction"`
	AttachmentCount        int                              `json:"attachment_count" jsonschema_description:"Count of attachments of the transaction"`
	References             []*MCPTransactionReference       `json:"references,omitempty" jsonschema_description:"Structured evidence / source references of the transaction"`
}

type mcpQueryTransactionsToolHandler struct{}

var MCPQueryTransactionsToolHandler = &mcpQueryTransactionsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpQueryTransactionsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_QUERY_TRANSACTIONS
}

// Name returns the name of the MCP tool
func (h *mcpQueryTransactionsToolHandler) Name() string {
	return "query_transactions"
}

// Description returns the description of the MCP tool
func (h *mcpQueryTransactionsToolHandler) Description() string {
	return "Query transactions based on various filters (including evidence references such as invoice numbers or external source IDs) with pagination. Every transaction is returned with its stable ID and references (use get_transaction for full details). Results are ordered by transaction time descending."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpQueryTransactionsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpQueryTransactionsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpQueryTransactionsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpQueryTransactionsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryTransactionsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpQueryTransactionsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryTransactionsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpQueryTransactionsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	var queryTransactionsRequest MCPQueryTransactionsRequest

	if callToolReq.Arguments != nil {
		if err := json.Unmarshal(callToolReq.Arguments, &queryTransactionsRequest); err != nil {
			return nil, nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
		}
	} else {
		return nil, nil, errs.ErrIncompleteOrIncorrectSubmission
	}

	uid := user.Uid
	maxTime, err := parseMCPTime("end_time", queryTransactionsRequest.EndTime, true)

	if err != nil {
		return nil, nil, err
	}

	minTime, err := parseMCPTime("start_time", queryTransactionsRequest.StartTime, true)

	if err != nil {
		return nil, nil, err
	}

	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(maxTime.Unix())
	minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(minTime.Unix())

	if queryTransactionsRequest.Count <= 0 {
		queryTransactionsRequest.Count = 100
	}

	if queryTransactionsRequest.Page <= 0 {
		queryTransactionsRequest.Page = 1
	}

	transactionType := models.TransactionType(byte(0))

	if queryTransactionsRequest.Type == transactionTypeExpense {
		transactionType = models.TRANSACTION_TYPE_EXPENSE
	} else if queryTransactionsRequest.Type == transactionTypeIncome {
		transactionType = models.TRANSACTION_TYPE_INCOME
	} else if queryTransactionsRequest.Type == transactionTypeTransfer {
		transactionType = models.TRANSACTION_TYPE_TRANSFER
	} else if queryTransactionsRequest.Type == transactionTypeModifyBalance {
		transactionType = models.TRANSACTION_TYPE_MODIFY_BALANCE
	} else if queryTransactionsRequest.Type != "" {
		return nil, nil, NewMCPInvalidFieldError("type", queryTransactionsRequest.Type, "\"type\" must be one of income, expense, transfer, balance_modification")
	}

	queryTransactionsRequest.ReferenceValue = strings.TrimSpace(queryTransactionsRequest.ReferenceValue)
	queryTransactionsRequest.ReferenceType = strings.TrimSpace(queryTransactionsRequest.ReferenceType)

	if queryTransactionsRequest.ReferenceType != "" && !models.IsValidTransactionReferenceType(queryTransactionsRequest.ReferenceType) {
		return nil, nil, NewMCPInvalidFieldError("reference_type", queryTransactionsRequest.ReferenceType, "\"reference_type\" must be one of "+mcpReferenceTypesDescription)
	}

	if queryTransactionsRequest.ReferenceType != "" && queryTransactionsRequest.ReferenceValue == "" {
		return nil, nil, NewMCPRequiredFieldError("reference_value")
	}

	amountFilter, err := mcpTxnParseAmountFilter("amount_filter", queryTransactionsRequest.AmountFilter)

	if err != nil {
		return nil, nil, err
	}

	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Warnf(c, "[query_transactions_tool_handler.Handle] get account error, because %s", err.Error())
		return nil, nil, err
	}

	filterAccountIds := make([]int64, 0)

	if queryTransactionsRequest.AccountId != "" {
		filterAccountIds, err = mcpTxnGetAccountAndSubAccountIds(allAccounts, "account_id", queryTransactionsRequest.AccountId)

		if err != nil {
			return nil, nil, err
		}
	} else if queryTransactionsRequest.AccountName != "" {
		filterAccountIds = services.GetAccountService().GetAccountOrSubAccountIdsByAccountName(allAccounts, queryTransactionsRequest.AccountName)

		if len(filterAccountIds) < 1 {
			return nil, nil, NewMCPNotFoundError("account_name", "account", queryTransactionsRequest.AccountName)
		}
	}

	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Warnf(c, "[query_transactions_tool_handler.Handle] get transaction category error, because %s", err.Error())
		return nil, nil, err
	}

	filterCategoryIds := make([]int64, 0)

	if queryTransactionsRequest.CategoryId != "" {
		filterCategoryIds, err = mcpTxnGetCategoryAndSubCategoryIds(allCategories, "category_id", queryTransactionsRequest.CategoryId)

		if err != nil {
			return nil, nil, err
		}
	} else if queryTransactionsRequest.SecondaryCategoryName != "" {
		filterCategoryIds = services.GetTransactionCategoryService().GetCategoryOrSubCategoryIdsByCategoryName(allCategories, queryTransactionsRequest.SecondaryCategoryName)

		if len(filterCategoryIds) < 1 {
			return nil, nil, NewMCPNotFoundError("category_name", "category", queryTransactionsRequest.SecondaryCategoryName)
		}
	}

	allTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Warnf(c, "[query_transactions_tool_handler.Handle] get transaction tag error, because %s", err.Error())
		return nil, nil, err
	}

	tagFilters, err := mcpTxnGetTagFilters(allTags, queryTransactionsRequest.TagIds, queryTransactionsRequest.TagNames, queryTransactionsRequest.TagFilterMode, queryTransactionsRequest.NoTags)

	if err != nil {
		return nil, nil, err
	}

	matchModeType := core.MATCH_MODE_DEFAULT

	if queryTransactionsRequest.MatchMode == "ignore_case" {
		matchModeType = core.MATCH_MODE_IGNORE_CASE
	}

	var totalCount int64
	var transactions []*models.Transaction

	if queryTransactionsRequest.ReferenceValue != "" {
		if len(tagFilters) > 0 || queryTransactionsRequest.NoTags || amountFilter != "" || queryTransactionsRequest.Keyword != "" || queryTransactionsRequest.HasAttachments {
			return nil, nil, NewMCPNotSupportedError("reference_value", "\"reference_value\" can only be combined with the time range, type, account and category filters")
		}

		totalCount, transactions, err = mcpTxnQueryTransactionsByReference(c, uid, queryTransactionsRequest.ReferenceType, queryTransactionsRequest.ReferenceValue, maxTransactionTime, minTransactionTime, transactionType, filterAccountIds, filterCategoryIds, queryTransactionsRequest.Page, queryTransactionsRequest.Count, services)

		if err != nil {
			return nil, nil, err
		}
	} else {
		totalCount, err = services.GetTransactionService().GetTransactionCount(c, uid, maxTransactionTime, minTransactionTime, transactionType, filterCategoryIds, filterAccountIds, tagFilters, queryTransactionsRequest.NoTags, amountFilter, queryTransactionsRequest.Keyword, matchModeType, queryTransactionsRequest.HasAttachments)

		if err != nil {
			log.Errorf(c, "[query_transactions_tool_handler.Handle] failed to get transaction count for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		transactions, err = services.GetTransactionService().GetTransactionsByMaxTimeUpToCount(c, uid, maxTransactionTime, minTransactionTime, transactionType, filterCategoryIds, filterAccountIds, tagFilters, queryTransactionsRequest.NoTags, amountFilter, queryTransactionsRequest.Keyword, matchModeType, queryTransactionsRequest.HasAttachments, queryTransactionsRequest.Page, queryTransactionsRequest.Count, pageCountForLoadTransactions, false, true)

		if err != nil {
			log.Errorf(c, "[query_transactions_tool_handler.Handle] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}
	}

	detailData := &mcpTxnDetailData{
		accounts:    services.GetAccountService().GetAccountMapByList(allAccounts),
		categories:  services.GetTransactionCategoryService().GetCategoryMapByList(allCategories),
		tags:        services.GetTransactionTagService().GetTagMapByList(allTags),
		tagIds:      make(map[int64][]int64),
		attachments: make(map[int64][]*models.TransactionPictureInfo),
	}

	if len(transactions) > 0 {
		transactionIds := make([]int64, 0, len(transactions))

		for i := 0; i < len(transactions); i++ {
			transactionIds = append(transactionIds, transactions[i].TransactionId)
		}

		allTransactionTagIds, err := services.GetTransactionTagService().GetAllTagIdsOfTransactions(c, uid, transactionIds)

		if err != nil {
			log.Errorf(c, "[query_transactions_tool_handler.Handle] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		if allTransactionTagIds != nil {
			detailData.tagIds = allTransactionTagIds
		}

		if currentConfig != nil && currentConfig.EnableTransactionPictures {
			attachments, err := services.GetTransactionPictureService().GetPictureInfosByTransactionIds(c, uid, transactionIds)

			if err != nil {
				log.Errorf(c, "[query_transactions_tool_handler.Handle] failed to get transactions attachments for user \"uid:%d\", because %s", uid, err.Error())
				return nil, nil, err
			}

			if attachments != nil {
				detailData.attachments = attachments
			}
		}

		detailData.references = mcpReferenceLoad(c, uid, transactionIds, services)
	}

	return h.createNewMCPQueryTransactionsResponse(c, &queryTransactionsRequest, transactions, totalCount, detailData)
}

func (h *mcpQueryTransactionsToolHandler) createNewMCPQueryTransactionsResponse(c *core.WebContext, queryTransactionsRequest *MCPQueryTransactionsRequest, transactions []*models.Transaction, totalCount int64, detailData *mcpTxnDetailData) (any, []*MCPTextContent, error) {
	response := MCPQueryTransactionsResponse{
		TotalCount:   totalCount,
		CurrentPage:  queryTransactionsRequest.Page,
		TotalPage:    int32((totalCount + int64(queryTransactionsRequest.Count) - 1) / int64(queryTransactionsRequest.Count)),
		Order:        "time_desc",
		Transactions: make([]*MCPTransactionInfo, 0, len(transactions)),
	}

	filteredFields := make(map[string]bool)

	if queryTransactionsRequest.ResponseFields != "" {
		for _, field := range strings.Split(queryTransactionsRequest.ResponseFields, ",") {
			filteredFields[strings.TrimSpace(field)] = true
		}
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if getMCPTransactionTypeName(transaction.Type) == "" || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			log.Warnf(c, "[query_transactions_tool_handler.createNewMCPQueryTransactionsResponse] encountered transaction with unexpected type \"%d\" for transaction \"id:%d\"", transaction.Type, transaction.TransactionId)
			continue
		}

		detail := mcpTxnBuildDetail(transaction, detailData, nil, false)
		transactionInfo := MCPTransactionInfo{
			Id:                   detail.Id,
			Type:                 detail.Type,
			Amount:               detail.Amount,
			AccountId:            detail.AccountId,
			CategoryId:           detail.CategoryId,
			DestinationAmount:    detail.DestinationAmount,
			DestinationAccountId: detail.DestinationAccountId,
			AttachmentCount:      detail.AttachmentCount,
		}

		if h.isFieldIncluded(filteredFields, "time") {
			transactionInfo.Time = detail.Time
		}

		if h.isFieldIncluded(filteredFields, "currency") {
			transactionInfo.Currency = detail.Currency
			transactionInfo.DestinationCurrency = detail.DestinationCurrency
		}

		if h.isFieldIncluded(filteredFields, "category_name") {
			transactionInfo.SecondaryCategoryName = detail.CategoryName
		}

		if h.isFieldIncluded(filteredFields, "account_name") {
			transactionInfo.AccountName = detail.AccountName
			transactionInfo.DestinationAccountName = detail.DestinationAccountName
		}

		if h.isFieldIncluded(filteredFields, "comment") {
			transactionInfo.Comment = detail.Comment
		}

		if h.isFieldIncluded(filteredFields, "tags") && len(detail.Tags) > 0 {
			transactionInfo.Tags = detail.Tags
		}

		if h.isFieldIncluded(filteredFields, "geo_location") {
			transactionInfo.GeoLocation = detail.GeoLocation
		}

		if h.isFieldIncluded(filteredFields, "references") {
			transactionInfo.References = detail.References
		}

		response.Transactions = append(response.Transactions, &transactionInfo)
	}

	return newMCPJSONTextResult(response)
}

func (h *mcpQueryTransactionsToolHandler) isFieldIncluded(filteredFields map[string]bool, field string) bool {
	if len(filteredFields) == 0 {
		return true
	}

	_, exists := filteredFields[field]
	return exists
}

// mcpTxnGetAccountAndSubAccountIds returns the account id and all its sub-account ids
func mcpTxnGetAccountAndSubAccountIds(allAccounts []*models.Account, field string, textualAccountId string) ([]int64, error) {
	accountId, err := parseMCPRequiredId(field, textualAccountId)

	if err != nil {
		return nil, err
	}

	accountIds := make([]int64, 0)
	found := false

	for i := 0; i < len(allAccounts); i++ {
		if allAccounts[i].AccountId == accountId {
			found = true
			accountIds = append(accountIds, accountId)
		} else if allAccounts[i].ParentAccountId == accountId {
			accountIds = append(accountIds, allAccounts[i].AccountId)
		}
	}

	if !found {
		return nil, NewMCPNotFoundError(field, "account", textualAccountId)
	}

	return accountIds, nil
}

// mcpTxnGetCategoryAndSubCategoryIds returns the category id and all its sub-category ids
func mcpTxnGetCategoryAndSubCategoryIds(allCategories []*models.TransactionCategory, field string, textualCategoryId string) ([]int64, error) {
	categoryId, err := parseMCPRequiredId(field, textualCategoryId)

	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0)
	found := false

	for i := 0; i < len(allCategories); i++ {
		if allCategories[i].CategoryId == categoryId {
			found = true
			categoryIds = append(categoryIds, categoryId)
		} else if allCategories[i].ParentCategoryId == categoryId {
			categoryIds = append(categoryIds, allCategories[i].CategoryId)
		}
	}

	if !found {
		return nil, NewMCPNotFoundError(field, "category", textualCategoryId)
	}

	return categoryIds, nil
}

// mcpTxnGetTagFilters returns the tag filters which are specified by tag ids and tag names
func mcpTxnGetTagFilters(allTags []*models.TransactionTag, tagIds []string, tagNames []string, mode string, noTags bool) ([]*models.TransactionTagFilter, error) {
	if len(tagIds) == 0 && len(tagNames) == 0 {
		return nil, nil
	}

	if noTags {
		return nil, NewMCPInvalidFieldError("no_tags", "true", "\"no_tags\" cannot be used together with \"tag_ids\" or \"tag_names\"")
	}

	filterType := models.TRANSACTION_TAG_FILTER_HAS_ANY

	switch mode {
	case "", "has_any":
		filterType = models.TRANSACTION_TAG_FILTER_HAS_ANY
	case "has_all":
		filterType = models.TRANSACTION_TAG_FILTER_HAS_ALL
	case "not_has_any":
		filterType = models.TRANSACTION_TAG_FILTER_NOT_HAS_ANY
	case "not_has_all":
		filterType = models.TRANSACTION_TAG_FILTER_NOT_HAS_ALL
	default:
		return nil, NewMCPInvalidFieldError("tag_filter_mode", mode, "\"tag_filter_mode\" must be one of has_any, has_all, not_has_any, not_has_all")
	}

	resolver := newMCPEntityResolver[*models.TransactionTag]("tag",
		func(tag *models.TransactionTag) int64 { return tag.TagId },
		func(tag *models.TransactionTag) string { return tag.Name },
		nil)

	resolvedTagIds := make([]int64, 0, len(tagIds)+len(tagNames))

	for i := 0; i < len(tagIds); i++ {
		tag, err := resolver.Resolve(allTags, "tag_ids", tagIds[i], "tag_names", "")

		if err != nil {
			return nil, err
		}

		resolvedTagIds = append(resolvedTagIds, tag.TagId)
	}

	for i := 0; i < len(tagNames); i++ {
		tag, err := resolver.Resolve(allTags, "tag_ids", "", "tag_names", tagNames[i])

		if err != nil {
			return nil, err
		}

		resolvedTagIds = append(resolvedTagIds, tag.TagId)
	}

	return []*models.TransactionTagFilter{
		{
			TagIds: utils.ToUniqueInt64Slice(resolvedTagIds),
			Type:   filterType,
		},
	}, nil
}

// mcpTxnQueryReferenceMaxMatches is the maximum count of transactions which are matched by one reference value
const mcpTxnQueryReferenceMaxMatches = 1000

// mcpTxnQueryTransactionsByReference returns the total count and the transactions of the requested page which have the given reference and match the other filters, ordered by transaction time descending
func mcpTxnQueryTransactionsByReference(c *core.WebContext, uid int64, referenceType string, referenceValue string, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, accountIds []int64, categoryIds []int64, page int32, count int32, services MCPAvailableServices) (int64, []*models.Transaction, error) {
	transactionIds, err := mcpGetTransactionReferenceService(services).GetTransactionIdsByReference(c, uid, referenceType, referenceValue, mcpTxnQueryReferenceMaxMatches)

	if err != nil {
		log.Errorf(c, "[query_transactions_tool_handler.mcpTxnQueryTransactionsByReference] failed to get transaction ids by reference for user \"uid:%d\", because %s", uid, err.Error())
		return 0, nil, err
	}

	if len(transactionIds) < 1 {
		return 0, make([]*models.Transaction, 0), nil
	}

	allTransactions, err := services.GetTransactionService().GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[query_transactions_tool_handler.mcpTxnQueryTransactionsByReference] failed to get transactions by ids for user \"uid:%d\", because %s", uid, err.Error())
		return 0, nil, err
	}

	filteredTransactions := mcpTxnFilterTransactions(allTransactions, maxTransactionTime, minTransactionTime, transactionType, accountIds, categoryIds)
	totalCount := int64(len(filteredTransactions))
	start := int64(page-1) * int64(count)

	if start >= totalCount {
		return totalCount, make([]*models.Transaction, 0), nil
	}

	end := start + int64(count)

	if end > totalCount {
		end = totalCount
	}

	return totalCount, filteredTransactions[start:end], nil
}

// mcpTxnFilterTransactions returns the transactions (incoming sides of transfers excluded) which match the time range, type, accounts and categories, ordered by transaction time descending
func mcpTxnFilterTransactions(transactions []*models.Transaction, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, accountIds []int64, categoryIds []int64) []*models.Transaction {
	accountIdSet := make(map[int64]bool, len(accountIds))
	categoryIdSet := make(map[int64]bool, len(categoryIds))

	for i := 0; i < len(accountIds); i++ {
		accountIdSet[accountIds[i]] = true
	}

	for i := 0; i < len(categoryIds); i++ {
		categoryIdSet[categoryIds[i]] = true
	}

	result := make([]*models.Transaction, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			continue
		}

		if maxTransactionTime > 0 && transaction.TransactionTime > maxTransactionTime {
			continue
		}

		if minTransactionTime > 0 && transaction.TransactionTime < minTransactionTime {
			continue
		}

		if transactionType > 0 {
			transactionDbType, err := transactionType.ToTransactionDbType()

			if err != nil || transaction.Type != transactionDbType {
				continue
			}
		}

		if len(accountIdSet) > 0 && !accountIdSet[transaction.AccountId] && !(transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && accountIdSet[transaction.RelatedAccountId]) {
			continue
		}

		if len(categoryIdSet) > 0 && !categoryIdSet[transaction.CategoryId] {
			continue
		}

		result = append(result, transaction)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TransactionTime != result[j].TransactionTime {
			return result[i].TransactionTime > result[j].TransactionTime
		}

		return result[i].TransactionId > result[j].TransactionId
	})

	return result
}
