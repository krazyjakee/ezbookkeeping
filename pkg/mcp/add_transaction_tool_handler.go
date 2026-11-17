package mcp

import (
	"math/big"
	"reflect"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const transactionTypeIncome = "income"
const transactionTypeExpense = "expense"
const transactionTypeTransfer = "transfer"
const transactionTypeModifyBalance = "balance_modification"

// MCPAddTransactionRequest represents all parameters of the add transaction request
type MCPAddTransactionRequest struct {
	Type                   string                           `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction type (income, expense, transfer)"`
	Time                   string                           `json:"time" jsonschema:"format=date-time" jsonschema_description:"Transaction time in RFC 3339 format (e.g. 2023-01-01T12:00:00Z), the UTC offset of the time is saved as the transaction timezone"`
	SecondaryCategoryName  string                           `json:"category_name,omitempty" jsonschema_description:"Secondary category name for the transaction (required if category_id is not set, must match exactly one visible secondary category of the transaction type)"`
	CategoryId             string                           `json:"category_id,omitempty" jsonschema_description:"Secondary category ID for the transaction (optional, takes precedence over category_name)"`
	AccountName            string                           `json:"account_name,omitempty" jsonschema_description:"Account name for the transaction (required if account_id is not set, must match exactly one visible account)"`
	AccountId              string                           `json:"account_id,omitempty" jsonschema_description:"Account ID for the transaction (optional, takes precedence over account_name)"`
	Amount                 string                           `json:"amount" jsonschema_description:"Transaction amount (e.g. for an expense transaction, 12.34 represents an expense of 12.34)"`
	DestinationAccountName string                           `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name for transfer transactions (required for transfer if destination_account_id is not set)"`
	DestinationAccountId   string                           `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID for transfer transactions (optional, takes precedence over destination_account_name)"`
	DestinationAmount      string                           `json:"destination_amount,omitempty" jsonschema_description:"Destination amount for transfer transactions (required for transfer, must equal amount when both accounts use the same currency)"`
	Tags                   []string                         `json:"tags,omitempty" jsonschema_description:"List of tag names associated with the transaction (optional, maximum 10 tags allowed, unknown or ambiguous tag names are rejected)"`
	TagIds                 []string                         `json:"tag_ids,omitempty" jsonschema_description:"List of tag IDs associated with the transaction (optional, combined with tags, maximum 10 tags allowed)"`
	Comment                string                           `json:"comment,omitempty" jsonschema_description:"Transaction description (maximum length is reported by get_server_capabilities)"`
	GeoLocation            *MCPTransactionDetailGeoLocation `json:"geo_location,omitempty" jsonschema_description:"Geographic location of the transaction (optional)"`
	DryRun                 bool                             `json:"dry_run,omitempty" jsonschema_description:"If true, the transaction will not be saved, only validated with the same rules as a real write (optional)"`
}

// MCPAddTransactionResponse represents the response structure for add transaction
type MCPAddTransactionResponse struct {
	Success                         bool                  `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun                          bool                  `json:"dry_run,omitzero" jsonschema_description:"Indicates whether this operation is a dry run (transaction not saved actually)"`
	Committed                       bool                  `json:"committed" jsonschema_description:"Indicates whether the transaction has been saved to the database"`
	TransactionId                   string                `json:"transaction_id,omitempty" jsonschema_description:"ID of the saved transaction (only when committed)"`
	AccountBalance                  string                `json:"account_balance,omitempty" jsonschema_description:"Account balance (or outstanding balance for debt accounts) after the transaction"`
	AccountSignedBalance            string                `json:"account_signed_balance,omitempty" jsonschema_description:"Signed accounting balance of the account after the transaction (negative means money owed for liability accounts)"`
	DestinationAccountBalance       string                `json:"destination_account_balance,omitempty" jsonschema_description:"Destination account balance (or outstanding balance for debt accounts) after the transaction (only for transfer transactions)"`
	DestinationAccountSignedBalance string                `json:"destination_account_signed_balance,omitempty" jsonschema_description:"Signed accounting balance of the destination account after the transaction (only for transfer transactions)"`
	Transaction                     *MCPTransactionDetail `json:"transaction,omitempty" jsonschema_description:"The persisted transaction (when committed) or the validated transaction which would be saved (when dry run, without ID)"`
}

type mcpAddTransactionToolHandler struct{}

var MCPAddTransactionToolHandler = &mcpAddTransactionToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpAddTransactionToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_ADD_TRANSACTION
}

// Name returns the name of the MCP tool
func (h *mcpAddTransactionToolHandler) Name() string {
	return "add_transaction"
}

// Description returns the description of the MCP tool
func (h *mcpAddTransactionToolHandler) Description() string {
	return "Add a new transaction in ezBookkeeping. Accounts, category and tags can be specified by ID (preferred) or by name; ambiguous or unknown names are rejected with candidates. Use dry_run to validate without saving. Returns the saved transaction ID and the resulting account balances."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpAddTransactionToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpAddTransactionToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpAddTransactionToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpAddTransactionToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPAddTransactionRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpAddTransactionToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPAddTransactionResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpAddTransactionToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	addTransactionRequest, err := parseMCPToolArguments[MCPAddTransactionRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := h.validateRequest(addTransactionRequest); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.Handle] get account error, because %s", err.Error())
		return nil, nil, err
	}

	sourceAccount, err := mcpTxnResolveAccount(allAccounts, "account_id", addTransactionRequest.AccountId, "account_name", addTransactionRequest.AccountName)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.Handle] cannot resolve source account for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	var destinationAccount *models.Account
	destinationAccountId := int64(0)

	if addTransactionRequest.Type == transactionTypeTransfer {
		destinationAccount, err = mcpTxnResolveAccount(allAccounts, "destination_account_id", addTransactionRequest.DestinationAccountId, "destination_account_name", addTransactionRequest.DestinationAccountName)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.Handle] cannot resolve destination account for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		destinationAccountId = destinationAccount.AccountId
	}

	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.Handle] get transaction category error, because %s", err.Error())
		return nil, nil, err
	}

	transactionCategory, err := mcpTxnResolveCategory(allCategories, getMCPCategoryTypeByTransactionType(addTransactionRequest.Type), "category_id", addTransactionRequest.CategoryId, "category_name", addTransactionRequest.SecondaryCategoryName)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.Handle] cannot resolve category for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	var allTags []*models.TransactionTag
	tagIds := make([]int64, 0)

	if len(addTransactionRequest.Tags) > 0 || len(addTransactionRequest.TagIds) > 0 {
		allTags, err = services.GetTransactionTagService().GetAllTagsByUid(c, uid)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.Handle] get transaction tag ids error, because %s", err.Error())
			return nil, nil, err
		}

		tagIds, err = mcpTxnResolveTagIds(allTags, "tag_ids", addTransactionRequest.TagIds, "tags", addTransactionRequest.Tags)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.Handle] cannot resolve tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}
	}

	transaction, err := h.createNewTransactionModel(c, uid, addTransactionRequest, transactionCategory.CategoryId, sourceAccount.AccountId, destinationAccountId, c.ClientIP())

	if err != nil {
		return nil, nil, err
	}

	if err := mcpTxnValidateAccounts(transaction, sourceAccount, destinationAccount); err != nil {
		return nil, nil, err
	}

	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60), sourceAccount, destinationAccount)

	if !transactionEditable {
		return nil, nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
	}

	if !addTransactionRequest.DryRun {
		err = services.GetTransactionService().CreateTransaction(c, transaction, tagIds, nil)

		if err != nil {
			log.Errorf(c, "[add_transaction_tool_handler.Handle] failed to create transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, nil, err
		}

		log.Infof(c, "[add_transaction_tool_handler.Handle] user \"uid:%d\" has created a new transaction \"id:%d\" successfully", uid, transaction.TransactionId)

		accountIds := []int64{sourceAccount.AccountId}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			accountIds = append(accountIds, destinationAccountId)
		}

		newAccounts, err := services.GetAccountService().GetAccountsByAccountIds(c, uid, accountIds)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.Handle] failed to get latest accounts info after transaction created, because %s", err.Error())
		}

		var detail *MCPTransactionDetail
		savedTransaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, transaction.TransactionId)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.Handle] failed to get transaction \"id:%d\" after created, because %s", transaction.TransactionId, err.Error())
		} else {
			detailData, err := mcpTxnLoadDetailData(c, uid, []*models.Transaction{savedTransaction}, currentConfig, services)

			if err != nil {
				log.Warnf(c, "[add_transaction_tool_handler.Handle] failed to get related data of transaction \"id:%d\" after created, because %s", transaction.TransactionId, err.Error())
			} else {
				detail = mcpTxnBuildDetail(savedTransaction, detailData, user, true)
			}
		}

		return h.createNewMCPAddTransactionResponse(transaction, newAccounts, detail, false)
	} else {
		newAccounts := make(map[int64]*models.Account)
		sourceAccountCopy := *sourceAccount
		newAccounts[sourceAccount.AccountId] = &sourceAccountCopy

		if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			newBalance, err := h.getAccountBalanceAfterUpdate(sourceAccountCopy.Balance, -transaction.Amount)

			if err != nil {
				return nil, nil, err
			}

			sourceAccountCopy.Balance = newBalance
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			newBalance, err := h.getAccountBalanceAfterUpdate(sourceAccountCopy.Balance, transaction.Amount)

			if err != nil {
				return nil, nil, err
			}

			sourceAccountCopy.Balance = newBalance
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && destinationAccount != nil {
			destinationAccountCopy := *destinationAccount
			newAccounts[destinationAccount.AccountId] = &destinationAccountCopy
			newBalance, err := h.getAccountBalanceAfterUpdate(destinationAccountCopy.Balance, transaction.RelatedAccountAmount)

			if err != nil {
				return nil, nil, err
			}

			destinationAccountCopy.Balance = newBalance
		}

		detailData := &mcpTxnDetailData{
			accounts:   services.GetAccountService().GetAccountMapByList(allAccounts),
			categories: services.GetTransactionCategoryService().GetCategoryMapByList(allCategories),
			tags:       services.GetTransactionTagService().GetTagMapByList(allTags),
			tagIds: map[int64][]int64{
				0: tagIds,
			},
		}

		detail := mcpTxnBuildDetail(transaction, detailData, nil, false)
		detail.Id = ""

		return h.createNewMCPAddTransactionResponse(transaction, newAccounts, detail, true)
	}
}

// validateRequest validates all fields which do not depend on stored data, so that dry run and real write use the same rules
func (h *mcpAddTransactionToolHandler) validateRequest(addTransactionRequest *MCPAddTransactionRequest) error {
	if addTransactionRequest.Type != transactionTypeIncome && addTransactionRequest.Type != transactionTypeExpense && addTransactionRequest.Type != transactionTypeTransfer {
		return NewMCPInvalidFieldError("type", addTransactionRequest.Type, "\"type\" must be one of income, expense, transfer")
	}

	if addTransactionRequest.Type == transactionTypeTransfer {
		if addTransactionRequest.DestinationAccountName == "" && addTransactionRequest.DestinationAccountId == "" {
			return NewMCPRequiredFieldError("destination_account_id or destination_account_name")
		}

		if addTransactionRequest.DestinationAmount == "" {
			return NewMCPRequiredFieldError("destination_amount")
		}
	} else {
		if addTransactionRequest.DestinationAccountName != "" || addTransactionRequest.DestinationAccountId != "" {
			return NewMCPInvalidFieldError("destination_account_id", addTransactionRequest.DestinationAccountId+addTransactionRequest.DestinationAccountName, "destination account can only be set for transfer transactions")
		}

		if addTransactionRequest.DestinationAmount != "" {
			return NewMCPInvalidFieldError("destination_amount", addTransactionRequest.DestinationAmount, "\"destination_amount\" can only be set for transfer transactions")
		}
	}

	if count := len(addTransactionRequest.Tags) + len(addTransactionRequest.TagIds); count > models.MaximumTagsCountOfTransaction {
		return NewMCPTooManyItemsError("tags", models.MaximumTagsCountOfTransaction, count)
	}

	if err := checkMCPStringLength("comment", addTransactionRequest.Comment, false, mcpTransactionCommentMaxLength); err != nil {
		return err
	}

	if _, _, err := mcpTxnParseGeoLocation("geo_location", addTransactionRequest.GeoLocation); err != nil {
		return err
	}

	if _, err := parseMCPTime("time", addTransactionRequest.Time, true); err != nil {
		return err
	}

	if _, err := parseMCPAmount("amount", addTransactionRequest.Amount, true); err != nil {
		return err
	}

	if addTransactionRequest.Type == transactionTypeTransfer {
		if _, err := parseMCPAmount("destination_amount", addTransactionRequest.DestinationAmount, true); err != nil {
			return err
		}
	}

	return nil
}

func (h *mcpAddTransactionToolHandler) getAccountBalanceAfterUpdate(balance int64, delta int64) (int64, error) {
	newBalance, ok := utils.AddInt64(balance, delta)

	if !ok {
		return 0, errs.ErrAccountBalanceOverflow
	}

	return newBalance, nil
}

func (h *mcpAddTransactionToolHandler) createNewTransactionModel(c *core.WebContext, uid int64, addTransactionRequest *MCPAddTransactionRequest, categoryId int64, sourceAccountId int64, destinationAccountId int64, clientIp string) (*models.Transaction, error) {
	var transactionDbType models.TransactionDbType

	if addTransactionRequest.Type == transactionTypeExpense {
		transactionDbType = models.TRANSACTION_DB_TYPE_EXPENSE
	} else if addTransactionRequest.Type == transactionTypeIncome {
		transactionDbType = models.TRANSACTION_DB_TYPE_INCOME
	} else if addTransactionRequest.Type == transactionTypeTransfer {
		transactionDbType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	} else {
		return nil, errs.ErrTransactionTypeInvalid
	}

	transactionTime, err := parseMCPTime("time", addTransactionRequest.Time, true)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.createNewTransactionModel] parse transaction time \"%s\" error, because %s", addTransactionRequest.Time, err.Error())
		return nil, err
	}

	amount, err := parseMCPAmount("amount", addTransactionRequest.Amount, true)

	if err != nil {
		log.Warnf(c, "[add_transaction_tool_handler.createNewTransactionModel] parse transaction amount \"%s\" error, because %s", addTransactionRequest.Amount, err.Error())
		return nil, err
	}

	latitude, longitude, err := mcpTxnParseGeoLocation("geo_location", addTransactionRequest.GeoLocation)

	if err != nil {
		return nil, err
	}

	transaction := &models.Transaction{
		Uid:               uid,
		Type:              transactionDbType,
		CategoryId:        categoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(transactionTime.Unix()),
		TimezoneUtcOffset: utils.GetTimezoneOffsetMinutes(transactionTime.Unix(), transactionTime.Location()),
		AccountId:         sourceAccountId,
		Amount:            amount,
		HideAmount:        false,
		Comment:           addTransactionRequest.Comment,
		GeoLatitude:       latitude,
		GeoLongitude:      longitude,
		CreatedIp:         clientIp,
	}

	if addTransactionRequest.Type == transactionTypeTransfer {
		transaction.RelatedAccountId = destinationAccountId

		destinationAmount, err := parseMCPAmount("destination_amount", addTransactionRequest.DestinationAmount, true)

		if err != nil {
			log.Warnf(c, "[add_transaction_tool_handler.createNewTransactionModel] parse transaction destination amount \"%s\" error, because %s", addTransactionRequest.DestinationAmount, err.Error())
			return nil, err
		}

		transaction.RelatedAccountAmount = destinationAmount
	}

	return transaction, nil
}

func (h *mcpAddTransactionToolHandler) createNewMCPAddTransactionResponse(transaction *models.Transaction, accountsMap map[int64]*models.Account, detail *MCPTransactionDetail, dryRun bool) (any, []*MCPTextContent, error) {
	var sourceAccount *models.Account
	var destinationAccount *models.Account

	if account, exists := accountsMap[transaction.AccountId]; exists {
		sourceAccount = account
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if account, exists := accountsMap[transaction.RelatedAccountId]; exists {
			destinationAccount = account
		}
	}

	response := MCPAddTransactionResponse{
		Success:     true,
		DryRun:      dryRun,
		Committed:   !dryRun,
		Transaction: detail,
	}

	if !dryRun {
		response.TransactionId = utils.Int64ToString(transaction.TransactionId)
	}

	if sourceAccount != nil {
		response.AccountBalance, response.AccountSignedBalance = mcpTxnGetAccountBalances(sourceAccount)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && destinationAccount != nil {
		response.DestinationAccountBalance, response.DestinationAccountSignedBalance = mcpTxnGetAccountBalances(destinationAccount)
	}

	return newMCPJSONTextResult(response)
}

// mcpTxnGetAccountBalances returns the displayed balance (outstanding balance for liability accounts, same as before) and the signed accounting balance of the account
func mcpTxnGetAccountBalances(account *models.Account) (string, string) {
	signedBalance := utils.FormatAmount(account.Balance)

	if account.Category.IsAsset() {
		return utils.FormatAmount(account.Balance), signedBalance
	} else if account.Category.IsLiability() {
		return utils.FormatBigIntAmount(new(big.Int).Neg(big.NewInt(account.Balance))), signedBalance
	}

	return "", signedBalance
}

// mcpTxnValidateAccounts validates the accounts of the transaction with the same rules as the transaction service, so that dry run reports the same errors as a real write
func mcpTxnValidateAccounts(transaction *models.Transaction, sourceAccount *models.Account, destinationAccount *models.Account) error {
	if sourceAccount == nil {
		return NewMCPRequiredFieldError("account_id or account_name")
	}

	if sourceAccount.Hidden {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "account_id", "account \""+sourceAccount.Name+"\" is hidden, transactions cannot be added to or modified in hidden accounts")
	}

	if sourceAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "account_id", "account \""+sourceAccount.Name+"\" has sub-accounts, please specify one of its sub-accounts instead")
	}

	if transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		return nil
	}

	if destinationAccount == nil {
		return NewMCPRequiredFieldError("destination_account_id or destination_account_name")
	}

	if destinationAccount.Hidden {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "destination_account_id", "account \""+destinationAccount.Name+"\" is hidden, transactions cannot be added to or modified in hidden accounts")
	}

	if destinationAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "destination_account_id", "account \""+destinationAccount.Name+"\" has sub-accounts, please specify one of its sub-accounts instead")
	}

	if sourceAccount.AccountId == destinationAccount.AccountId {
		return NewMCPInvalidFieldError("destination_account_id", utils.Int64ToString(destinationAccount.AccountId), "source account and destination account of a transfer cannot be the same")
	}

	if sourceAccount.Currency == destinationAccount.Currency && transaction.Amount != transaction.RelatedAccountAmount {
		return NewMCPInvalidFieldError("destination_amount", utils.FormatAmount(transaction.RelatedAccountAmount), "\"destination_amount\" must equal \"amount\" when source and destination accounts use the same currency ("+sourceAccount.Currency+")")
	}

	if transaction.Amount < 0 || transaction.RelatedAccountAmount < 0 {
		return NewMCPToolError(MCPToolErrorCodeOutOfRange, "amount", "transfer amounts cannot be less than zero")
	}

	return nil
}
