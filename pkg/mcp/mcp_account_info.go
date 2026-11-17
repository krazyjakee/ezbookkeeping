package mcp

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
)

// Textual account types used in MCP tools
const (
	mcpAccountTypeSingleAccount    = "single_account"
	mcpAccountTypeMultiSubAccounts = "multi_sub_accounts"
)

// Default values of account icon and color, which are the same as the web interface
const (
	mcpAccountDefaultIconId = int64(1)
	mcpAccountDefaultColor  = "000000"
)

// mcpAccountMaxCreditCardStatementDate is the maximum statement date of credit card account (the same as the binding tag of AccountCreateRequest)
const mcpAccountMaxCreditCardStatementDate = 28

// MCPAccountInfo defines the structure of account information (the account may be a top-level account or a sub-account)
type MCPAccountInfo struct {
	Id                      string                      `json:"id,omitempty" jsonschema_description:"Stable account ID (not returned for dry run results of accounts not created yet)"`
	Name                    string                      `json:"name" jsonschema_description:"Account name"`
	ParentId                string                      `json:"parent_id,omitempty" jsonschema_description:"Parent account ID (only for sub-accounts)"`
	Category                string                      `json:"category" jsonschema:"enum=cash,enum=checking_account,enum=savings_account,enum=credit_card,enum=virtual,enum=debt,enum=receivables,enum=certificate_of_deposit,enum=investment" jsonschema_description:"Account category"`
	Type                    string                      `json:"type" jsonschema:"enum=single_account,enum=multi_sub_accounts" jsonschema_description:"Account type (single_account, or multi_sub_accounts for a parent account whose balances are held by its sub-accounts)"`
	IsAsset                 bool                        `json:"is_asset" jsonschema_description:"Whether the account is an asset account"`
	IsLiability             bool                        `json:"is_liability" jsonschema_description:"Whether the account is a liability account (credit_card or debt)"`
	Currency                string                      `json:"currency" jsonschema_description:"Currency code of the account (e.g. USD, GBP), or --- if not set (only for parent accounts with sub-accounts)"`
	Balance                 string                      `json:"balance" jsonschema_description:"Signed accounting balance. For asset accounts a positive value is money held; for liability accounts a negative value is money owed (e.g. -120.00 means 120.00 is owed)"`
	DisplayBalance          string                      `json:"display_balance" jsonschema_description:"Balance as displayed in the web interface: equals balance for asset accounts, and equals outstanding_balance for liability accounts"`
	OutstandingBalance      string                      `json:"outstanding_balance,omitempty" jsonschema_description:"Outstanding balance for liability accounts only, i.e. the negated signed balance (positive means money owed, e.g. 120.00 means 120.00 is owed)"`
	Icon                    string                      `json:"icon" jsonschema_description:"Icon ID of the account"`
	Color                   string                      `json:"color" jsonschema_description:"Icon color of the account in hex RGB format (e.g. 000000)"`
	Comment                 string                      `json:"comment,omitempty" jsonschema_description:"Account description"`
	Hidden                  bool                        `json:"hidden" jsonschema_description:"Whether the account is hidden"`
	DisplayOrder            int32                       `json:"display_order" jsonschema_description:"Display order of the account in its category (or in its parent account)"`
	CreditCardStatementDate *int                        `json:"credit_card_statement_date,omitempty" jsonschema_description:"Statement date of credit card account (only for top-level credit card accounts, 0 means not set)"`
	CreditCardLimit         string                      `json:"credit_card_limit,omitempty" jsonschema_description:"Credit limit of credit card account (only for top-level credit card accounts, 0.00 means not set)"`
	LastReconciledTime      string                      `json:"last_reconciled_time,omitempty" jsonschema_description:"Last reconciled time of the account in RFC 3339 format (only if set)"`
	SubAccounts             []*MCPAccountSubAccountInfo `json:"sub_accounts,omitempty" jsonschema_description:"Sub-accounts of the account (only for accounts of multi_sub_accounts type)"`
}

