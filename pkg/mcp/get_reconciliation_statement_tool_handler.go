package mcp

import (
	"math/big"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpReconciliationStatementPageCount is the page count used to load transactions of the statement, same as the HTTP API
const mcpReconciliationStatementPageCount = int32(1000)

const mcpReconciliationStatementDefaultMaxItems = 200
const mcpReconciliationStatementMaxItems = 1000

// MCPGetReconciliationStatementRequest represents all parameters of the get account reconciliation statement request
type MCPGetReconciliationStatementRequest struct {
	AccountId   string `json:"account_id,omitempty" jsonschema_description:"Account ID (preferred, either account_id or account_name is required, must not be a parent account with sub-accounts)"`
	AccountName string `json:"account_name,omitempty" jsonschema_description:"Account name (used if account_id is not set)"`
	StartTime   string `json:"start_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Start time (inclusive) in RFC 3339 format (optional, default is from the earliest transaction)"`
	EndTime     string `json:"end_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"End time (inclusive) in RFC 3339 format (optional, default is up to now)"`
	MaxItems    int    `json:"max_items,omitempty" jsonschema_description:"Maximum count of statement lines to return (optional, default 200, maximum 1000). Opening / closing balances and totals always cover the whole range"`
}

// MCPReconciliationStatementAccount represents the account of the reconciliation statement
type MCPReconciliationStatementAccount struct {
	Id          string `json:"id" jsonschema_description:"Account ID"`
	Name        string `json:"name" jsonschema_description:"Account name"`
	Category    string `json:"category" jsonschema_description:"Account category"`
	Currency    string `json:"currency" jsonschema_description:"Currency code of the account"`
	IsLiability bool   `json:"is_liability" jsonschema_description:"Whether the account is a liability account (credit card, debt)"`
}

// MCPReconciliationStatementItem represents a line of the reconciliation statement
type MCPReconciliationStatementItem struct {
	Transaction            *MCPTransactionDetail `json:"transaction" jsonschema_description:"The transaction (for the incoming side of a transfer, account_id is this account and the transfer is identified by transfer_id)"`
	TransferId             string                `json:"transfer_id,omitempty" jsonschema_description:"ID of the transfer (outgoing side) to use with get_transaction / update_transaction, only for the incoming side of a transfer"`
	Direction              string                `json:"direction" jsonschema:"enum=inflow,enum=outflow" jsonschema_description:"Whether the transaction increases (inflow) or decreases (outflow) the signed balance of this account"`
	AccountAmount          string                `json:"account_amount" jsonschema_description:"Signed change of the signed balance of this account"`
	CounterpartAccountId   string                `json:"counterpart_account_id,omitempty" jsonschema_description:"The other account of a transfer"`
	CounterpartAccountName string                `json:"counterpart_account_name,omitempty" jsonschema_description:"Name of the other account of a transfer"`
	OpeningSignedBalance   string                `json:"opening_signed_balance" jsonschema_description:"Signed balance of this account before the transaction"`
	ClosingSignedBalance   string                `json:"closing_signed_balance" jsonschema_description:"Signed balance of this account after the transaction"`
}

// MCPGetReconciliationStatementResponse represents the response structure for get account reconciliation statement
type MCPGetReconciliationStatementResponse struct {
	Account              *MCPReconciliationStatementAccount `json:"account" jsonschema_description:"The account"`
	StartTime            string                             `json:"start_time,omitempty" jsonschema_description:"Start time of the range"`
	EndTime              string                             `json:"end_time,omitempty" jsonschema_description:"End time of the range"`
	OpeningBalance       string                             `json:"opening_balance" jsonschema_description:"Displayed balance before the range (outstanding balance for liability accounts)"`
	OpeningSignedBalance string                             `json:"opening_signed_balance" jsonschema_description:"Signed accounting balance before the range"`
	ClosingBalance       string                             `json:"closing_balance" jsonschema_description:"Displayed balance at the end of the range (outstanding balance for liability accounts)"`
	ClosingSignedBalance string                             `json:"closing_signed_balance" jsonschema_description:"Signed accounting balance at the end of the range"`
	TotalInflows         string                             `json:"total_inflows" jsonschema_description:"Total inflows within the range (same as the reconciliation statement in the web interface)"`
	TotalOutflows        string                             `json:"total_outflows" jsonschema_description:"Total outflows within the range (same as the reconciliation statement in the web interface)"`
	TransactionCount     int                                `json:"transaction_count" jsonschema_description:"Count of transactions within the range"`
	ReturnedCount        int                                `json:"returned_count" jsonschema_description:"Count of returned statement lines"`
	Truncated            bool                               `json:"truncated" jsonschema_description:"Whether only the earliest max_items lines are returned, narrow the time range to get the rest"`
	LastReconciledTime   string                             `json:"last_reconciled_time,omitempty" jsonschema_description:"Last reconciled time of the account in RFC 3339 format (UTC), if set"`
	Transactions         []*MCPReconciliationStatementItem  `json:"transactions" jsonschema_description:"Statement lines ordered by transaction time ascending"`
}

type mcpGetReconciliationStatementToolHandler struct{}

var MCPGetReconciliationStatementToolHandler = &mcpGetReconciliationStatementToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetReconciliationStatementToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_RECONCILIATION_STATEMENT
}

