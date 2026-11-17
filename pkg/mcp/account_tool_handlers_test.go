package mcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func mcpAccountTestGetToolError(t *testing.T, err error) *MCPToolError {
	t.Helper()
	var toolErr *MCPToolError

	if !errors.As(err, &toolErr) {
		t.Fatalf("expected MCPToolError, but got %v", err)
	}

	return toolErr
}

func TestMCPAccountCreatePlan_SingleAccount(t *testing.T) {
	plan, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:               "Bank",
		Category:           "checking_account",
		Currency:           "GBP",
		OpeningBalance:     "100.50",
		OpeningBalanceTime: "2024-01-01T00:00:00+01:00",
	})

	assert.Nil(t, err)
	assert.Equal(t, "Bank", plan.mainAccount.Name)
	assert.Equal(t, models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, plan.mainAccount.Category)
	assert.Equal(t, models.ACCOUNT_TYPE_SINGLE_ACCOUNT, plan.mainAccount.Type)
	assert.Equal(t, int64(10050), plan.mainAccount.Balance)
	assert.Equal(t, mcpAccountDefaultIconId, plan.mainAccount.Icon)
	assert.Equal(t, core.ICON_TYPE_SYSTEM, plan.mainAccount.IconType)
	assert.Equal(t, mcpAccountDefaultColor, plan.mainAccount.Color)
	assert.Equal(t, int64(1704063600), plan.mainAccountBalanceTime)
	assert.Equal(t, 0, len(plan.childrenAccounts))
}

func TestMCPAccountCreatePlan_LiabilityOutstandingBalance(t *testing.T) {
	plan, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:               "Director's Loan",
		Category:           "debt",
		Currency:           "GBP",
		OutstandingBalance: "120",
		OpeningBalanceTime: "2024-01-01T00:00:00Z",
	})

	assert.Nil(t, err)
	assert.Equal(t, int64(-12000), plan.mainAccount.Balance)

	accountInfo := mcpAccountBuildInfo(plan.mainAccount, nil, true)
	assert.Equal(t, "", accountInfo.Id)
	assert.True(t, accountInfo.IsLiability)
	assert.Equal(t, "-120.00", accountInfo.Balance)
	assert.Equal(t, "120.00", accountInfo.OutstandingBalance)
	assert.Equal(t, "120.00", accountInfo.DisplayBalance)
}

func TestMCPAccountCreatePlan_OutstandingBalanceForAssetAccount(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:               "Cash",
		Category:           "cash",
		Currency:           "GBP",
		OutstandingBalance: "120",
		OpeningBalanceTime: "2024-01-01T00:00:00Z",
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "outstanding_balance", toolErr.Field)
}

func TestMCPAccountCreatePlan_BothBalances(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:               "Card",
		Category:           "credit_card",
		Currency:           "GBP",
		OpeningBalance:     "-120",
		OutstandingBalance: "120",
		OpeningBalanceTime: "2024-01-01T00:00:00Z",
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
}

func TestMCPAccountCreatePlan_BalanceTimeRequired(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:           "Cash",
		Category:       "cash",
		Currency:       "GBP",
		OpeningBalance: "1.00",
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "opening_balance_time", toolErr.Field)
}

func TestMCPAccountCreatePlan_NameTooLong(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     strings.Repeat("a", mcpAccountNameMaxLength+1),
		Category: "cash",
		Currency: "GBP",
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "name", toolErr.Field)
	assert.Equal(t, int64(mcpAccountNameMaxLength), *toolErr.Limit)
	assert.Equal(t, int64(mcpAccountNameMaxLength+1), *toolErr.Actual)
}

func TestMCPAccountCreatePlan_CommentTooLong(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Cash",
		Category: "cash",
		Currency: "GBP",
		Comment:  strings.Repeat("中", mcpAccountCommentMaxLength+1),
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "comment", toolErr.Field)
	assert.Equal(t, int64(mcpAccountCommentMaxLength), *toolErr.Limit)
}

func TestMCPAccountCreatePlan_InvalidCurrency(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Cash",
		Category: "cash",
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "currency", toolErr.Field)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Cash",
		Category: "cash",
		Currency: "XXXX",
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Cash",
		Category: "cash",
		Currency: core.AccountCurrencyNotSetValue,
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
}

func TestMCPAccountCreatePlan_CreditCardFieldsForNonCreditCard(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:                    "Cash",
		Category:                "cash",
		Currency:                "GBP",
		CreditCardStatementDate: 5,
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "credit_card_statement_date", toolErr.Field)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:            "Cash",
		Category:        "cash",
		Currency:        "GBP",
		CreditCardLimit: "1000",
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "credit_card_limit", toolErr.Field)
}

