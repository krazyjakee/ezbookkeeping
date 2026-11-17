package mcp

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func getReportTestAccounts() map[int64]*models.Account {
	return map[int64]*models.Account{
		101: {AccountId: 101, Name: "Bank GBP", Currency: "GBP", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT},
		102: {AccountId: 102, Name: "Director Loan", Currency: "GBP", Category: models.ACCOUNT_CATEGORY_DEBT},
		103: {AccountId: 103, Name: "Bank USD", Currency: "USD", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT},
	}
}

func getReportTestCategories() map[int64]*models.TransactionCategory {
	return map[int64]*models.TransactionCategory{
		201: {CategoryId: 201, Name: "Business", ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		202: {CategoryId: 202, Name: "Software", ParentCategoryId: 201},
		203: {CategoryId: 203, Name: "Travel", ParentCategoryId: 201},
		204: {CategoryId: 204, Name: "Transfers", ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		205: {CategoryId: 205, Name: "Reimbursement", ParentCategoryId: 204},
		206: {CategoryId: 206, Name: "Sales", ParentCategoryId: models.LevelOneTransactionCategoryParentId},
		207: {CategoryId: 207, Name: "Services", ParentCategoryId: 206},
	}
}

func getReportTestTotalAmounts() []*models.TransactionTotalAmount {
	return []*models.TransactionTotalAmount{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 202, AccountId: 102, Amount: big.NewInt(12000)},
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 203, AccountId: 101, Amount: big.NewInt(5050)},
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 203, AccountId: 103, Amount: big.NewInt(1000)},
		{Type: models.TRANSACTION_DB_TYPE_INCOME, CategoryId: 207, AccountId: 101, Amount: big.NewInt(100000)},
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: 205, AccountId: 101, RelatedAccountId: 102, Amount: big.NewInt(12000)},
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_IN, CategoryId: 205, AccountId: 102, RelatedAccountId: 101, Amount: big.NewInt(12000)},
	}
}

func aggregateReportTestTotals(groupBy string, accountFilter map[int64]bool, categoryFilter map[int64]bool) []*MCPTransactionTotalsItem {
	accounts := getReportTestAccounts()
	categories := getReportTestCategories()
	aggregator := newMCPTransactionTotalsAggregator()

	for _, row := range getReportTestTotalAmounts() {
		if !mcpTotalsRowMatches(row, accountFilter, categoryFilter, categories) {
			continue
		}

		groupId, groupName := mcpTotalsGetGroup(groupBy, row, accounts, categories)
		aggregator.Add(groupId, groupName, mcpTotalsGetCurrency(row, accounts), row.Type, row.Amount)
	}

	return aggregator.Items()
}

func TestMCPTransactionTotals_GroupByCategory(t *testing.T) {
	items := aggregateReportTestTotals(mcpTotalsGroupByCategory, nil, nil)
	assert.Equal(t, 5, len(items))

	assert.Equal(t, "Reimbursement", items[0].Name)
	assert.Equal(t, "205", items[0].Id)
	assert.Equal(t, "GBP", items[0].Currency)
	assert.Equal(t, "120.00", items[0].TransferIn)
	assert.Equal(t, "120.00", items[0].TransferOut)
	assert.Equal(t, "0.00", items[0].Net)

	assert.Equal(t, "Services", items[1].Name)
	assert.Equal(t, "1000.00", items[1].Income)
	assert.Equal(t, "1000.00", items[1].Net)

	assert.Equal(t, "Software", items[2].Name)
	assert.Equal(t, "120.00", items[2].Expense)
	assert.Equal(t, "-120.00", items[2].Net)

	assert.Equal(t, "Travel", items[3].Name)
	assert.Equal(t, "GBP", items[3].Currency)
	assert.Equal(t, "50.50", items[3].Expense)
	assert.Equal(t, "Travel", items[4].Name)
	assert.Equal(t, "USD", items[4].Currency)
	assert.Equal(t, "10.00", items[4].Expense)
	assert.Equal(t, "0.00", items[4].Income)
}

