package mcp

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPCreateAccountRequest represents all parameters of the create account request
type MCPCreateAccountRequest struct {
	Name                    string                               `json:"name" jsonschema_description:"Account name"`
	Category                string                               `json:"category" jsonschema:"enum=cash,enum=checking_account,enum=savings_account,enum=credit_card,enum=virtual,enum=debt,enum=receivables,enum=certificate_of_deposit,enum=investment" jsonschema_description:"Account category (credit_card and debt are liability accounts, e.g. use debt for a director's loan account owed to a director)"`
	Currency                string                               `json:"currency,omitempty" jsonschema_description:"Currency code in ISO 4217 format (e.g. USD, GBP). Required for accounts without sub-accounts; must be empty for parent accounts with sub-accounts except credit_card parents (optional there)"`
	Comment                 string                               `json:"comment,omitempty" jsonschema_description:"Account description (optional)"`
	Icon                    string                               `json:"icon,omitempty" jsonschema_description:"Built-in icon ID (optional, default: 1)"`
	Color                   string                               `json:"color,omitempty" jsonschema_description:"Icon color in 6-digit hex RGB format (optional, default: 000000)"`
	OpeningBalance          string                               `json:"opening_balance,omitempty" jsonschema_description:"Signed opening balance (optional, e.g. 100.00; for liability accounts a negative value means money owed). Not allowed for parent accounts with sub-accounts. Do not set together with outstanding_balance"`
	OutstandingBalance      string                               `json:"outstanding_balance,omitempty" jsonschema_description:"Opening outstanding balance for liability accounts only (optional, positive means money owed, e.g. 120.00 is stored as signed balance -120.00). Do not set together with opening_balance"`
	OpeningBalanceTime      string                               `json:"opening_balance_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Effective time of the opening balance in RFC 3339 format (required if the opening balance is not zero), the opening balance is recorded as a balance modification transaction at this time and in the timezone of this value"`
	CreditCardStatementDate int                                  `json:"credit_card_statement_date,omitempty" jsonschema:"minimum=0,maximum=28" jsonschema_description:"Statement date of the credit card (optional, 1-28, only for credit_card accounts)"`
	CreditCardLimit         string                               `json:"credit_card_limit,omitempty" jsonschema_description:"Credit limit of the credit card (optional, positive amount, only for credit_card accounts)"`
	SubAccounts             []*MCPCreateAccountSubAccountRequest `json:"sub_accounts,omitempty" jsonschema_description:"Sub-accounts to create (optional). If set, the account is created as a parent account (multi_sub_accounts type) whose balances are held by the sub-accounts, all sub-accounts have the same category as the parent"`
	SkipExisting            bool                                 `json:"skip_existing,omitempty" jsonschema_description:"If true and a top-level account with the same name and category already exists, the existing account is returned with status existing and nothing is created (optional, recommended for setup scripts which may be rerun)"`
	DryRun                  bool                                 `json:"dry_run,omitempty" jsonschema_description:"If true, the account is only validated with the same rules as the real creation and not saved (optional)"`
}

// MCPCreateAccountResponse represents the response structure for create account
type MCPCreateAccountResponse struct {
	Success bool            `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun  bool            `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (if true, nothing is saved)"`
	Status  string          `json:"status" jsonschema:"enum=created,enum=existing,enum=would_be_created" jsonschema_description:"Result status: created (committed), existing (an existing account is reused because skip_existing is true, nothing is saved), would_be_created (dry run, nothing is saved)"`
	Account *MCPAccountInfo `json:"account" jsonschema_description:"The persisted account (re-read after creation), the existing account, or the would-be account for dry run"`
}

type mcpCreateAccountToolHandler struct{}

var MCPCreateAccountToolHandler = &mcpCreateAccountToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpCreateAccountToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_CREATE_ACCOUNT
}

// Name returns the name of the MCP tool
func (h *mcpCreateAccountToolHandler) Name() string {
	return "create_account"
}