// Name returns the name of the MCP tool
func (h *mcpGetReconciliationStatementToolHandler) Name() string {
	return "get_account_reconciliation_statement"
}

// Description returns the description of the MCP tool
func (h *mcpGetReconciliationStatementToolHandler) Description() string {
	return "Get the reconciliation statement of an account over a date range (the same data as the reconciliation statement in the web interface): opening and closing balances, total inflows and outflows, and every transaction with its ID, attachment references and the running balance before and after it."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetReconciliationStatementToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetReconciliationStatementToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetReconciliationStatementToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetReconciliationStatementToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetReconciliationStatementRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetReconciliationStatementToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetReconciliationStatementResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetReconciliationStatementToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPGetReconciliationStatementRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	maxItems := request.MaxItems

	if maxItems <= 0 {
		maxItems = mcpReconciliationStatementDefaultMaxItems
	} else if maxItems > mcpReconciliationStatementMaxItems {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeOutOfRange, "max_items", "\"max_items\" must not be greater than 1000")
	}

	startUnixTime, endUnixTime, timezone, err := mcpReportParseTimeRange(request.StartTime, request.EndTime)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[get_reconciliation_statement_tool_handler.Handle] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	account, err := mcpTxnResolveAccount(allAccounts, "account_id", request.AccountId, "account_name", request.AccountName)

	if err != nil {
		return nil, nil, err
	}

	maxTransactionTime := int64(0)

	if endUnixTime > 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(endUnixTime)
	}

	minTransactionTime := int64(0)

	if startUnixTime > 0 {
		minTransactionTime = utils.GetMinTransactionTimeFromUnixTime(startUnixTime)
	}

	transactionsWithAccountBalance, totalInflows, totalOutflows, openingBalance, closingBalance, err := services.GetTransactionService().GetAllTransactionsInOneAccountWithAccountBalanceByMaxTime(c, uid, mcpReconciliationStatementPageCount, maxTransactionTime, minTransactionTime, account.AccountId, account.Category)

	if err != nil {
		log.Errorf(c, "[get_reconciliation_statement_tool_handler.Handle] failed to get statement of account \"id:%d\" for user \"uid:%d\", because %s", account.AccountId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	response := &MCPGetReconciliationStatementResponse{
		Account: &MCPReconciliationStatementAccount{
			Id:          utils.Int64ToString(account.AccountId),
			Name:        account.Name,
			Category:    getMCPAccountCategoryName(account.Category),
			Currency:    account.Currency,
			IsLiability: account.Category.IsLiability(),
		},
		TotalInflows:       utils.FormatBigIntAmount(totalInflows),
		TotalOutflows:      utils.FormatBigIntAmount(totalOutflows),
		TransactionCount:   len(transactionsWithAccountBalance),
		LastReconciledTime: mcpAccountGetLastReconciledTime(account),
		Transactions:       make([]*MCPReconciliationStatementItem, 0),
	}

	response.OpeningBalance, response.OpeningSignedBalance, _ = mcpReportFormatBalances(account.Category, openingBalance)
	response.ClosingBalance, response.ClosingSignedBalance, _ = mcpReportFormatBalances(account.Category, closingBalance)

	if startUnixTime > 0 {
		response.StartTime = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(startUnixTime, timezone)
	}

	if endUnixTime > 0 {
		response.EndTime = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(endUnixTime, timezone)
	}

	if len(transactionsWithAccountBalance) > maxItems {
		transactionsWithAccountBalance = transactionsWithAccountBalance[:maxItems]
		response.Truncated = true
	}

	response.ReturnedCount = len(transactionsWithAccountBalance)

	if len(transactionsWithAccountBalance) < 1 {
		return newMCPJSONTextResult(response)
	}

	transactions := make([]*models.Transaction, 0, len(transactionsWithAccountBalance))
	transferIds := make([]int64, 0)

	for i := 0; i < len(transactionsWithAccountBalance); i++ {
		transaction := transactionsWithAccountBalance[i].Transaction
		transactions = append(transactions, transaction)

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			transferIds = append(transferIds, transaction.RelatedId)
		}
	}

	detailData, err := mcpTxnLoadDetailData(c, uid, transactions, currentConfig, services)

	if err != nil {
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	// attachments of a transfer are saved with the outgoing side, so load them for the incoming side lines of this statement
	if len(transferIds) > 0 && currentConfig != nil && currentConfig.EnableTransactionPictures {
		transferAttachments, err := services.GetTransactionPictureService().GetPictureInfosByTransactionIds(c, uid, transferIds)

		if err != nil {
			log.Errorf(c, "[get_reconciliation_statement_tool_handler.Handle] failed to get attachments of transfers for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		for _, transaction := range transactions {
			if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
				detailData.attachments[transaction.TransactionId] = transferAttachments[transaction.RelatedId]
			}
		}
	}

	for i := 0; i < len(transactionsWithAccountBalance); i++ {
		transactionWithBalance := transactionsWithAccountBalance[i]
		transaction := transactionWithBalance.Transaction
		accountAmount := mcpReconciliationGetAccountAmount(transaction)

		item := &MCPReconciliationStatementItem{
			Transaction:          mcpTxnBuildDetail(transaction, detailData, user, true),
			Direction:            "inflow",
			AccountAmount:        utils.FormatBigIntAmount(accountAmount),
			OpeningSignedBalance: utils.FormatBigIntAmount(mcpReconciliationBigIntOrZero(transactionWithBalance.AccountOpeningBalance)),
			ClosingSignedBalance: utils.FormatBigIntAmount(mcpReconciliationBigIntOrZero(transactionWithBalance.AccountClosingBalance)),
		}

		if accountAmount.Sign() < 0 {
			item.Direction = "outflow"
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			item.CounterpartAccountId = utils.Int64ToString(transaction.RelatedAccountId)

			if counterpartAccount, exists := detailData.accounts[transaction.RelatedAccountId]; exists && counterpartAccount != nil {
				item.CounterpartAccountName = counterpartAccount.Name
			}
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			item.TransferId = utils.Int64ToString(transaction.RelatedId)
		}

		response.Transactions = append(response.Transactions, item)
	}

	return newMCPJSONTextResult(response)
}

// mcpReconciliationGetAccountAmount returns the signed change of the account balance caused by the transaction (same calculation as the statement service)
func mcpReconciliationGetAccountAmount(transaction *models.Transaction) *big.Int {
	switch transaction.Type {
	case models.TRANSACTION_DB_TYPE_MODIFY_BALANCE:
		return big.NewInt(transaction.RelatedAccountAmount)
	case models.TRANSACTION_DB_TYPE_INCOME, models.TRANSACTION_DB_TYPE_TRANSFER_IN:
		return big.NewInt(transaction.Amount)
	case models.TRANSACTION_DB_TYPE_EXPENSE, models.TRANSACTION_DB_TYPE_TRANSFER_OUT:
		return big.NewInt(-transaction.Amount)
	default:
		return big.NewInt(0)
	}
}

// mcpReconciliationBigIntOrZero returns the value or zero if nil
func mcpReconciliationBigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}

	return value
}