func TestMCPTransactionTotals_GroupByPrimaryCategoryAccountAndCurrency(t *testing.T) {
	items := aggregateReportTestTotals(mcpTotalsGroupByPrimaryCategory, nil, nil)
	assert.Equal(t, 4, len(items))
	assert.Equal(t, "Business", items[0].Name)
	assert.Equal(t, "201", items[0].Id)
	assert.Equal(t, "GBP", items[0].Currency)
	assert.Equal(t, "170.50", items[0].Expense)
	assert.Equal(t, "Business", items[1].Name)
	assert.Equal(t, "USD", items[1].Currency)

	items = aggregateReportTestTotals(mcpTotalsGroupByAccount, nil, nil)
	assert.Equal(t, 3, len(items))
	assert.Equal(t, "Bank GBP", items[0].Name)
	assert.Equal(t, "101", items[0].Id)
	assert.Equal(t, "1000.00", items[0].Income)
	assert.Equal(t, "50.50", items[0].Expense)
	assert.Equal(t, "120.00", items[0].TransferOut)
	assert.Equal(t, "829.50", items[0].Net)
	assert.Equal(t, "Director Loan", items[2].Name)
	assert.Equal(t, "120.00", items[2].Expense)
	assert.Equal(t, "120.00", items[2].TransferIn)
	assert.Equal(t, "0.00", items[2].Net)

	items = aggregateReportTestTotals(mcpTotalsGroupByCurrency, nil, nil)
	assert.Equal(t, 2, len(items))
	assert.Equal(t, "", items[0].Id)
	assert.Equal(t, "GBP", items[0].Name)
	assert.Equal(t, "GBP", items[0].Currency)
	assert.Equal(t, "1000.00", items[0].Income)
	assert.Equal(t, "170.50", items[0].Expense)
	assert.Equal(t, "USD", items[1].Currency)
	assert.Equal(t, "-10.00", items[1].Net)
}

func TestMCPTransactionTotals_Filters(t *testing.T) {
	items := aggregateReportTestTotals(mcpTotalsGroupByCategory, map[int64]bool{102: true}, nil)
	assert.Equal(t, 2, len(items))
	assert.Equal(t, "Reimbursement", items[0].Name)
	assert.Equal(t, "0.00", items[0].TransferOut)
	assert.Equal(t, "120.00", items[0].TransferIn)
	assert.Equal(t, "Software", items[1].Name)

	// primary category id matches all its secondary categories
	items = aggregateReportTestTotals(mcpTotalsGroupByCategory, nil, map[int64]bool{201: true})
	assert.Equal(t, 3, len(items))

	for _, item := range items {
		assert.NotEqual(t, "0.00", item.Expense)
		assert.Equal(t, "0.00", item.Income)
	}

	items = aggregateReportTestTotals(mcpTotalsGroupByCategory, map[int64]bool{101: true}, map[int64]bool{203: true})
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "Travel", items[0].Name)
	assert.Equal(t, "GBP", items[0].Currency)
}

func TestMCPTransactionTotalsAggregator_IgnoreNilAmount(t *testing.T) {
	aggregator := newMCPTransactionTotalsAggregator()
	aggregator.Add("1", "a", "GBP", models.TRANSACTION_DB_TYPE_EXPENSE, nil)
	assert.Equal(t, 0, len(aggregator.Items()))

	aggregator.Add("1", "a", "GBP", models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, big.NewInt(100))
	items := aggregator.Items()
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "0.00", items[0].Net)
}

func TestMCPFormatAccountBalances(t *testing.T) {
	display, signed, outstanding := mcpFormatAccountBalances(models.ACCOUNT_CATEGORY_DEBT, big.NewInt(-12000))
	assert.Equal(t, "120.00", display)
	assert.Equal(t, "-120.00", signed)
	assert.Equal(t, "120.00", outstanding)

	display, signed, outstanding = mcpFormatAccountBalances(models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, big.NewInt(50050))
	assert.Equal(t, "500.50", display)
	assert.Equal(t, "500.50", signed)
	assert.Equal(t, "", outstanding)

	display, signed, _ = mcpFormatAccountBalances(models.ACCOUNT_CATEGORY_CASH, nil)
	assert.Equal(t, "0.00", display)
	assert.Equal(t, "0.00", signed)
}