// Description returns the description of the MCP tool
func (h *mcpCreateAccountToolHandler) Description() string {
	return "Create a new account (optionally with sub-accounts) in ezBookkeeping, including liability accounts (credit_card, debt) and an optional opening balance with its effective time. Supports dry_run and skip_existing to make setup reruns safe."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpCreateAccountToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpCreateAccountToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpCreateAccountToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpCreateAccountToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateAccountRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpCreateAccountToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateAccountResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpCreateAccountToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	createAccountRequest, err := parseMCPToolArguments[MCPCreateAccountRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	plan, err := h.buildCreatePlan(uid, createAccountRequest)

	if err != nil {
		return nil, nil, err
	}

	if createAccountRequest.SkipExisting {
		allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

		if err != nil {
			log.Errorf(c, "[create_account_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		existingAccount, err := mcpAccountFindExistingForSkip(allAccounts, createAccountRequest.Name, plan.mainAccount.Category)

		if err != nil {
			return nil, nil, err
		}

		if existingAccount != nil {
			log.Infof(c, "[create_account_tool_handler.Handle] account \"id:%d\" with the same name already exists for user \"uid:%d\", skip creating", existingAccount.AccountId, uid)

			return newMCPJSONTextResult(&MCPCreateAccountResponse{
				Success: true,
				DryRun:  createAccountRequest.DryRun,
				Status:  MCPWriteStatusExisting,
				Account: mcpAccountBuildInfo(existingAccount, allAccounts, true),
			})
		}
	}

	if createAccountRequest.DryRun {
		return newMCPJSONTextResult(&MCPCreateAccountResponse{
			Success: true,
			DryRun:  true,
			Status:  MCPWriteStatusWouldBeCreated,
			Account: mcpAccountBuildInfo(plan.mainAccount, plan.childrenAccounts, true),
		})
	}

	maxOrderId, err := services.GetAccountService().GetMaxDisplayOrder(c, uid, plan.mainAccount.Category)

	if err != nil {
		log.Errorf(c, "[create_account_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	plan.mainAccount.DisplayOrder = maxOrderId + 1

	err = services.GetAccountService().CreateAccounts(c, plan.mainAccount, plan.mainAccountBalanceTime, plan.childrenAccounts, plan.childrenAccountBalanceTimes, plan.clientTimezone)

	if err != nil {
		log.Errorf(c, "[create_account_tool_handler.Handle] failed to create account \"id:%d\" for user \"uid:%d\", because %s", plan.mainAccount.AccountId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[create_account_tool_handler.Handle] user \"uid:%d\" has created a new account \"id:%d\" successfully", uid, plan.mainAccount.AccountId)

	accountInfo := mcpAccountBuildInfo(plan.mainAccount, plan.childrenAccounts, true)
	accountAndSubAccounts, err := services.GetAccountService().GetAccountAndSubAccountsByAccountId(c, uid, plan.mainAccount.AccountId)

	if err != nil {
		log.Warnf(c, "[create_account_tool_handler.Handle] failed to get latest account \"id:%d\" info after account created, because %s", plan.mainAccount.AccountId, err.Error())
	} else if latestAccountInfo, err := mcpAccountBuildInfoFromList(accountAndSubAccounts, plan.mainAccount.AccountId); err == nil {
		accountInfo = latestAccountInfo
	}

	return newMCPJSONTextResult(&MCPCreateAccountResponse{
		Success: true,
		DryRun:  false,
		Status:  MCPWriteStatusCreated,
		Account: accountInfo,
	})
}

// buildCreatePlan validates the request with the same rules as the account creation api and returns the account models to be created
func (h *mcpCreateAccountToolHandler) buildCreatePlan(uid int64, req *MCPCreateAccountRequest) (*mcpAccountCreatePlan, error) {
	if err := checkMCPStringLength("name", req.Name, true, mcpAccountNameMaxLength); err != nil {
		return nil, err
	}

	if err := checkMCPStringLength("comment", req.Comment, false, mcpAccountCommentMaxLength); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Category) == "" {
		return nil, NewMCPRequiredFieldError("category")
	}

	category, err := parseMCPAccountCategory("category", req.Category)

	if err != nil {
		return nil, err
	}

	icon, err := mcpAccountParseIcon("icon", req.Icon)

	if err != nil {
		return nil, err
	}

	color, err := mcpAccountParseColor("color", req.Color)

	if err != nil {
		return nil, err
	}

	if category != models.ACCOUNT_CATEGORY_CREDIT_CARD && req.CreditCardStatementDate != 0 {
		return nil, NewMCPNotSupportedError("credit_card_statement_date", "\"credit_card_statement_date\" can only be set for credit_card accounts")
	}

	if req.CreditCardStatementDate < 0 || req.CreditCardStatementDate > mcpAccountMaxCreditCardStatementDate {
		toolErr := NewMCPToolError(MCPToolErrorCodeOutOfRange, "credit_card_statement_date", "\"credit_card_statement_date\" must be between 0 and "+strconv.Itoa(mcpAccountMaxCreditCardStatementDate))
		limit := int64(mcpAccountMaxCreditCardStatementDate)
		toolErr.Limit = &limit
		return nil, toolErr
	}

	if category != models.ACCOUNT_CATEGORY_CREDIT_CARD && strings.TrimSpace(req.CreditCardLimit) != "" {
		return nil, NewMCPNotSupportedError("credit_card_limit", "\"credit_card_limit\" can only be set for credit_card accounts")
	}

	creditLimitForCreditCard, err := parseMCPAmount("credit_card_limit", req.CreditCardLimit, false)

	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.CreditCardLimit) != "" && creditLimitForCreditCard <= 0 {
		return nil, NewMCPInvalidFieldError("credit_card_limit", req.CreditCardLimit, "\"credit_card_limit\" must be a positive amount")
	}

	mainAccountBalance, err := mcpAccountParseOpeningBalance("", req.OpeningBalance, req.OutstandingBalance, category)

	if err != nil {
		return nil, err
	}

	mainAccountBalanceTime, err := mcpAccountParseBalanceTime("opening_balance_time", req.OpeningBalanceTime, mainAccountBalance)

	if err != nil {
		return nil, err
	}

	var clientTimezone *time.Location

	if mainAccountBalanceTime != nil {
		clientTimezone = mainAccountBalanceTime.Location()
	}

	accountType := models.ACCOUNT_TYPE_SINGLE_ACCOUNT
	currency := ""

	if len(req.SubAccounts) < 1 {
		currency, err = mcpAccountParseCurrency("currency", req.Currency, false)

		if err != nil {
			return nil, err
		}
	} else {
		accountType = models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS

		if len(req.SubAccounts) > mcpMaximumBatchItemsCount {
			return nil, NewMCPTooManyItemsError("sub_accounts", mcpMaximumBatchItemsCount, len(req.SubAccounts))
		}

		if category == models.ACCOUNT_CATEGORY_CREDIT_CARD {
			currency, err = mcpAccountParseCurrency("currency", req.Currency, true)

			if err != nil {
				return nil, err
			}

			if currency == core.AccountCurrencyNotSetValue && creditLimitForCreditCard > 0 {
				return nil, NewMCPToolError(MCPToolErrorCodeRequired, "currency", "\"currency\" of the parent account must be set when \"credit_card_limit\" is set")
			}
		} else {
			if strings.TrimSpace(req.Currency) != "" && strings.TrimSpace(req.Currency) != core.AccountCurrencyNotSetValue {
				return nil, NewMCPNotSupportedError("currency", "\"currency\" cannot be set for a parent account with sub-accounts, set the currency of each sub-account instead")
			}

			currency = core.AccountCurrencyNotSetValue
		}

		if mainAccountBalance != 0 {
			return nil, NewMCPNotSupportedError("opening_balance", "the opening balance cannot be set for a parent account with sub-accounts, set the opening balance of each sub-account instead")
		}
	}

	accountExtend := &models.AccountExtend{}

	if category == models.ACCOUNT_CATEGORY_CREDIT_CARD {
		statementDate := req.CreditCardStatementDate
		accountExtend.CreditCardStatementDate = &statementDate
		accountExtend.CreditCardLimit = &creditLimitForCreditCard
	}

	plan := &mcpAccountCreatePlan{
		mainAccount: &models.Account{
			Uid:      uid,
			Name:     req.Name,
			Category: category,
			Type:     accountType,
			Icon:     icon,
			IconType: core.ICON_TYPE_SYSTEM,
			Color:    color,
			Currency: currency,
			Balance:  mainAccountBalance,
			Comment:  req.Comment,
			Extend:   accountExtend,
		},
		mainAccountBalanceTime: mcpAccountGetBalanceTimeUnix(mainAccountBalanceTime, mainAccountBalance),
	}

	if accountType == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		plan.childrenAccounts = make([]*models.Account, len(req.SubAccounts))
		plan.childrenAccountBalanceTimes = make([]int64, len(req.SubAccounts))

		for i := 0; i < len(req.SubAccounts); i++ {
			subAccount, subAccountBalanceTime, err := mcpAccountBuildNewSubAccountModel(uid, "sub_accounts["+strconv.Itoa(i)+"].", req.SubAccounts[i], category, int32(i+1))

			if err != nil {
				return nil, err
			}

			plan.childrenAccounts[i] = subAccount
			plan.childrenAccountBalanceTimes[i] = mcpAccountGetBalanceTimeUnix(subAccountBalanceTime, subAccount.Balance)

			if clientTimezone == nil && subAccountBalanceTime != nil {
				clientTimezone = subAccountBalanceTime.Location()
			}
		}
	}

	if clientTimezone == nil {
		clientTimezone = time.Local
	}

	plan.clientTimezone = clientTimezone

	return plan, nil
}