func TestMCPAccountCreatePlan_CreditCard(t *testing.T) {
	plan, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:                    "Card",
		Category:                "credit_card",
		Currency:                "USD",
		CreditCardStatementDate: 15,
		CreditCardLimit:         "5000",
	})

	assert.Nil(t, err)
	assert.Equal(t, 15, *plan.mainAccount.Extend.CreditCardStatementDate)
	assert.Equal(t, int64(500000), *plan.mainAccount.Extend.CreditCardLimit)

	accountInfo := mcpAccountBuildInfo(plan.mainAccount, nil, true)
	assert.Equal(t, 15, *accountInfo.CreditCardStatementDate)
	assert.Equal(t, "5000.00", accountInfo.CreditCardLimit)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:                    "Card",
		Category:                "credit_card",
		Currency:                "USD",
		CreditCardStatementDate: 29,
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeOutOfRange, toolErr.Code)
}

func TestMCPAccountCreatePlan_MultiSubAccounts(t *testing.T) {
	plan, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Bank",
		Category: "savings_account",
		SubAccounts: []*MCPCreateAccountSubAccountRequest{
			{
				Name:               "GBP",
				Currency:           "GBP",
				OpeningBalance:     "10",
				OpeningBalanceTime: "2024-01-01T00:00:00Z",
			},
			{
				Name:     "USD",
				Currency: "USD",
			},
		},
	})

	assert.Nil(t, err)
	assert.Equal(t, models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS, plan.mainAccount.Type)
	assert.Equal(t, core.AccountCurrencyNotSetValue, plan.mainAccount.Currency)
	assert.Equal(t, 2, len(plan.childrenAccounts))
	assert.Equal(t, models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, plan.childrenAccounts[1].Category)
	assert.Equal(t, int64(1000), plan.childrenAccounts[0].Balance)
	assert.Equal(t, int64(1704067200), plan.childrenAccountBalanceTimes[0])
	assert.Equal(t, int64(0), plan.childrenAccountBalanceTimes[1])

	accountInfo := mcpAccountBuildInfo(plan.mainAccount, plan.childrenAccounts, true)
	assert.Equal(t, mcpAccountTypeMultiSubAccounts, accountInfo.Type)
	assert.Equal(t, 2, len(accountInfo.SubAccounts))
	assert.Equal(t, "GBP", accountInfo.SubAccounts[0].Name)
}

func TestMCPAccountCreatePlan_MultiSubAccountsInvalid(t *testing.T) {
	_, err := MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Bank",
		Category: "savings_account",
		Currency: "GBP",
		SubAccounts: []*MCPCreateAccountSubAccountRequest{
			{Name: "GBP", Currency: "GBP"},
		},
	})

	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "currency", toolErr.Field)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:               "Bank",
		Category:           "savings_account",
		OpeningBalance:     "10",
		OpeningBalanceTime: "2024-01-01T00:00:00Z",
		SubAccounts: []*MCPCreateAccountSubAccountRequest{
			{Name: "GBP", Currency: "GBP"},
		},
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "opening_balance", toolErr.Field)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:     "Bank",
		Category: "savings_account",
		SubAccounts: []*MCPCreateAccountSubAccountRequest{
			{Name: "GBP"},
		},
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "sub_accounts[0].currency", toolErr.Field)

	_, err = MCPCreateAccountToolHandler.buildCreatePlan(1, &MCPCreateAccountRequest{
		Name:            "Card",
		Category:        "credit_card",
		CreditCardLimit: "100",
		SubAccounts: []*MCPCreateAccountSubAccountRequest{
			{Name: "GBP", Currency: "GBP"},
		},
	})

	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "currency", toolErr.Field)
}

func TestMCPAccountFindExistingForSkip(t *testing.T) {
	accounts := []*models.Account{
		{AccountId: 1, Name: "Bank", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT},
		{AccountId: 2, Name: "Loan", Category: models.ACCOUNT_CATEGORY_DEBT},
		{AccountId: 3, Name: "Loan", Category: models.ACCOUNT_CATEGORY_CASH},
		{AccountId: 4, Name: "Sub", Category: models.ACCOUNT_CATEGORY_DEBT, ParentAccountId: 2},
		{AccountId: 5, Name: "Dup", Category: models.ACCOUNT_CATEGORY_CASH},
		{AccountId: 6, Name: "Dup", Category: models.ACCOUNT_CATEGORY_CASH},
	}

	account, err := mcpAccountFindExistingForSkip(accounts, "Loan", models.ACCOUNT_CATEGORY_DEBT)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), account.AccountId)

	account, err = mcpAccountFindExistingForSkip(accounts, "Bank", models.ACCOUNT_CATEGORY_CASH)
	assert.Nil(t, err)
	assert.Nil(t, account)

	account, err = mcpAccountFindExistingForSkip(accounts, "Sub", models.ACCOUNT_CATEGORY_DEBT)
	assert.Nil(t, err)
	assert.Nil(t, account)

	_, err = mcpAccountFindExistingForSkip(accounts, "Dup", models.ACCOUNT_CATEGORY_CASH)
	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))
	assert.Equal(t, "5", toolErr.Candidates[0].Id)
	assert.Equal(t, "6", toolErr.Candidates[1].Id)
}