func TestMCPReportGetLatestClosingBalances(t *testing.T) {
	dailyBalances := map[int32][]*models.TransactionWithAccountBalance{
		20240102: {
			{Transaction: &models.Transaction{AccountId: 101}, AccountOpeningBalance: big.NewInt(100), AccountClosingBalance: big.NewInt(300)},
		},
		20240101: {
			{Transaction: &models.Transaction{AccountId: 101}, AccountOpeningBalance: big.NewInt(0), AccountClosingBalance: big.NewInt(100)},
			{Transaction: &models.Transaction{AccountId: 102}, AccountOpeningBalance: big.NewInt(0), AccountClosingBalance: big.NewInt(-12000)},
		},
	}

	balances := mcpReportGetLatestClosingBalances(dailyBalances)
	assert.Equal(t, 2, len(balances))
	assert.Equal(t, int64(300), balances[101].Int64())
	assert.Equal(t, int64(-12000), balances[102].Int64())
	assert.Nil(t, balances[103])

	assert.Equal(t, 0, len(mcpReportGetLatestClosingBalances(nil)))
}

func TestMCPReportParseTimeRange(t *testing.T) {
	startUnixTime, endUnixTime, location, err := mcpReportParseTimeRange("", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), startUnixTime)
	assert.Equal(t, int64(0), endUnixTime)
	assert.NotNil(t, location)

	startUnixTime, endUnixTime, location, err = mcpReportParseTimeRange("2024-01-01T00:00:00+01:00", "2024-01-31T23:59:59+01:00")
	assert.Nil(t, err)
	assert.Equal(t, int64(1704063600), startUnixTime)
	assert.Equal(t, int64(1706741999), endUnixTime)
	_, offset := time.Unix(startUnixTime, 0).In(location).Zone()
	assert.Equal(t, 3600, offset)

	_, _, _, err = mcpReportParseTimeRange("2024-02-01T00:00:00Z", "2024-01-01T00:00:00Z")
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, "end_time", toolErr.Field)

	_, _, _, err = mcpReportParseTimeRange("yesterday", "")
	toolErr = mcpTestGetToolError(t, err)
	assert.Equal(t, "start_time", toolErr.Field)
}

func TestMCPReconciliationGetAccountAmount(t *testing.T) {
	assert.Equal(t, int64(-12000), mcpReconciliationGetAccountAmount(&models.Transaction{Type: models.TRANSACTION_DB_TYPE_EXPENSE, Amount: 12000}).Int64())
	assert.Equal(t, int64(-12000), mcpReconciliationGetAccountAmount(&models.Transaction{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, Amount: 12000}).Int64())
	assert.Equal(t, int64(12000), mcpReconciliationGetAccountAmount(&models.Transaction{Type: models.TRANSACTION_DB_TYPE_TRANSFER_IN, Amount: 12000}).Int64())
	assert.Equal(t, int64(500), mcpReconciliationGetAccountAmount(&models.Transaction{Type: models.TRANSACTION_DB_TYPE_INCOME, Amount: 500}).Int64())
	assert.Equal(t, int64(-300), mcpReconciliationGetAccountAmount(&models.Transaction{Type: models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, Amount: 100, RelatedAccountAmount: -300}).Int64())
}

func TestMCPReportGetAccountFilter(t *testing.T) {
	accounts := []*models.Account{
		{AccountId: 1, Name: "Parent", Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS},
		{AccountId: 2, Name: "Child 1", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1},
		{AccountId: 3, Name: "Child 2", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, ParentAccountId: 1},
		{AccountId: 4, Name: "Other", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT},
	}

	filter, err := mcpReportGetAccountFilter(accounts, "account_ids", nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(filter))

	filter, err = mcpReportGetAccountFilter(accounts, "account_ids", []string{"1"})
	assert.Nil(t, err)
	assert.True(t, filter[2])
	assert.True(t, filter[3])
	assert.False(t, filter[4])

	_, err = mcpReportGetAccountFilter(accounts, "account_ids", []string{"99"})
	assert.Equal(t, MCPToolErrorCodeNotFound, mcpTestGetToolError(t, err).Code)

	_, err = mcpReportGetAccountFilter(accounts, "account_ids", []string{"abc"})
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTestGetToolError(t, err).Code)
}