// MCPAccountSubAccountInfo defines the structure of sub-account information
type MCPAccountSubAccountInfo struct {
	Id                 string `json:"id,omitempty" jsonschema_description:"Stable sub-account ID (not returned for dry run results of sub-accounts not created yet)"`
	Name               string `json:"name" jsonschema_description:"Sub-account name"`
	ParentId           string `json:"parent_id,omitempty" jsonschema_description:"Parent account ID"`
	Category           string `json:"category" jsonschema_description:"Account category (always the same as the parent account)"`
	IsAsset            bool   `json:"is_asset" jsonschema_description:"Whether the sub-account is an asset account"`
	IsLiability        bool   `json:"is_liability" jsonschema_description:"Whether the sub-account is a liability account (credit_card or debt)"`
	Currency           string `json:"currency" jsonschema_description:"Currency code of the sub-account"`
	Balance            string `json:"balance" jsonschema_description:"Signed accounting balance (negative means money owed for liability accounts)"`
	DisplayBalance     string `json:"display_balance" jsonschema_description:"Balance as displayed in the web interface (outstanding balance for liability accounts)"`
	OutstandingBalance string `json:"outstanding_balance,omitempty" jsonschema_description:"Outstanding balance for liability accounts only (positive means money owed)"`
	Icon               string `json:"icon" jsonschema_description:"Icon ID of the sub-account"`
	Color              string `json:"color" jsonschema_description:"Icon color of the sub-account in hex RGB format"`
	Comment            string `json:"comment,omitempty" jsonschema_description:"Sub-account description"`
	Hidden             bool   `json:"hidden" jsonschema_description:"Whether the sub-account is hidden"`
	DisplayOrder       int32  `json:"display_order" jsonschema_description:"Display order of the sub-account in its parent account"`
	LastReconciledTime string `json:"last_reconciled_time,omitempty" jsonschema_description:"Last reconciled time of the sub-account in RFC 3339 format (only if set)"`
}

// MCPCreateAccountSubAccountRequest represents the parameters of a sub-account to be created
type MCPCreateAccountSubAccountRequest struct {
	Name               string `json:"name" jsonschema_description:"Sub-account name"`
	Currency           string `json:"currency" jsonschema_description:"Currency code of the sub-account in ISO 4217 format (e.g. USD, GBP)"`
	Comment            string `json:"comment,omitempty" jsonschema_description:"Sub-account description (optional)"`
	Icon               string `json:"icon,omitempty" jsonschema_description:"Built-in icon ID (optional, default: 1)"`
	Color              string `json:"color,omitempty" jsonschema_description:"Icon color in 6-digit hex RGB format (optional, default: 000000)"`
	OpeningBalance     string `json:"opening_balance,omitempty" jsonschema_description:"Signed opening balance (optional, e.g. 100.00; for liability accounts a negative value means money owed). Do not set together with outstanding_balance"`
	OutstandingBalance string `json:"outstanding_balance,omitempty" jsonschema_description:"Opening outstanding balance for liability accounts only (optional, positive means money owed, e.g. 120.00 is stored as signed balance -120.00). Do not set together with opening_balance"`
	OpeningBalanceTime string `json:"opening_balance_time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Effective time of the opening balance in RFC 3339 format (required if the opening balance is not zero), the opening balance is recorded as a balance modification transaction at this time"`
}

// mcpAccountCreatePlan represents the validated account models to be created
type mcpAccountCreatePlan struct {
	mainAccount                 *models.Account
	mainAccountBalanceTime      int64
	childrenAccounts            []*models.Account
	childrenAccountBalanceTimes []int64
	clientTimezone              *time.Location
}

// mcpAccountNewResolver returns an entity resolver for accounts
func mcpAccountNewResolver() *mcpEntityResolver[*models.Account] {
	return newMCPEntityResolver[*models.Account]("account",
		func(account *models.Account) int64 {
			return account.AccountId
		},
		func(account *models.Account) string {
			return account.Name
		},
		func(account *models.Account) string {
			description := fmt.Sprintf("category: %s, currency: %s", getMCPAccountCategoryName(account.Category), account.Currency)

			if account.ParentAccountId > models.LevelOneAccountParentId {
				description = description + ", parent_id: " + utils.Int64ToString(account.ParentAccountId)
			}

			if account.Hidden {
				description = description + ", hidden"
			}

			return description
		},
	)
}

// mcpAccountGetTypeName returns the textual account type
func mcpAccountGetTypeName(accountType models.AccountType) string {
	if accountType == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return mcpAccountTypeMultiSubAccounts
	}

	return mcpAccountTypeSingleAccount
}

// mcpAccountGetLastReconciledTime returns the textual last reconciled time of the account, or empty if not set
func mcpAccountGetLastReconciledTime(account *models.Account) string {
	if account.Extend != nil && account.Extend.LastReconciledTime != nil && *account.Extend.LastReconciledTime > 0 {
		return formatMCPUnixTime(*account.Extend.LastReconciledTime)
	}

	return ""
}

// mcpAccountGetTextualId returns the textual account id, or empty if the account is not saved yet
func mcpAccountGetTextualId(accountId int64) string {
	if accountId <= 0 {
		return ""
	}

	return utils.Int64ToString(accountId)
}

