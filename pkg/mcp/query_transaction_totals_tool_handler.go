package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPQueryTransactionTotalsRequest represents all parameters of the query transaction totals request
type MCPQueryTransactionTotalsRequest struct {
	StartTime   string   `json:"start_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Start time (inclusive) in RFC 3339 format (optional, default is from the earliest transaction)"`
	EndTime     string   `json:"end_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"End time (inclusive) in RFC 3339 format (optional, default is up to now)"`
	GroupBy     string   `json:"group_by,omitempty" jsonschema:"enum=account,enum=category,enum=primary_category,enum=tag,enum=currency" jsonschema_description:"How to group the totals (optional, default is category). Every group is further split by currency"`
	AccountIds  []string `json:"account_ids,omitempty" jsonschema_description:"Only include transactions of these accounts (optional, a parent account ID includes all its sub-accounts)"`
	CategoryIds []string `json:"category_ids,omitempty" jsonschema_description:"Only include transactions of these categories (optional, primary or secondary category IDs)"`
	TagIds      []string `json:"tag_ids,omitempty" jsonschema_description:"Only include transactions which have any of these tags (optional, when grouping by tag, only these tags are returned as groups)"`
	Keyword     string   `json:"keyword,omitempty" jsonschema_description:"Only include transactions whose description contains this keyword (optional)"`
}

// MCPQueryTransactionTotalsResponse represents the response structure for query transaction totals
type MCPQueryTransactionTotalsResponse struct {
	StartTime        string                      `json:"start_time,omitempty" jsonschema_description:"Start time of the range"`
	EndTime          string                      `json:"end_time,omitempty" jsonschema_description:"End time of the range"`
	GroupBy          string                      `json:"group_by" jsonschema_description:"How the totals are grouped"`
	Items            []*MCPTransactionTotalsItem `json:"items" jsonschema_description:"Totals of every group and currency"`
	TotalsByCurrency []*MCPTransactionTotalsItem `json:"totals_by_currency" jsonschema_description:"Overall totals of all matched transactions by currency (each transaction is counted once, even when grouping by tag)"`
	Note             string                      `json:"note,omitempty" jsonschema_description:"Notes about the semantics of the totals"`
}

type mcpQueryTransactionTotalsToolHandler struct{}

var MCPQueryTransactionTotalsToolHandler = &mcpQueryTransactionTotalsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpQueryTransactionTotalsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_QUERY_TRANSACTION_TOTALS
}

// Name returns the name of the MCP tool
func (h *mcpQueryTransactionTotalsToolHandler) Name() string {
	return "query_transaction_totals"
}

