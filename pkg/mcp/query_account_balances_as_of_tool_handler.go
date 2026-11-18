package mcp

import (
	"math/big"
	"reflect"
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPQueryAccountBalancesAsOfRequest represents all parameters of the query account balances as of request
type MCPQueryAccountBalancesAsOfRequest struct {
	AsOf          string   `json:"as_of" jsonschema:"format=date-time" jsonschema_description:"Point in time in RFC 3339 format (e.g. 2024-03-31T23:59:59Z), all transactions at or before this time are included"`
	AccountIds    []string `json:"account_ids,omitempty" jsonschema_description:"IDs of the accounts to return (optional, a parent account ID includes all its sub-accounts, default is all accounts)"`
	IncludeHidden bool     `json:"include_hidden,omitempty" jsonschema_description:"Whether to include hidden accounts when account_ids is not set (optional)"`
}

// MCPAccountBalanceAsOfItem represents the balance of an account at a point in time
type MCPAccountBalanceAsOfItem struct {
	AccountId            string `json:"account_id" jsonschema_description:"Account ID"`
	AccountName          string `json:"account_name" jsonschema_description:"Account name"`
	ParentAccountId      string `json:"parent_account_id,omitempty" jsonschema_description:"Parent account ID (only for sub-accounts)"`
	Category             string `json:"category" jsonschema_description:"Account category"`
	Currency             string `json:"currency" jsonschema_description:"Currency code of the account"`
	Hidden               bool   `json:"hidden,omitzero" jsonschema_description:"Whether the account is hidden"`
	IsLiability          bool   `json:"is_liability" jsonschema_description:"Whether the account is a liability account (credit card, debt)"`
	Balance              string `json:"balance" jsonschema_description:"Displayed balance as of the time (outstanding balance for liability accounts, positive means money owed)"`
	SignedBalance        string `json:"signed_balance" jsonschema_description:"Signed accounting balance as of the time (negative means money owed for liability accounts)"`
	OutstandingBalance   string `json:"outstanding_balance,omitempty" jsonschema_description:"Outstanding balance as of the time (only for liability accounts, positive means money owed)"`
	CurrentSignedBalance string `json:"current_signed_balance" jsonschema_description:"Current signed accounting balance of the account (for comparison)"`
}

// MCPAccountBalanceCurrencyTotal represents the total balances of all returned accounts in one currency
type MCPAccountBalanceCurrencyTotal struct {
	Currency    string `json:"currency" jsonschema_description:"Currency code"`
	Assets      string `json:"assets" jsonschema_description:"Total signed balance of asset accounts"`
	Liabilities string `json:"liabilities" jsonschema_description:"Total outstanding balance of liability accounts (positive means money owed)"`
	Net         string `json:"net" jsonschema_description:"Net balance (assets - liabilities)"`
}

// MCPQueryAccountBalancesAsOfResponse represents the response structure for query account balances as of
type MCPQueryAccountBalancesAsOfResponse struct {
	AsOf             string                            `json:"as_of" jsonschema_description:"The point in time of the balances in RFC 3339 format"`
	Accounts         []*MCPAccountBalanceAsOfItem      `json:"accounts" jsonschema_description:"Balances of the accounts which can hold transactions (parent accounts are represented by their sub-accounts)"`
	TotalsByCurrency []*MCPAccountBalanceCurrencyTotal `json:"totals_by_currency" jsonschema_description:"Total balances of the returned accounts grouped by currency"`
}

type mcpQueryAccountBalancesAsOfToolHandler struct{}

var MCPQueryAccountBalancesAsOfToolHandler = &mcpQueryAccountBalancesAsOfToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpQueryAccountBalancesAsOfToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_QUERY_ACCOUNT_BALANCES_AS_OF
}

// Name returns the name of the MCP tool
func (h *mcpQueryAccountBalancesAsOfToolHandler) Name() string {
	return "query_account_balances_as_of"
}