// mcpAccountBuildSubAccountInfo returns the sub-account information of the account model
func mcpAccountBuildSubAccountInfo(account *models.Account) *MCPAccountSubAccountInfo {
	displayBalance, signedBalance, outstandingBalance := mcpFormatAccountBalances(account.Category, big.NewInt(account.Balance))

	return &MCPAccountSubAccountInfo{
		Id:                 mcpAccountGetTextualId(account.AccountId),
		Name:               account.Name,
		ParentId:           mcpAccountGetTextualId(account.ParentAccountId),
		Category:           getMCPAccountCategoryName(account.Category),
		IsAsset:            account.Category.IsAsset(),
		IsLiability:        account.Category.IsLiability(),
		Currency:           account.Currency,
		Balance:            signedBalance,
		DisplayBalance:     displayBalance,
		OutstandingBalance: outstandingBalance,
		Icon:               utils.Int64ToString(account.Icon),
		Color:              account.Color,
		Comment:            account.Comment,
		Hidden:             account.Hidden,
		DisplayOrder:       account.DisplayOrder,
		LastReconciledTime: mcpAccountGetLastReconciledTime(account),
	}
}

// mcpAccountBuildInfo returns the account information of the account model and its sub-account models
func mcpAccountBuildInfo(account *models.Account, subAccounts []*models.Account, includeHiddenSubAccounts bool) *MCPAccountInfo {
	displayBalance, signedBalance, outstandingBalance := mcpFormatAccountBalances(account.Category, big.NewInt(account.Balance))

	accountInfo := &MCPAccountInfo{
		Id:                 mcpAccountGetTextualId(account.AccountId),
		Name:               account.Name,
		ParentId:           mcpAccountGetTextualId(account.ParentAccountId),
		Category:           getMCPAccountCategoryName(account.Category),
		Type:               mcpAccountGetTypeName(account.Type),
		IsAsset:            account.Category.IsAsset(),
		IsLiability:        account.Category.IsLiability(),
		Currency:           account.Currency,
		Balance:            signedBalance,
		DisplayBalance:     displayBalance,
		OutstandingBalance: outstandingBalance,
		Icon:               utils.Int64ToString(account.Icon),
		Color:              account.Color,
		Comment:            account.Comment,
		Hidden:             account.Hidden,
		DisplayOrder:       account.DisplayOrder,
		LastReconciledTime: mcpAccountGetLastReconciledTime(account),
	}

	if account.ParentAccountId == models.LevelOneAccountParentId && account.Category == models.ACCOUNT_CATEGORY_CREDIT_CARD {
		statementDate := 0
		creditLimit := int64(0)

		if account.Extend != nil && account.Extend.CreditCardStatementDate != nil {
			statementDate = *account.Extend.CreditCardStatementDate
		}

		if account.Extend != nil && account.Extend.CreditCardLimit != nil {
			creditLimit = *account.Extend.CreditCardLimit
		}

		accountInfo.CreditCardStatementDate = &statementDate
		accountInfo.CreditCardLimit = utils.FormatAmount(creditLimit)
	}

	sortedSubAccounts := make([]*models.Account, 0, len(subAccounts))

	for i := 0; i < len(subAccounts); i++ {
		if subAccounts[i] == nil || subAccounts[i] == account {
			continue
		}

		if subAccounts[i].AccountId > 0 && subAccounts[i].ParentAccountId != account.AccountId {
			continue
		}

		if !includeHiddenSubAccounts && subAccounts[i].Hidden {
			continue
		}

		sortedSubAccounts = append(sortedSubAccounts, subAccounts[i])
	}

	sort.SliceStable(sortedSubAccounts, func(i, j int) bool {
		return sortedSubAccounts[i].DisplayOrder < sortedSubAccounts[j].DisplayOrder
	})

	for i := 0; i < len(sortedSubAccounts); i++ {
		accountInfo.SubAccounts = append(accountInfo.SubAccounts, mcpAccountBuildSubAccountInfo(sortedSubAccounts[i]))
	}

	return accountInfo
}