// Description returns the description of the MCP tool
func (h *mcpQueryTransactionTotalsToolHandler) Description() string {
	return "Query activity totals (income, expense, transfer in / out and net) over a date range, grouped by account, category, primary category, tag or currency, using the same statistics as the web interface. Amounts are in the currency of each transaction's account, so every group is split by currency. Balance modifications (opening balances) are not included, use query_account_balances_as_of for balances."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpQueryTransactionTotalsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpQueryTransactionTotalsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpQueryTransactionTotalsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpQueryTransactionTotalsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryTransactionTotalsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpQueryTransactionTotalsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryTransactionTotalsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpQueryTransactionTotalsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPQueryTransactionTotalsRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	groupBy := request.GroupBy

	if groupBy == "" {
		groupBy = mcpTotalsGroupByCategory
	}

	if groupBy != mcpTotalsGroupByAccount && groupBy != mcpTotalsGroupByCategory && groupBy != mcpTotalsGroupByPrimaryCategory && groupBy != mcpTotalsGroupByTag && groupBy != mcpTotalsGroupByCurrency {
		return nil, nil, NewMCPInvalidFieldError("group_by", request.GroupBy, "\"group_by\" must be one of account, category, primary_category, tag, currency")
	}

	startUnixTime, endUnixTime, timezone, err := mcpReportParseTimeRange(request.StartTime, request.EndTime)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[query_transaction_totals_tool_handler.Handle] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountFilter, err := mcpReportGetAccountFilter(allAccounts, "account_ids", request.AccountIds)

	if err != nil {
		return nil, nil, err
	}

	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[query_transaction_totals_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	categoryMap := services.GetTransactionCategoryService().GetCategoryMapByList(allCategories)
	categoryFilter := make(map[int64]bool, len(request.CategoryIds))

	for i := 0; i < len(request.CategoryIds); i++ {
		categoryId, err := parseMCPRequiredId("category_ids", request.CategoryIds[i])

		if err != nil {
			return nil, nil, err
		}

		if _, exists := categoryMap[categoryId]; !exists {
			return nil, nil, NewMCPNotFoundError("category_ids", "category", request.CategoryIds[i])
		}

		categoryFilter[categoryId] = true
	}

	allTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[query_transaction_totals_tool_handler.Handle] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	tagMap := services.GetTransactionTagService().GetTagMapByList(allTags)
	tagIds := make([]int64, 0, len(request.TagIds))

	for i := 0; i < len(request.TagIds); i++ {
		tagId, err := parseMCPRequiredId("tag_ids", request.TagIds[i])

		if err != nil {
			return nil, nil, err
		}

		if _, exists := tagMap[tagId]; !exists {
			return nil, nil, NewMCPNotFoundError("tag_ids", "tag", request.TagIds[i])
		}

		tagIds = append(tagIds, tagId)
	}

	tagIds = utils.ToUniqueInt64Slice(tagIds)
	var baseTagFilters []*models.TransactionTagFilter

	if len(tagIds) > 0 {
		baseTagFilters = []*models.TransactionTagFilter{
			{
				TagIds: tagIds,
				Type:   models.TRANSACTION_TAG_FILTER_HAS_ANY,
			},
		}
	}

	accountMap := services.GetAccountService().GetAccountMapByList(allAccounts)
	fetchTotalAmounts := func(tagFilters []*models.TransactionTagFilter, noTags bool) ([]*models.TransactionTotalAmount, error) {
		totalAmounts, err := services.GetTransactionService().GetAccountsAndCategoriesTotalInflowAndOutflow(c, uid, startUnixTime, endUnixTime, tagFilters, noTags, request.Keyword, core.MATCH_MODE_DEFAULT, timezone, false)

		if err != nil {
			log.Errorf(c, "[query_transaction_totals_tool_handler.Handle] failed to get transaction totals for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		return totalAmounts, nil
	}

	allTotalAmounts, err := fetchTotalAmounts(baseTagFilters, false)

	if err != nil {
		return nil, nil, err
	}

	totalsAggregator := newMCPTransactionTotalsAggregator()
	groupAggregator := newMCPTransactionTotalsAggregator()

	for _, row := range allTotalAmounts {
		if !mcpTotalsRowMatches(row, accountFilter, categoryFilter, categoryMap) {
			continue
		}

		totalsAggregator.Add("", "", mcpTotalsGetCurrency(row, accountMap), row.Type, row.Amount)

		if groupBy != mcpTotalsGroupByTag {
			groupId, groupName := mcpTotalsGetGroup(groupBy, row, accountMap, categoryMap)
			groupAggregator.Add(groupId, groupName, mcpTotalsGetCurrency(row, accountMap), row.Type, row.Amount)
		}
	}

	response := &MCPQueryTransactionTotalsResponse{
		GroupBy: groupBy,
		Note:    "Amounts are in the currency of each transaction's account and are not converted. Transfers between your own accounts appear as transfer_out on the source account and transfer_in on the destination account.",
	}

	if groupBy == mcpTotalsGroupByTag {
		groupTagIds := tagIds

		if len(groupTagIds) < 1 {
			for i := 0; i < len(allTags); i++ {
				groupTagIds = append(groupTagIds, allTags[i].TagId)
			}
		}

		for _, tagId := range groupTagIds {
			tagTotalAmounts, err := fetchTotalAmounts([]*models.TransactionTagFilter{
				{
					TagIds: []int64{tagId},
					Type:   models.TRANSACTION_TAG_FILTER_HAS_ANY,
				},
			}, false)

			if err != nil {
				return nil, nil, err
			}

			tagName := ""

			if tag, exists := tagMap[tagId]; exists && tag != nil {
				tagName = tag.Name
			}

			for _, row := range tagTotalAmounts {
				if mcpTotalsRowMatches(row, accountFilter, categoryFilter, categoryMap) {
					groupAggregator.Add(utils.Int64ToString(tagId), tagName, mcpTotalsGetCurrency(row, accountMap), row.Type, row.Amount)
				}
			}
		}

		if len(tagIds) < 1 {
			noTagsTotalAmounts, err := fetchTotalAmounts(nil, true)

			if err != nil {
				return nil, nil, err
			}

			for _, row := range noTagsTotalAmounts {
				if mcpTotalsRowMatches(row, accountFilter, categoryFilter, categoryMap) {
					groupAggregator.Add("", mcpTotalsNoTagsGroupName, mcpTotalsGetCurrency(row, accountMap), row.Type, row.Amount)
				}
			}
		}

		response.Note = response.Note + " A transaction with several tags is counted in every one of its tag groups, so the sum of the groups can exceed totals_by_currency."
	}

	if startUnixTime > 0 {
		response.StartTime = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(startUnixTime, timezone)
	}

	if endUnixTime > 0 {
		response.EndTime = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(endUnixTime, timezone)
	}

	response.Items = groupAggregator.Items()
	response.TotalsByCurrency = totalsAggregator.Items()

	return newMCPJSONTextResult(response)
}