func getReportTestTags() map[int64]*models.TransactionTag {
	return map[int64]*models.TransactionTag{
		301: {TagId: 301, Name: "Client A"},
		302: {TagId: 302, Name: "Client B"},
		303: {TagId: 303, Name: "Unused"},
	}
}

func getReportTestTaggedTransactions() ([]*models.Transaction, map[int64][]int64) {
	transactions := []*models.Transaction{
		{TransactionId: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 202, AccountId: 101, Amount: 1000},
		{TransactionId: 2, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 203, AccountId: 103, Amount: 2000},
		{TransactionId: 3, Type: models.TRANSACTION_DB_TYPE_INCOME, CategoryId: 207, AccountId: 101, Amount: 50000},
		{TransactionId: 4, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: 205, AccountId: 101, RelatedId: 5, RelatedAccountId: 102, Amount: 3000},
		{TransactionId: 5, Type: models.TRANSACTION_DB_TYPE_TRANSFER_IN, CategoryId: 205, AccountId: 102, RelatedId: 4, RelatedAccountId: 101, Amount: 3000},
		{TransactionId: 6, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 202, AccountId: 101, Amount: 700},
	}
	transactionTagIds := map[int64][]int64{
		1: {301, 302},
		2: {301},
		3: {302, 302},
		4: {301},
	}

	return transactions, transactionTagIds
}

func aggregateReportTestTotalsByTag(groupTagIds []int64, includeNoTags bool, accountFilter map[int64]bool, categoryFilter map[int64]bool) []*MCPTransactionTotalsItem {
	transactions, transactionTagIds := getReportTestTaggedTransactions()
	aggregator := newMCPTransactionTotalsAggregator()
	mcpTotalsAggregateByTag(aggregator, transactions, transactionTagIds, groupTagIds, getReportTestTags(), includeNoTags, accountFilter, categoryFilter, getReportTestCategories(), getReportTestAccounts())
	return aggregator.Items()
}

func TestMCPTotalsAggregateByTag_AllTags(t *testing.T) {
	items := aggregateReportTestTotalsByTag([]int64{301, 302, 303}, true, nil, nil)

	// tag 303 has no transactions so it has no group, and Client B only has GBP transactions
	assert.Equal(t, 4, len(items))

	// "(no tags)" sorts first by name: transaction 6
	assert.Equal(t, "", items[0].Id)
	assert.Equal(t, mcpTotalsNoTagsGroupName, items[0].Name)
	assert.Equal(t, "GBP", items[0].Currency)
	assert.Equal(t, "7.00", items[0].Expense)
	assert.Equal(t, "-7.00", items[0].Net)

	// Client A in GBP: transaction 1 and both sides of the transfer (the transfer in side gets the tags of the transfer out side)
	assert.Equal(t, "301", items[1].Id)
	assert.Equal(t, "Client A", items[1].Name)
	assert.Equal(t, "GBP", items[1].Currency)
	assert.Equal(t, "10.00", items[1].Expense)
	assert.Equal(t, "30.00", items[1].TransferIn)
	assert.Equal(t, "30.00", items[1].TransferOut)
	assert.Equal(t, "-10.00", items[1].Net)

	// Client A in USD: transaction 2
	assert.Equal(t, "301", items[2].Id)
	assert.Equal(t, "USD", items[2].Currency)
	assert.Equal(t, "20.00", items[2].Expense)

	// Client B: transaction 1 and transaction 3 (duplicated tag ids are counted once)
	assert.Equal(t, "302", items[3].Id)
	assert.Equal(t, "Client B", items[3].Name)
	assert.Equal(t, "GBP", items[3].Currency)
	assert.Equal(t, "500.00", items[3].Income)
	assert.Equal(t, "10.00", items[3].Expense)
	assert.Equal(t, "490.00", items[3].Net)
}

func TestMCPTotalsAggregateByTag_SpecifiedTagsWithoutNoTagsGroup(t *testing.T) {
	items := aggregateReportTestTotalsByTag([]int64{302}, false, nil, nil)

	assert.Equal(t, 1, len(items))
	assert.Equal(t, "302", items[0].Id)
	assert.Equal(t, "500.00", items[0].Income)
	assert.Equal(t, "10.00", items[0].Expense)
}