// mcpAccountBuildInfoList returns the hierarchical account information list (top-level accounts with nested sub-accounts),
// sorted by category and display order as the web interface does
func mcpAccountBuildInfoList(accounts []*models.Account, includeHidden bool, category models.AccountCategory) []*MCPAccountInfo {
	subAccountsMap := make(map[int64][]*models.Account)
	topLevelAccounts := make([]*models.Account, 0, len(accounts))

	for i := 0; i < len(accounts); i++ {
		account := accounts[i]

		if account.ParentAccountId == models.LevelOneAccountParentId {
			topLevelAccounts = append(topLevelAccounts, account)
		} else {
			subAccountsMap[account.ParentAccountId] = append(subAccountsMap[account.ParentAccountId], account)
		}
	}

	sort.SliceStable(topLevelAccounts, func(i, j int) bool {
		if topLevelAccounts[i].Category != topLevelAccounts[j].Category {
			return topLevelAccounts[i].Category < topLevelAccounts[j].Category
		}

		return topLevelAccounts[i].DisplayOrder < topLevelAccounts[j].DisplayOrder
	})

	result := make([]*MCPAccountInfo, 0, len(topLevelAccounts))

	for i := 0; i < len(topLevelAccounts); i++ {
		account := topLevelAccounts[i]

		if !includeHidden && account.Hidden {
			continue
		}

		if category > 0 && account.Category != category {
			continue
		}

		result = append(result, mcpAccountBuildInfo(account, subAccountsMap[account.AccountId], includeHidden))
	}

	return result
}

// mcpAccountBuildInfoFromList returns the account information of the specified account id from the list containing the account and its sub-accounts
func mcpAccountBuildInfoFromList(accountAndSubAccounts []*models.Account, accountId int64) (*MCPAccountInfo, error) {
	var account *models.Account

	for i := 0; i < len(accountAndSubAccounts); i++ {
		if accountAndSubAccounts[i].AccountId == accountId {
			account = accountAndSubAccounts[i]
			break
		}
	}

	if account == nil {
		return nil, errs.ErrAccountNotFound
	}

	return mcpAccountBuildInfo(account, accountAndSubAccounts, true), nil
}

// mcpAccountParseIcon parses the textual built-in icon id, returns the default icon id if empty
func mcpAccountParseIcon(field string, value string) (int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return mcpAccountDefaultIconId, nil
	}

	iconId, err := utils.StringToInt64(value)

	if err != nil || iconId < 1 {
		return 0, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a positive built-in icon ID in string format (e.g. \"1\")")
	}

	return iconId, nil
}

// mcpAccountParseColor parses the textual hex RGB color, returns the default color if empty
func mcpAccountParseColor(field string, value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return mcpAccountDefaultColor, nil
	}

	if len(value) != 6 || !utils.IsValidHexRGBColor(value) {
		return "", NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a 6-digit hex RGB color (e.g. 000000)")
	}

	return value, nil
}

// mcpAccountParseCurrency checks the currency code, the not set value (---) is allowed only if allowNotSet is true
func mcpAccountParseCurrency(field string, value string, allowNotSet bool) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		if allowNotSet {
			return core.AccountCurrencyNotSetValue, nil
		}

		return "", NewMCPRequiredFieldError(field)
	}

	if value == core.AccountCurrencyNotSetValue {
		if allowNotSet {
			return value, nil
		}

		return "", NewMCPInvalidFieldError(field, value, "\""+field+"\" must be set to a valid ISO 4217 currency code for this account")
	}

	if _, exists := validators.AllCurrencyNames[value]; !exists {
		return "", NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a supported ISO 4217 currency code (e.g. USD, GBP)")
	}

	return value, nil
}

// mcpAccountParseOpeningBalance parses the opening balance or outstanding balance (for liability accounts only) to the signed accounting balance
func mcpAccountParseOpeningBalance(fieldPrefix string, openingBalance string, outstandingBalance string, category models.AccountCategory) (int64, error) {
	openingBalance = strings.TrimSpace(openingBalance)
	outstandingBalance = strings.TrimSpace(outstandingBalance)

	if openingBalance != "" && outstandingBalance != "" {
		return 0, NewMCPInvalidFieldError(fieldPrefix+"outstanding_balance", outstandingBalance, "only one of \""+fieldPrefix+"opening_balance\" and \""+fieldPrefix+"outstanding_balance\" can be set")
	}

	if outstandingBalance != "" {
		if !category.IsLiability() {
			return 0, NewMCPNotSupportedError(fieldPrefix+"outstanding_balance", "\""+fieldPrefix+"outstanding_balance\" can only be set for liability accounts (credit_card, debt), use \""+fieldPrefix+"opening_balance\" instead")
		}

		amount, err := parseMCPAmount(fieldPrefix+"outstanding_balance", outstandingBalance, false)

		if err != nil {
			return 0, err
		}

		return -amount, nil
	}

	return parseMCPAmount(fieldPrefix+"opening_balance", openingBalance, false)
}

