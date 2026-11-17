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
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPUpdateAccountRequest represents all parameters of the update account request
type MCPUpdateAccountRequest struct {
	AccountId               string                               `json:"account_id,omitempty" jsonschema_description:"ID of the account or sub-account to update (preferred, either account_id or account_name is required)"`
	AccountName             string                               `json:"account_name,omitempty" jsonschema_description:"Name of the account or sub-account to update (used if account_id is not set, returns an ambiguous_name error with candidates if more than one account has this name)"`
	Name                    *string                              `json:"name,omitempty" jsonschema_description:"New account name (optional)"`
	Category                *string                              `json:"category,omitempty" jsonschema:"enum=cash,enum=checking_account,enum=savings_account,enum=credit_card,enum=virtual,enum=debt,enum=receivables,enum=certificate_of_deposit,enum=investment" jsonschema_description:"New account category (optional, top-level accounts only, the sub-accounts of a parent account are changed together)"`
	Currency                *string                              `json:"currency,omitempty" jsonschema_description:"New currency code (optional, can only be changed for credit_card parent accounts with sub-accounts, the currency of other accounts cannot be changed)"`
	Comment                 *string                              `json:"comment,omitempty" jsonschema_description:"New account description (optional)"`
	Icon                    *string                              `json:"icon,omitempty" jsonschema_description:"New built-in icon ID (optional)"`
	Color                   *string                              `json:"color,omitempty" jsonschema_description:"New icon color in 6-digit hex RGB format (optional)"`
	CreditCardStatementDate *int                                 `json:"credit_card_statement_date,omitempty" jsonschema:"minimum=0,maximum=28" jsonschema_description:"New statement date of the credit card (optional, 0-28, 0 means not set, only for top-level credit_card accounts)"`
	CreditCardLimit         *string                              `json:"credit_card_limit,omitempty" jsonschema_description:"New credit limit of the credit card (optional, positive amount, or empty string to clear, only for top-level credit_card accounts)"`
	Hidden                  *bool                                `json:"hidden,omitempty" jsonschema_description:"Whether the account is hidden (optional)"`
	AddSubAccounts          []*MCPCreateAccountSubAccountRequest `json:"add_sub_accounts,omitempty" jsonschema_description:"New sub-accounts to add (optional, only for parent accounts of multi_sub_accounts type)"`
	DryRun                  bool                                 `json:"dry_run,omitempty" jsonschema_description:"If true, the changes are only validated with the same rules as the real update and not saved (optional)"`
}

// MCPUpdateAccountResponse represents the response structure for update account
type MCPUpdateAccountResponse struct {
	Success            bool            `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun             bool            `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (if true, nothing is saved)"`
	Status             string          `json:"status" jsonschema:"enum=updated,enum=unchanged,enum=would_be_updated" jsonschema_description:"Result status: updated (committed), unchanged (nothing needs to be updated, nothing is saved), would_be_updated (dry run, nothing is saved)"`
	Account            *MCPAccountInfo `json:"account" jsonschema_description:"The persisted account after update (re-read), or the would-be account for dry run"`
	AddedSubAccountIds []string        `json:"added_sub_account_ids,omitempty" jsonschema_description:"IDs of the newly added sub-accounts (only for committed updates)"`
}

// mcpAccountUpdatePlan represents the validated account models to be updated
type mcpAccountUpdatePlan struct {
	mainAccount               *models.Account
	targetAccount             *models.Account
	updatedTargetAccount      *models.Account
	toUpdateAccounts          []*models.Account
	toAddAccounts             []*models.Account
	toAddAccountBalanceTimes  []int64
	updateMainAccountCurrency bool
	categoryChanged           bool
	clientTimezone            *time.Location
}

type mcpUpdateAccountToolHandler struct{}

var MCPUpdateAccountToolHandler = &mcpUpdateAccountToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateAccountToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_ACCOUNT
}