func TestMCPAccountBuildInfoList(t *testing.T) {
	accounts := []*models.Account{
		{AccountId: 1, Name: "Bank", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS, DisplayOrder: 1},
		{AccountId: 2, Name: "Cash", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, DisplayOrder: 1, Balance: 500},
		{AccountId: 3, Name: "Hidden", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, DisplayOrder: 2, Hidden: true},
		{AccountId: 4, Name: "USD", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1, DisplayOrder: 2},
		{AccountId: 5, Name: "GBP", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1, DisplayOrder: 1},
		{AccountId: 6, Name: "EUR", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1, DisplayOrder: 3, Hidden: true},
	}

	result := mcpAccountBuildInfoList(accounts, false, 0)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "2", result[0].Id)
	assert.Equal(t, "5.00", result[0].Balance)
	assert.Equal(t, "5.00", result[0].DisplayBalance)
	assert.Equal(t, "", result[0].OutstandingBalance)
	assert.Equal(t, "1", result[1].Id)
	assert.Equal(t, 2, len(result[1].SubAccounts))
	assert.Equal(t, "5", result[1].SubAccounts[0].Id)
	assert.Equal(t, "1", result[1].SubAccounts[0].ParentId)
	assert.Equal(t, "4", result[1].SubAccounts[1].Id)

	result = mcpAccountBuildInfoList(accounts, true, models.ACCOUNT_CATEGORY_CASH)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "3", result[1].Id)
	assert.True(t, result[1].Hidden)

	result = mcpAccountBuildInfoList(accounts, true, models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, 3, len(result[0].SubAccounts))
}

func TestMCPAccountUpdatePlan(t *testing.T) {
	statementDate := 10
	accounts := []*models.Account{
		{AccountId: 1, Uid: 1, Name: "Bank", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS, Currency: core.AccountCurrencyNotSetValue, Icon: 1, Color: "000000"},
		{AccountId: 2, Uid: 1, Name: "GBP", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1, Currency: "GBP", Icon: 1, Color: "000000", DisplayOrder: 1},
		{AccountId: 3, Uid: 1, Name: "Cash", Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP", Icon: 1, Color: "000000"},
		{AccountId: 4, Uid: 1, Name: "Card", Category: models.ACCOUNT_CATEGORY_CREDIT_CARD, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Currency: "GBP", Icon: 1, Color: "000000", Extend: &models.AccountExtend{CreditCardStatementDate: &statementDate}},
	}

	newName := "Wallet"
	plan, err := MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Name: &newName}, accounts[2], accounts)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(plan.toUpdateAccounts))
	assert.Equal(t, "Wallet", plan.toUpdateAccounts[0].Name)
	assert.Equal(t, "Cash", accounts[2].Name)

	sameName := "Cash"
	plan, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Name: &sameName}, accounts[2], accounts)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(plan.toUpdateAccounts))

	newCurrency := "USD"
	_, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Currency: &newCurrency}, accounts[2], accounts)
	toolErr := mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "currency", toolErr.Field)

	newCategory := "cash"
	_, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Category: &newCategory}, accounts[1], accounts)
	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "category", toolErr.Field)

	plan, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Category: &newCategory}, accounts[0], accounts)
	assert.Nil(t, err)
	assert.True(t, plan.categoryChanged)
	assert.Equal(t, 2, len(plan.toUpdateAccounts))
	assert.Equal(t, models.ACCOUNT_CATEGORY_CASH, plan.toUpdateAccounts[1].Category)
	assert.Equal(t, int64(2), plan.toUpdateAccounts[1].AccountId)

	_, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{AddSubAccounts: []*MCPCreateAccountSubAccountRequest{{Name: "USD", Currency: "USD"}}}, accounts[2], accounts)
	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "add_sub_accounts", toolErr.Field)

	plan, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{AddSubAccounts: []*MCPCreateAccountSubAccountRequest{{Name: "USD", Currency: "USD"}}}, accounts[0], accounts)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(plan.toUpdateAccounts))
	assert.Equal(t, 1, len(plan.toAddAccounts))
	assert.Equal(t, int32(2), plan.toAddAccounts[0].DisplayOrder)
	assert.Equal(t, models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, plan.toAddAccounts[0].Category)

	hidden := true
	plan, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{Hidden: &hidden}, accounts[3], accounts)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(plan.toUpdateAccounts))
	assert.Equal(t, 10, *plan.toUpdateAccounts[0].Extend.CreditCardStatementDate)

	newStatementDate := 5
	_, err = MCPUpdateAccountToolHandler.buildUpdatePlan(1, &MCPUpdateAccountRequest{CreditCardStatementDate: &newStatementDate}, accounts[2], accounts)
	toolErr = mcpAccountTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeNotSupported, toolErr.Code)
	assert.Equal(t, "credit_card_statement_date", toolErr.Field)
}