// mcpAccountParseBalanceTime parses the opening balance time, it is required if the balance is not zero
func mcpAccountParseBalanceTime(field string, value string, balance int64) (*time.Time, error) {
	balanceTime, err := parseMCPTime(field, value, false)

	if err != nil {
		return nil, err
	}

	if balance != 0 && balanceTime == nil {
		return nil, NewMCPToolError(MCPToolErrorCodeRequired, field, "\""+field+"\" is required when the opening balance is not zero")
	}

	if balanceTime != nil && balanceTime.Unix() <= 0 {
		return nil, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be after 1970-01-01T00:00:00Z")
	}

	return balanceTime, nil
}

// mcpAccountBuildNewSubAccountModel validates the sub-account parameters and returns the new sub-account model and its opening balance time
func mcpAccountBuildNewSubAccountModel(uid int64, fieldPrefix string, subAccountReq *MCPCreateAccountSubAccountRequest, category models.AccountCategory, displayOrder int32) (*models.Account, *time.Time, error) {
	if subAccountReq == nil {
		return nil, nil, NewMCPRequiredFieldError(strings.TrimSuffix(fieldPrefix, "."))
	}

	if err := checkMCPStringLength(fieldPrefix+"name", subAccountReq.Name, true, mcpAccountNameMaxLength); err != nil {
		return nil, nil, err
	}

	if err := checkMCPStringLength(fieldPrefix+"comment", subAccountReq.Comment, false, mcpAccountCommentMaxLength); err != nil {
		return nil, nil, err
	}

	currency, err := mcpAccountParseCurrency(fieldPrefix+"currency", subAccountReq.Currency, false)

	if err != nil {
		return nil, nil, err
	}

	icon, err := mcpAccountParseIcon(fieldPrefix+"icon", subAccountReq.Icon)

	if err != nil {
		return nil, nil, err
	}

	color, err := mcpAccountParseColor(fieldPrefix+"color", subAccountReq.Color)

	if err != nil {
		return nil, nil, err
	}

	balance, err := mcpAccountParseOpeningBalance(fieldPrefix, subAccountReq.OpeningBalance, subAccountReq.OutstandingBalance, category)

	if err != nil {
		return nil, nil, err
	}

	balanceTime, err := mcpAccountParseBalanceTime(fieldPrefix+"opening_balance_time", subAccountReq.OpeningBalanceTime, balance)

	if err != nil {
		return nil, nil, err
	}

	subAccount := &models.Account{
		Uid:          uid,
		Name:         subAccountReq.Name,
		DisplayOrder: displayOrder,
		Category:     category,
		Type:         models.ACCOUNT_TYPE_SINGLE_ACCOUNT,
		Icon:         icon,
		IconType:     core.ICON_TYPE_SYSTEM,
		Color:        color,
		Currency:     currency,
		Balance:      balance,
		Comment:      subAccountReq.Comment,
		Extend:       &models.AccountExtend{},
	}

	return subAccount, balanceTime, nil
}

// mcpAccountGetBalanceTimeUnix returns the unix time of the balance time if the balance is not zero, otherwise returns 0
func mcpAccountGetBalanceTimeUnix(balanceTime *time.Time, balance int64) int64 {
	if balanceTime == nil || balance == 0 {
		return 0
	}

	return balanceTime.Unix()
}

// mcpAccountFindExistingForSkip returns the existing top-level account with the same name and category, returns nil if not exists,
// or returns an ambiguous name error if more than one account matches
func mcpAccountFindExistingForSkip(accounts []*models.Account, name string, category models.AccountCategory) (*models.Account, error) {
	candidates := make([]*models.Account, 0)

	for i := 0; i < len(accounts); i++ {
		if accounts[i].ParentAccountId == models.LevelOneAccountParentId && accounts[i].Category == category {
			candidates = append(candidates, accounts[i])
		}
	}

	resolver := mcpAccountNewResolver()
	matches := resolver.FindAllByName(candidates, name)

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, NewMCPAmbiguousNameError("name", "account", name, resolver.GetCandidates(matches))
	}

	return nil, nil
}

// mcpAccountConvertServiceError converts the account in-use errors of the service to the user-correctable MCP tool errors
func mcpAccountConvertServiceError(err error) error {
	if errors.Is(err, errs.ErrAccountInUseCannotBeDeleted) || errors.Is(err, errs.ErrSubAccountInUseCannotBeDeleted) {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "account_id", "the account cannot be deleted because it (or one of its sub-accounts) is still used by transactions other than the opening balance, or by transaction templates or active scheduled transactions; move or delete them first, or hide the account instead")
	} else if errors.Is(err, errs.ErrAccountHaveNoSubAccount) {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, "account_id", "the sub-account cannot be deleted because it is the last sub-account of its parent account; delete the parent account instead")
	}

	return err
}