// Name returns the name of the MCP tool
func (h *mcpUpdateAccountToolHandler) Name() string {
	return "update_account"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateAccountToolHandler) Description() string {
	return "Update an existing account or sub-account in ezBookkeeping (name, category, icon, color, description, visibility, credit card settings) or add sub-accounts to a parent account. Only the specified fields are changed. " +
		"The balance and the opening balance time of existing accounts cannot be changed (the same as the web interface), record transactions to change balances instead. Supports dry_run."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateAccountToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateAccountToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateAccountToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateAccountToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateAccountRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateAccountToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateAccountResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateAccountToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	updateAccountRequest, err := parseMCPToolArguments[MCPUpdateAccountRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allAccounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[update_account_tool_handler.Handle] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	targetAccount, err := mcpAccountNewResolver().Resolve(allAccounts, "account_id", updateAccountRequest.AccountId, "account_name", updateAccountRequest.AccountName)

	if err != nil {
		return nil, nil, err
	}

	plan, err := h.buildUpdatePlan(uid, updateAccountRequest, targetAccount, allAccounts)

	if err != nil {
		return nil, nil, err
	}

	if len(plan.toUpdateAccounts) < 1 && len(plan.toAddAccounts) < 1 {
		return newMCPJSONTextResult(&MCPUpdateAccountResponse{
			Success: true,
			DryRun:  updateAccountRequest.DryRun,
			Status:  MCPWriteStatusUnchanged,
			Account: mcpAccountBuildInfo(targetAccount, allAccounts, true),
		})
	}

	if updateAccountRequest.DryRun {
		return newMCPJSONTextResult(&MCPUpdateAccountResponse{
			Success: true,
			DryRun:  true,
			Status:  MCPWriteStatusWouldBeUpdated,
			Account: mcpAccountBuildInfo(plan.updatedTargetAccount, h.getSimulatedAccounts(plan, allAccounts), true),
		})
	}

	if plan.categoryChanged && plan.updatedTargetAccount.ParentAccountId == models.LevelOneAccountParentId {
		maxOrderId, err := services.GetAccountService().GetMaxDisplayOrder(c, uid, plan.updatedTargetAccount.Category)

		if err != nil {
			log.Errorf(c, "[update_account_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		plan.updatedTargetAccount.DisplayOrder = maxOrderId + 1
	}

	err = services.GetAccountService().ModifyAccounts(c, plan.mainAccount, plan.toUpdateAccounts, plan.toAddAccounts, plan.toAddAccountBalanceTimes, nil, plan.updateMainAccountCurrency, plan.clientTimezone)

	if err != nil {
		log.Errorf(c, "[update_account_tool_handler.Handle] failed to update account \"id:%d\" for user \"uid:%d\", because %s", targetAccount.AccountId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[update_account_tool_handler.Handle] user \"uid:%d\" has updated account \"id:%d\" successfully", uid, targetAccount.AccountId)

	addedSubAccountIds := make([]string, 0, len(plan.toAddAccounts))

	for i := 0; i < len(plan.toAddAccounts); i++ {
		addedSubAccountIds = append(addedSubAccountIds, utils.Int64ToString(plan.toAddAccounts[i].AccountId))
	}

	accountInfo := mcpAccountBuildInfo(plan.updatedTargetAccount, h.getSimulatedAccounts(plan, allAccounts), true)
	accountAndSubAccounts, err := services.GetAccountService().GetAccountAndSubAccountsByAccountId(c, uid, targetAccount.AccountId)

	if err != nil {
		log.Warnf(c, "[update_account_tool_handler.Handle] failed to get latest account \"id:%d\" info after account updated, because %s", targetAccount.AccountId, err.Error())
	} else if latestAccountInfo, err := mcpAccountBuildInfoFromList(accountAndSubAccounts, targetAccount.AccountId); err == nil {
		accountInfo = latestAccountInfo
	}

	return newMCPJSONTextResult(&MCPUpdateAccountResponse{
		Success:            true,
		DryRun:             false,
		Status:             MCPWriteStatusUpdated,
		Account:            accountInfo,
		AddedSubAccountIds: addedSubAccountIds,
	})
}

// buildUpdatePlan validates the request with the same rules as the account modification api and returns the account models to be updated
func (h *mcpUpdateAccountToolHandler) buildUpdatePlan(uid int64, req *MCPUpdateAccountRequest, targetAccount *models.Account, allAccounts []*models.Account) (*mcpAccountUpdatePlan, error) {
	isSubAccount := targetAccount.ParentAccountId != models.LevelOneAccountParentId
	mainAccount := targetAccount
	subAccounts := make([]*models.Account, 0)

	if isSubAccount {
		mainAccount = nil

		for i := 0; i < len(allAccounts); i++ {
			if allAccounts[i].AccountId == targetAccount.ParentAccountId {
				mainAccount = allAccounts[i]
				break
			}
		}

		if mainAccount == nil {
			return nil, NewMCPNotFoundError("account_id", "parent account", utils.Int64ToString(targetAccount.ParentAccountId))
		}
	} else {
		for i := 0; i < len(allAccounts); i++ {
			if allAccounts[i].ParentAccountId == targetAccount.AccountId {
				subAccounts = append(subAccounts, allAccounts[i])
			}
		}
	}

	newAccount := mcpAccountCloneModel(targetAccount)
	plan := &mcpAccountUpdatePlan{
		mainAccount:          mainAccount,
		targetAccount:        targetAccount,
		updatedTargetAccount: newAccount,
	}

	if req.Name != nil {
		if err := checkMCPStringLength("name", *req.Name, true, mcpAccountNameMaxLength); err != nil {
			return nil, err
		}

		newAccount.Name = *req.Name
	}

	if req.Comment != nil {
		if err := checkMCPStringLength("comment", *req.Comment, false, mcpAccountCommentMaxLength); err != nil {
			return nil, err
		}

		newAccount.Comment = *req.Comment
	}

	if req.Icon != nil {
		if strings.TrimSpace(*req.Icon) == "" {
			return nil, NewMCPRequiredFieldError("icon")
		}

		icon, err := mcpAccountParseIcon("icon", *req.Icon)

		if err != nil {
			return nil, err
		}

		newAccount.Icon = icon
		newAccount.IconType = core.ICON_TYPE_SYSTEM
	}

	if req.Color != nil {
		if strings.TrimSpace(*req.Color) == "" {
			return nil, NewMCPRequiredFieldError("color")
		}

		color, err := mcpAccountParseColor("color", *req.Color)

		if err != nil {
			return nil, err
		}

		newAccount.Color = color
	}

	if req.Hidden != nil {
		newAccount.Hidden = *req.Hidden
	}

	if req.Category != nil {
		category, err := parseMCPAccountCategory("category", strings.TrimSpace(*req.Category))

		if err != nil {
			return nil, err
		}

		if isSubAccount && category != targetAccount.Category {
			return nil, NewMCPNotSupportedError("category", "the category of a sub-account is always the same as its parent account, update the category of the parent account instead")
		}

		newAccount.Category = category
	}

	plan.categoryChanged = newAccount.Category != targetAccount.Category

	if req.Currency != nil {
		currency := strings.TrimSpace(*req.Currency)

		if !isSubAccount && targetAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS && newAccount.Category == models.ACCOUNT_CATEGORY_CREDIT_CARD {
			parsedCurrency, err := mcpAccountParseCurrency("currency", currency, true)

			if err != nil {
				return nil, err
			}

			if parsedCurrency != targetAccount.Currency {
				newAccount.Currency = parsedCurrency
				plan.updateMainAccountCurrency = true
			}
		} else if currency != targetAccount.Currency {
			return nil, NewMCPNotSupportedError("currency", "the currency of this account cannot be changed")
		}
	}

	isCreditCardMainAccount := !isSubAccount && newAccount.Category == models.ACCOUNT_CATEGORY_CREDIT_CARD

	if req.CreditCardStatementDate != nil {
		if *req.CreditCardStatementDate != 0 && !isCreditCardMainAccount {
			return nil, NewMCPNotSupportedError("credit_card_statement_date", "\"credit_card_statement_date\" can only be set for top-level credit_card accounts")
		}

		if *req.CreditCardStatementDate < 0 || *req.CreditCardStatementDate > mcpAccountMaxCreditCardStatementDate {
			toolErr := NewMCPToolError(MCPToolErrorCodeOutOfRange, "credit_card_statement_date", "\"credit_card_statement_date\" must be between 0 and "+strconv.Itoa(mcpAccountMaxCreditCardStatementDate))
			limit := int64(mcpAccountMaxCreditCardStatementDate)
			toolErr.Limit = &limit
			return nil, toolErr
		}
	}

	var newCreditCardLimit *int64

	if req.CreditCardLimit != nil {
		limitText := strings.TrimSpace(*req.CreditCardLimit)

		if limitText != "" && !isCreditCardMainAccount {
			return nil, NewMCPNotSupportedError("credit_card_limit", "\"credit_card_limit\" can only be set for top-level credit_card accounts")
		}

		creditLimit, err := parseMCPAmount("credit_card_limit", limitText, false)

		if err != nil {
			return nil, err
		}

		if limitText != "" && creditLimit <= 0 {
			return nil, NewMCPInvalidFieldError("credit_card_limit", limitText, "\"credit_card_limit\" must be a positive amount")
		}

		newCreditCardLimit = &creditLimit
	}

	if isCreditCardMainAccount {
		statementDate := 0
		creditLimit := int64(0)

		if targetAccount.Extend != nil && targetAccount.Extend.CreditCardStatementDate != nil {
			statementDate = *targetAccount.Extend.CreditCardStatementDate
		}

		if targetAccount.Extend != nil && targetAccount.Extend.CreditCardLimit != nil {
			creditLimit = *targetAccount.Extend.CreditCardLimit
		}

		if req.CreditCardStatementDate != nil {
			statementDate = *req.CreditCardStatementDate
		}

		if newCreditCardLimit != nil {
			creditLimit = *newCreditCardLimit
		}

		if targetAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS && newAccount.Currency == core.AccountCurrencyNotSetValue && creditLimit > 0 {
			return nil, NewMCPToolError(MCPToolErrorCodeRequired, "currency", "\"currency\" of the parent account must be set when the credit limit is set")
		}

		newAccount.Extend.CreditCardStatementDate = &statementDate
		newAccount.Extend.CreditCardLimit = &creditLimit
	} else {
		newAccount.Extend.CreditCardStatementDate = nil
		newAccount.Extend.CreditCardLimit = nil
	}

	if mcpAccountIsModelChanged(targetAccount, newAccount) {
		plan.toUpdateAccounts = append(plan.toUpdateAccounts, newAccount)

		if !isSubAccount && plan.categoryChanged {
			for i := 0; i < len(subAccounts); i++ {
				updatedSubAccount := mcpAccountCloneModel(subAccounts[i])
				updatedSubAccount.Category = newAccount.Category
				plan.toUpdateAccounts = append(plan.toUpdateAccounts, updatedSubAccount)
			}
		}
	}

	if len(req.AddSubAccounts) > 0 {
		if isSubAccount || targetAccount.Type != models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			return nil, NewMCPNotSupportedError("add_sub_accounts", "sub-accounts can only be added to a parent account of multi_sub_accounts type")
		}

		if len(req.AddSubAccounts) > mcpMaximumBatchItemsCount {
			return nil, NewMCPTooManyItemsError("add_sub_accounts", mcpMaximumBatchItemsCount, len(req.AddSubAccounts))
		}

		maxOrderId := int32(0)

		for i := 0; i < len(subAccounts); i++ {
			if subAccounts[i].DisplayOrder > maxOrderId {
				maxOrderId = subAccounts[i].DisplayOrder
			}
		}

		for i := 0; i < len(req.AddSubAccounts); i++ {
			maxOrderId++
			subAccount, subAccountBalanceTime, err := mcpAccountBuildNewSubAccountModel(uid, "add_sub_accounts["+strconv.Itoa(i)+"].", req.AddSubAccounts[i], newAccount.Category, maxOrderId)

			if err != nil {
				return nil, err
			}

			plan.toAddAccounts = append(plan.toAddAccounts, subAccount)
			plan.toAddAccountBalanceTimes = append(plan.toAddAccountBalanceTimes, mcpAccountGetBalanceTimeUnix(subAccountBalanceTime, subAccount.Balance))

			if plan.clientTimezone == nil && subAccountBalanceTime != nil {
				plan.clientTimezone = subAccountBalanceTime.Location()
			}
		}
	}

	if plan.clientTimezone == nil {
		plan.clientTimezone = time.Local
	}

	return plan, nil
}

// getSimulatedAccounts returns the account list after applying the update plan (for building the response)
func (h *mcpUpdateAccountToolHandler) getSimulatedAccounts(plan *mcpAccountUpdatePlan, allAccounts []*models.Account) []*models.Account {
	updatedAccounts := make(map[int64]*models.Account, len(plan.toUpdateAccounts))

	for i := 0; i < len(plan.toUpdateAccounts); i++ {
		updatedAccounts[plan.toUpdateAccounts[i].AccountId] = plan.toUpdateAccounts[i]
	}

	result := make([]*models.Account, 0, len(allAccounts)+len(plan.toAddAccounts))

	for i := 0; i < len(allAccounts); i++ {
		if updatedAccount, exists := updatedAccounts[allAccounts[i].AccountId]; exists {
			result = append(result, updatedAccount)
		} else {
			result = append(result, allAccounts[i])
		}
	}

	for i := 0; i < len(plan.toAddAccounts); i++ {
		result = append(result, plan.toAddAccounts[i])
	}

	return result
}

// mcpAccountCloneModel returns a copy of the account model (including a copy of the extend data)
func mcpAccountCloneModel(account *models.Account) *models.Account {
	newAccount := *account
	newExtend := &models.AccountExtend{}

	if account.Extend != nil {
		if account.Extend.LastReconciledTime != nil {
			lastReconciledTime := *account.Extend.LastReconciledTime
			newExtend.LastReconciledTime = &lastReconciledTime
		}

		if account.Extend.CreditCardStatementDate != nil {
			statementDate := *account.Extend.CreditCardStatementDate
			newExtend.CreditCardStatementDate = &statementDate
		}

		if account.Extend.CreditCardLimit != nil {
			creditLimit := *account.Extend.CreditCardLimit
			newExtend.CreditCardLimit = &creditLimit
		}
	}

	newAccount.Extend = newExtend
	return &newAccount
}

// mcpAccountIsModelChanged returns whether the updatable fields of the account model are changed
func mcpAccountIsModelChanged(oldAccount *models.Account, newAccount *models.Account) bool {
	if newAccount.Name != oldAccount.Name ||
		newAccount.Category != oldAccount.Category ||
		newAccount.Icon != oldAccount.Icon ||
		newAccount.IconType != oldAccount.IconType ||
		newAccount.Color != oldAccount.Color ||
		newAccount.Currency != oldAccount.Currency ||
		newAccount.Comment != oldAccount.Comment ||
		newAccount.Hidden != oldAccount.Hidden {
		return true
	}

	oldExtend := oldAccount.Extend

	if oldExtend == nil {
		oldExtend = &models.AccountExtend{}
	}

	newExtend := newAccount.Extend

	if newExtend == nil {
		newExtend = &models.AccountExtend{}
	}

	if !mcpAccountIsInt64PointerEqual(oldExtend.LastReconciledTime, newExtend.LastReconciledTime) {
		return true
	}

	// the credit card statement date and credit limit which are not set are equivalent to 0
	if mcpAccountGetInt64PointerValue(oldExtend.CreditCardLimit) != mcpAccountGetInt64PointerValue(newExtend.CreditCardLimit) {
		return true
	}

	oldStatementDate := 0
	newStatementDate := 0

	if oldExtend.CreditCardStatementDate != nil {
		oldStatementDate = *oldExtend.CreditCardStatementDate
	}

	if newExtend.CreditCardStatementDate != nil {
		newStatementDate = *newExtend.CreditCardStatementDate
	}

	return oldStatementDate != newStatementDate
}

// mcpAccountIsInt64PointerEqual returns whether the two int64 pointers are both nil or have the same value
func mcpAccountIsInt64PointerEqual(a *int64, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

// mcpAccountGetInt64PointerValue returns the value of the int64 pointer, or 0 if the pointer is nil
func mcpAccountGetInt64PointerValue(a *int64) int64 {
	if a == nil {
		return 0
	}

	return *a
}