// Description returns the description of the MCP tool
func (h *mcpQueryAccountBalancesAsOfToolHandler) Description() string {
	return "Query the balance of every account as of a point in time (including opening balances and balance modifications), computed from transactions in the same way as the asset trends statistics. Returns account IDs, signed and displayed (outstanding) balances, and totals grouped by currency."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpQueryAccountBalancesAsOfToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpQueryAccountBalancesAsOfToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpQueryAccountBalancesAsOfToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpQueryAccountBalancesAsOfToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryAccountBalancesAsOfRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpQueryAccountBalancesAsOfToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPQueryAccountBalancesAsOfResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpQueryAccountBalancesAsOfToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPQueryAccountBalancesAsOfRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	asOf, err := parseMCPTime("as_of", request.AsOf, true)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[query_account_balances_as_of_tool_handler.Handle] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountFilter, err := mcpReportGetAccountFilter(allAccounts, "account_ids", request.AccountIds)

	if err != nil {
		return nil, nil, err
	}

	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(asOf.Unix())
	minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(asOf.Unix())
	dailyBalances, err := services.GetTransactionService().GetAllAccountsDailyOpeningAndClosingBalance(c, uid, maxTransactionTime, minTransactionTime, asOf.Location())

	if err != nil {
		log.Errorf(c, "[query_account_balances_as_of_tool_handler.Handle] failed to get account balances as of \"%d\" for user \"uid:%d\", because %s", asOf.Unix(), uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	balances := mcpReportGetLatestClosingBalances(dailyBalances)
	response := &MCPQueryAccountBalancesAsOfResponse{
		AsOf:             utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(asOf.Unix(), asOf.Location()),
		Accounts:         make([]*MCPAccountBalanceAsOfItem, 0, len(allAccounts)),
		TotalsByCurrency: make([]*MCPAccountBalanceCurrencyTotal, 0),
	}

	assetTotals := make(map[string]*big.Int)
	liabilityTotals := make(map[string]*big.Int)
	currencies := make([]string, 0)

	for i := 0; i < len(allAccounts); i++ {
		account := allAccounts[i]

		if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		if len(accountFilter) > 0 {
			if !accountFilter[account.AccountId] {
				continue
			}
		} else if account.Hidden && !request.IncludeHidden {
			continue
		}

		signedBalance := balances[account.AccountId]

		if signedBalance == nil {
			signedBalance = big.NewInt(0)
		}

		displayBalance, signed, outstandingBalance := mcpFormatAccountBalances(account.Category, signedBalance)
		item := &MCPAccountBalanceAsOfItem{
			AccountId:            utils.Int64ToString(account.AccountId),
			AccountName:          account.Name,
			Category:             getMCPAccountCategoryName(account.Category),
			Currency:             account.Currency,
			Hidden:               account.Hidden,
			IsLiability:          account.Category.IsLiability(),
			Balance:              displayBalance,
			SignedBalance:        signed,
			OutstandingBalance:   outstandingBalance,
			CurrentSignedBalance: utils.FormatAmount(account.Balance),
		}

		if account.ParentAccountId > 0 {
			item.ParentAccountId = utils.Int64ToString(account.ParentAccountId)
		}

		response.Accounts = append(response.Accounts, item)

		if _, exists := assetTotals[account.Currency]; !exists {
			assetTotals[account.Currency] = big.NewInt(0)
			liabilityTotals[account.Currency] = big.NewInt(0)
			currencies = append(currencies, account.Currency)
		}

		if account.Category.IsLiability() {
			liabilityTotals[account.Currency].Sub(liabilityTotals[account.Currency], signedBalance)
		} else {
			assetTotals[account.Currency].Add(assetTotals[account.Currency], signedBalance)
		}
	}

	sort.Strings(currencies)

	for _, currency := range currencies {
		net := new(big.Int).Sub(assetTotals[currency], liabilityTotals[currency])
		response.TotalsByCurrency = append(response.TotalsByCurrency, &MCPAccountBalanceCurrencyTotal{
			Currency:    currency,
			Assets:      utils.FormatBigIntAmount(assetTotals[currency]),
			Liabilities: utils.FormatBigIntAmount(liabilityTotals[currency]),
			Net:         utils.FormatBigIntAmount(net),
		})
	}

	return newMCPJSONTextResult(response)
}

// mcpReportGetAccountFilter returns the set of account ids specified by the textual ids, a parent account id includes all its sub-accounts
func mcpReportGetAccountFilter(allAccounts []*models.Account, field string, accountIds []string) (map[int64]bool, error) {
	accountFilter := make(map[int64]bool)

	if len(accountIds) < 1 {
		return accountFilter, nil
	}

	accountMap := make(map[int64]*models.Account, len(allAccounts))

	for i := 0; i < len(allAccounts); i++ {
		accountMap[allAccounts[i].AccountId] = allAccounts[i]
	}

	for i := 0; i < len(accountIds); i++ {
		accountId, err := parseMCPRequiredId(field, accountIds[i])

		if err != nil {
			return nil, err
		}

		account, exists := accountMap[accountId]

		if !exists {
			return nil, NewMCPNotFoundError(field, "account", accountIds[i])
		}

		accountFilter[account.AccountId] = true

		if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			for j := 0; j < len(allAccounts); j++ {
				if allAccounts[j].ParentAccountId == account.AccountId {
					accountFilter[allAccounts[j].AccountId] = true
				}
			}
		}
	}

	return accountFilter, nil
}