func TestMCPTotalsAggregateByTag_AccountAndCategoryFilters(t *testing.T) {
	items := aggregateReportTestTotalsByTag([]int64{301, 302}, true, map[int64]bool{101: true}, nil)

	assert.Equal(t, 3, len(items))
	assert.Equal(t, mcpTotalsNoTagsGroupName, items[0].Name)
	assert.Equal(t, "301", items[1].Id)
	assert.Equal(t, "GBP", items[1].Currency)
	assert.Equal(t, "10.00", items[1].Expense)
	assert.Equal(t, "0.00", items[1].TransferIn)
	assert.Equal(t, "30.00", items[1].TransferOut)
	assert.Equal(t, "302", items[2].Id)

	items = aggregateReportTestTotalsByTag([]int64{301, 302}, true, nil, map[int64]bool{201: true})

	assert.Equal(t, 4, len(items))
	assert.Equal(t, mcpTotalsNoTagsGroupName, items[0].Name)
	assert.Equal(t, "7.00", items[0].Expense)
	assert.Equal(t, "301", items[1].Id)
	assert.Equal(t, "10.00", items[1].Expense)
	assert.Equal(t, "0.00", items[1].TransferOut)
	assert.Equal(t, "301", items[2].Id)
	assert.Equal(t, "USD", items[2].Currency)
	assert.Equal(t, "302", items[3].Id)
	assert.Equal(t, "0.00", items[3].Income)
}

func TestMCPTotalsGetTransactionTagIds(t *testing.T) {
	_, transactionTagIds := getReportTestTaggedTransactions()

	assert.Equal(t, []int64{301}, mcpTotalsGetTransactionTagIds(&models.Transaction{TransactionId: 5, RelatedId: 4}, transactionTagIds))
	assert.Equal(t, []int64{302}, mcpTotalsGetTransactionTagIds(&models.Transaction{TransactionId: 3}, transactionTagIds))
	assert.Equal(t, 0, len(mcpTotalsGetTransactionTagIds(&models.Transaction{TransactionId: 6}, transactionTagIds)))
	assert.Equal(t, []int64{301, 302}, transactionTagIds[1])
}

func TestMCPTotalsTransactionTimeRange(t *testing.T) {
	location := time.FixedZone("Test", 8*60*60)
	startUnixTime := time.Date(2024, 1, 1, 0, 0, 0, 0, location).Unix()
	endUnixTime := time.Date(2024, 1, 31, 23, 59, 59, 0, location).Unix()
	timeRange := mcpTotalsGetTransactionTimeRange(startUnixTime, endUnixTime, location)

	newTransaction := func(transactionType models.TransactionDbType, localTime time.Time) *models.Transaction {
		return &models.Transaction{Type: transactionType, TransactionTime: localTime.Unix() * 1000}
	}

	assert.True(t, timeRange.minTransactionTime > 0)
	assert.True(t, timeRange.maxTransactionTime > timeRange.minTransactionTime)
	assert.True(t, timeRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_EXPENSE, time.Date(2024, 1, 1, 0, 0, 0, 0, location))))
	assert.True(t, timeRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_TRANSFER_IN, time.Date(2024, 1, 31, 23, 59, 59, 0, location))))
	assert.False(t, timeRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_INCOME, time.Date(2023, 12, 31, 23, 59, 59, 0, location))))
	assert.False(t, timeRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_INCOME, time.Date(2024, 2, 1, 0, 0, 0, 0, location))))
	assert.False(t, timeRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, time.Date(2024, 1, 15, 0, 0, 0, 0, location))))

	unboundedRange := mcpTotalsGetTransactionTimeRange(0, 0, location)
	assert.Equal(t, int64(0), unboundedRange.minTransactionTime)
	assert.Equal(t, int64(0), unboundedRange.maxTransactionTime)
	assert.True(t, unboundedRange.contains(newTransaction(models.TRANSACTION_DB_TYPE_TRANSFER_OUT, time.Date(2030, 1, 1, 0, 0, 0, 0, location))))
}
