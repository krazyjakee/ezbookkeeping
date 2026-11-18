package api

import (
	"math/big"
	"slices"
	"sort"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const pageCountForAccountStatement = 1000

// TransactionReconciliationStatementHandler returns transaction reconciliation statement list of current user
func (a *TransactionsApi) TransactionReconciliationStatementHandler(c *core.WebContext) (any, *errs.Error) {
	var reconciliationStatementRequest models.TransactionReconciliationStatementRequest
	err := c.ShouldBindQuery(&reconciliationStatementRequest)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionReconciliationStatementHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionReconciliationStatementHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	account, err := a.accounts.GetAccountByAccountId(c, uid, reconciliationStatementRequest.AccountId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] failed to get account \"id:%d\" for user \"uid:%d\", because %s", reconciliationStatementRequest.AccountId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if account.Type != models.ACCOUNT_TYPE_SINGLE_ACCOUNT {
		log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] account \"id:%d\" for user \"uid:%d\" is not a single account", reconciliationStatementRequest.AccountId, uid)
		return nil, errs.ErrAccountTypeInvalid
	}

	maxTransactionTime := int64(0)

	if reconciliationStatementRequest.EndTime > 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(reconciliationStatementRequest.EndTime)
	}

	minTransactionTime := int64(0)

	if reconciliationStatementRequest.StartTime > 0 {
		minTransactionTime = utils.GetMinTransactionTimeFromUnixTime(reconciliationStatementRequest.StartTime)
	}

	transactionsWithAccountBalance, totalInflows, totalOutflows, openingBalance, closingBalance, err := a.transactions.GetAllTransactionsInOneAccountWithAccountBalanceByMaxTime(c, uid, pageCountForAccountStatement, maxTransactionTime, minTransactionTime, reconciliationStatementRequest.AccountId, account.Category)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] failed to get transactions from \"%d\" to \"%d\" for user \"uid:%d\", because %s", reconciliationStatementRequest.StartTime, reconciliationStatementRequest.EndTime, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactions := make([]*models.Transaction, len(transactionsWithAccountBalance))
	transactionAccountBalanceMap := make(map[int64]*models.TransactionWithAccountBalance, len(transactionsWithAccountBalance))

	for i := 0; i < len(transactionsWithAccountBalance); i++ {
		transactionWithBalance := transactionsWithAccountBalance[i]
		transactions[i] = transactionWithBalance.Transaction
		transactionAccountBalanceMap[transactionWithBalance.TransactionId] = transactionWithBalance
		transactionAccountBalanceMap[transactionWithBalance.RelatedId] = transactionWithBalance
	}

	allAccountIds := make([]int64, 0, len(transactions)*2)

	for i := 0; i < len(transactions); i++ {
		allAccountIds = append(allAccountIds, transactions[i].AccountId)

		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			allAccountIds = append(allAccountIds, transactions[i].RelatedAccountId)
		}
	}

	allAccounts, err := a.accounts.GetAccountsByAccountIds(c, uid, utils.ToUniqueInt64Slice(allAccountIds))

	if err != nil {
		log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] failed to get essential data for assembling transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionResult, err := a.getTransactionResponseListResult(c, user, transactions, allAccounts, nil, nil, nil, nil, clientTimezone, false, true, true, true)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionReconciliationStatementHandler] failed to assemble transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	responseItems := make([]*models.TransactionReconciliationStatementResponseItem, len(transactionResult))

	for i := 0; i < len(transactionResult); i++ {
		transactionResult := transactionResult[i]
		accountOpeningBalance := big.NewInt(0)
		accountClosingBalance := big.NewInt(0)

		if transactionWithBalance, exists := transactionAccountBalanceMap[transactionResult.Id]; exists {
			accountOpeningBalance = transactionWithBalance.AccountOpeningBalance
			accountClosingBalance = transactionWithBalance.AccountClosingBalance
		} else {
			log.Warnf(c, "[transactions.TransactionReconciliationStatementHandler] missing account balance for transaction \"id:%d\" of user \"uid:%d\"", transactionResult.Id, uid)
		}

		responseItems[i] = &models.TransactionReconciliationStatementResponseItem{
			TransactionInfoResponse: transactionResult,
			AccountOpeningBalance:   accountOpeningBalance.String(),
			AccountClosingBalance:   accountClosingBalance.String(),
		}
	}

	reconciliationStatementResp := &models.TransactionReconciliationStatementResponse{
		Transactions:   responseItems,
		TotalInflows:   totalInflows.String(),
		TotalOutflows:  totalOutflows.String(),
		OpeningBalance: openingBalance.String(),
		ClosingBalance: closingBalance.String(),
	}

	return reconciliationStatementResp, nil
}

// TransactionUnreconciledTransactionCountsHandler returns unreconciled transaction counts of current user's accounts
func (a *TransactionsApi) TransactionUnreconciledTransactionCountsHandler(c *core.WebContext) (any, *errs.Error) {
	var unreconciledCountReq models.TransactionUnreconciledCountRequest
	err := c.ShouldBindQuery(&unreconciledCountReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionUnreconciledTransactionCountsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	selectedAccountIds := make(map[int64]bool)

	if unreconciledCountReq.AccountIds != "" {
		accountIds, err := utils.StringArrayToInt64Array(strings.Split(unreconciledCountReq.AccountIds, ","))

		if err != nil {
			return nil, errs.ErrAccountIdInvalid
		}

		for _, accountId := range accountIds {
			selectedAccountIds[accountId] = true
		}
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionUnreconciledTransactionCountsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	if !user.UseLastReconciledTime {
		return nil, errs.ErrLastReconciledTimeIsNotEnabled
	}

	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionUnreconciledTransactionCountsHandler] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountLastReconciledTimes := make(map[int64]int64, len(accounts))

	for _, account := range accounts {
		if account.Hidden || account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		if len(selectedAccountIds) > 0 && !selectedAccountIds[account.AccountId] {
			continue
		}

		if account.Extend != nil && account.Extend.LastReconciledTime != nil && *account.Extend.LastReconciledTime > 0 {
			accountLastReconciledTimes[account.AccountId] = *account.Extend.LastReconciledTime
		} else {
			accountLastReconciledTimes[account.AccountId] = 0
		}
	}

	counts, err := a.transactions.GetUnreconciledTransactionCounts(c, uid, accountLastReconciledTimes)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionUnreconciledTransactionCountsHandler] failed to get unreconciled transaction counts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return &models.TransactionUnreconciledCountResponse{
		Items: counts,
	}, nil
}

// TransactionStatisticsHandler returns transaction statistics of current user
func (a *TransactionsApi) TransactionStatisticsHandler(c *core.WebContext) (any, *errs.Error) {
	var statisticReq models.TransactionStatisticRequest
	err := c.ShouldBindQuery(&statisticReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	noTags := statisticReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(statisticReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionStatisticsHandler] parse transaction tag filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	uid := c.GetCurrentUid()
	totalAmounts, err := a.transactions.GetAccountsAndCategoriesTotalInflowAndOutflow(c, uid, statisticReq.StartTime, statisticReq.EndTime, tagFilters, noTags, statisticReq.Keyword, statisticReq.MatchMode, clientTimezone, statisticReq.UseTransactionTimezone)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionStatisticsHandler] failed to get accounts and categories total income and expense for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	statisticResp := &models.TransactionStatisticResponse{
		StartTime: statisticReq.StartTime,
		EndTime:   statisticReq.EndTime,
	}

	statisticResp.Items = make([]*models.TransactionStatisticResponseItem, len(totalAmounts))

	for i := 0; i < len(totalAmounts); i++ {
		totalAmountItem := totalAmounts[i]
		statisticResp.Items[i] = &models.TransactionStatisticResponseItem{
			CategoryId:  totalAmountItem.CategoryId,
			AccountId:   totalAmountItem.AccountId,
			TotalAmount: totalAmountItem.Amount.String(),
		}

		if totalAmountItem.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || totalAmountItem.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			statisticResp.Items[i].RelatedAccountId = totalAmountItem.RelatedAccountId
			statisticResp.Items[i].RelatedAccountType, _ = totalAmountItem.Type.ToTransactionRelatedAccountType()
		}
	}

	return statisticResp, nil
}

// TransactionStatisticsTrendsHandler returns transaction statistics trends of current user
func (a *TransactionsApi) TransactionStatisticsTrendsHandler(c *core.WebContext) (any, *errs.Error) {
	var statisticTrendsReq models.TransactionStatisticTrendsRequest
	err := c.ShouldBindQuery(&statisticTrendsReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsTrendsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsTrendsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	startYear, startMonth, endYear, endMonth, err := statisticTrendsReq.GetNumericYearMonthRange()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsTrendsHandler] cannot parse year month, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	noTags := statisticTrendsReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(statisticTrendsReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionStatisticsTrendsHandler] parse transaction tag filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	uid := c.GetCurrentUid()
	allMonthlyTotalAmounts, err := a.transactions.GetAccountsAndCategoriesMonthlyInflowAndOutflow(c, uid, startYear, startMonth, endYear, endMonth, tagFilters, noTags, statisticTrendsReq.Keyword, statisticTrendsReq.MatchMode, clientTimezone, statisticTrendsReq.UseTransactionTimezone)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionStatisticsTrendsHandler] failed to get accounts and categories total income and expense for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	statisticTrendsResp := make(models.TransactionStatisticTrendsResponseItemSlice, 0, len(allMonthlyTotalAmounts))

	for yearMonth, monthlyTotalAmounts := range allMonthlyTotalAmounts {
		monthlyStatisticResp := &models.TransactionStatisticTrendsResponseItem{
			Year:  yearMonth / 100,
			Month: yearMonth % 100,
			Items: make([]*models.TransactionStatisticResponseItem, len(monthlyTotalAmounts)),
		}

		for i := 0; i < len(monthlyTotalAmounts); i++ {
			totalAmountItem := monthlyTotalAmounts[i]
			monthlyStatisticResp.Items[i] = &models.TransactionStatisticResponseItem{
				CategoryId:  totalAmountItem.CategoryId,
				AccountId:   totalAmountItem.AccountId,
				TotalAmount: totalAmountItem.Amount.String(),
			}

			if totalAmountItem.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || totalAmountItem.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
				monthlyStatisticResp.Items[i].RelatedAccountId = totalAmountItem.RelatedAccountId
				monthlyStatisticResp.Items[i].RelatedAccountType, _ = totalAmountItem.Type.ToTransactionRelatedAccountType()
			}
		}

		statisticTrendsResp = append(statisticTrendsResp, monthlyStatisticResp)
	}

	sort.Sort(statisticTrendsResp)

	return statisticTrendsResp, nil
}

// TransactionStatisticsAssetTrendsHandler returns transaction statistics asset trends of current user
func (a *TransactionsApi) TransactionStatisticsAssetTrendsHandler(c *core.WebContext) (any, *errs.Error) {
	var statisticAssetTrendsReq models.TransactionStatisticAssetTrendsRequest
	err := c.ShouldBindQuery(&statisticAssetTrendsReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsAssetTrendsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionStatisticsAssetTrendsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()

	maxTransactionTime := int64(0)

	if statisticAssetTrendsReq.EndTime > 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(statisticAssetTrendsReq.EndTime)
	}

	minTransactionTime := int64(0)

	if statisticAssetTrendsReq.StartTime > 0 {
		minTransactionTime = utils.GetMinTransactionTimeFromUnixTime(statisticAssetTrendsReq.StartTime)
	}

	accountDailyBalances, err := a.transactions.GetAllAccountsDailyOpeningAndClosingBalance(c, uid, maxTransactionTime, minTransactionTime, clientTimezone)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionStatisticsAssetTrendsHandler] failed to get transactions from \"%d\" to \"%d\" for user \"uid:%d\", because %s", statisticAssetTrendsReq.StartTime, statisticAssetTrendsReq.EndTime, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	statisticAssetTrendsResp := make(models.TransactionStatisticAssetTrendsResponseItemSlice, 0)

	for yearMonthDay, dailyAccountBalances := range accountDailyBalances {
		dailyStatisticResp := &models.TransactionStatisticAssetTrendsResponseItem{
			Year:  yearMonthDay / 10000,
			Month: (yearMonthDay % 10000) / 100,
			Day:   yearMonthDay % 100,
			Items: make([]*models.TransactionStatisticAssetTrendsResponseDataItem, len(dailyAccountBalances)),
		}

		for i := 0; i < len(dailyAccountBalances); i++ {
			accountBalance := dailyAccountBalances[i]
			dailyStatisticResp.Items[i] = &models.TransactionStatisticAssetTrendsResponseDataItem{
				AccountId:             accountBalance.AccountId,
				AccountOpeningBalance: accountBalance.AccountOpeningBalance.String(),
				AccountClosingBalance: accountBalance.AccountClosingBalance.String(),
			}
		}

		statisticAssetTrendsResp = append(statisticAssetTrendsResp, dailyStatisticResp)
	}

	sort.Sort(statisticAssetTrendsResp)

	return statisticAssetTrendsResp, nil
}

// TransactionAmountsHandler returns transaction amounts of current user
func (a *TransactionsApi) TransactionAmountsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionAmountsReq models.TransactionAmountsRequest
	err := c.ShouldBindQuery(&transactionAmountsReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	requestItems, err := transactionAmountsReq.GetTransactionAmountsRequestItems()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] get request item failed, because %s", err.Error())
		return nil, errs.ErrQueryItemsInvalid
	}

	if len(requestItems) < 1 {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] parse request failed, because there are no valid items")
		return nil, errs.ErrQueryItemsEmpty
	}

	if len(requestItems) > 20 {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] parse request failed, because there are too many items")
		return nil, errs.ErrQueryItemsTooMuch
	}

	excludeAccountIds := make([]int64, 0)
	excludeCategoryIds := make([]int64, 0)

	if transactionAmountsReq.ExcludeAccountIds != "" {
		excludeAccountIds, err = utils.StringArrayToInt64Array(strings.Split(transactionAmountsReq.ExcludeAccountIds, ","))

		if err != nil {
			return nil, errs.ErrAccountIdInvalid
		}
	}

	if transactionAmountsReq.ExcludeCategoryIds != "" {
		excludeCategoryIds, err = utils.StringArrayToInt64Array(strings.Split(transactionAmountsReq.ExcludeCategoryIds, ","))

		if err != nil {
			return nil, errs.ErrTransactionCategoryIdInvalid
		}
	}

	tagFilters, err := models.ParseTransactionTagFilter(transactionAmountsReq.TagFilter)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] parse transaction tag filters error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionAmountsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()

	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)
	accountMap := a.accounts.GetAccountMapByList(accounts)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionAmountsHandler] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	amountsResp := orderedmap.New[string, *models.TransactionAmountsResponseItem]()

	for i := 0; i < len(requestItems); i++ {
		requestItem := requestItems[i]

		incomeAmounts, expenseAmounts, err := a.transactions.GetAccountsTotalIncomeAndExpense(c, uid, requestItem.StartTime, requestItem.EndTime, excludeAccountIds, excludeCategoryIds, tagFilters, clientTimezone, transactionAmountsReq.UseTransactionTimezone)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionAmountsHandler] failed to get transaction amounts item for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		amountsMap := make(map[string]*models.TransactionAmountsAndCurrency)

		for accountId, incomeAmount := range incomeAmounts {
			account, exists := accountMap[accountId]

			if !exists {
				log.Warnf(c, "[transactions.TransactionAmountsHandler] cannot find account for account \"id:%d\" of user \"uid:%d\"", accountId, uid)
				continue
			}

			totalAmounts, exists := amountsMap[account.Currency]

			if !exists {
				totalAmounts = &models.TransactionAmountsAndCurrency{
					Currency:      account.Currency,
					IncomeAmount:  big.NewInt(0),
					ExpenseAmount: big.NewInt(0),
				}
			}

			totalAmounts.IncomeAmount.Add(totalAmounts.IncomeAmount, incomeAmount)
			amountsMap[account.Currency] = totalAmounts
		}

		for accountId, expenseAmount := range expenseAmounts {
			account, exists := accountMap[accountId]

			if !exists {
				log.Warnf(c, "[transactions.TransactionAmountsHandler] cannot find account for account \"id:%d\" of user \"uid:%d\"", accountId, uid)
				continue
			}

			totalAmounts, exists := amountsMap[account.Currency]

			if !exists {
				totalAmounts = &models.TransactionAmountsAndCurrency{
					Currency:      account.Currency,
					IncomeAmount:  big.NewInt(0),
					ExpenseAmount: big.NewInt(0),
				}
			}

			totalAmounts.ExpenseAmount.Add(totalAmounts.ExpenseAmount, expenseAmount)
			amountsMap[account.Currency] = totalAmounts
		}

		allTotalAmounts := make(models.TransactionAmountsResponseItemAmountInfoSlice, 0)

		for _, totalAmounts := range amountsMap {
			allTotalAmounts = append(allTotalAmounts, &models.TransactionAmountsResponseItemAmountInfo{
				Currency:      totalAmounts.Currency,
				IncomeAmount:  totalAmounts.IncomeAmount.String(),
				ExpenseAmount: totalAmounts.ExpenseAmount.String(),
			})
		}

		sort.Sort(allTotalAmounts)

		amountsResp.Set(requestItem.Name, &models.TransactionAmountsResponseItem{
			StartTime: requestItem.StartTime,
			EndTime:   requestItem.EndTime,
			Amounts:   allTotalAmounts,
		})
	}

	return amountsResp, nil
}

// TransactionDailyAmountsHandler returns daily transaction amounts of current user
func (a *TransactionsApi) TransactionDailyAmountsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionAmountsReq models.TransactionDailyAmountsRequest
	err := c.ShouldBindQuery(&transactionAmountsReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionDailyAmountsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	if transactionAmountsReq.EndTime < transactionAmountsReq.StartTime {
		return nil, errs.ErrDateRangeInvalid
	}

	excludeAccountIds := make([]int64, 0)
	excludeCategoryIds := make([]int64, 0)

	if transactionAmountsReq.ExcludeAccountIds != "" {
		excludeAccountIds, err = utils.StringArrayToInt64Array(strings.Split(transactionAmountsReq.ExcludeAccountIds, ","))

		if err != nil {
			return nil, errs.ErrAccountIdInvalid
		}
	}

	if transactionAmountsReq.ExcludeCategoryIds != "" {
		excludeCategoryIds, err = utils.StringArrayToInt64Array(strings.Split(transactionAmountsReq.ExcludeCategoryIds, ","))

		if err != nil {
			return nil, errs.ErrTransactionCategoryIdInvalid
		}
	}

	tagFilters, err := models.ParseTransactionTagFilter(transactionAmountsReq.TagFilter)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionDailyAmountsHandler] parse transaction tag filters error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionDailyAmountsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)
	accountMap := a.accounts.GetAccountMapByList(accounts)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionDailyAmountsHandler] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	incomeAmounts, expenseAmounts, err := a.transactions.GetAccountsDailyIncomeAndExpense(c, uid, transactionAmountsReq.StartTime, transactionAmountsReq.EndTime, excludeAccountIds, excludeCategoryIds, tagFilters, clientTimezone, transactionAmountsReq.UseTransactionTimezone)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionDailyAmountsHandler] failed to get daily amounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allDates := make(map[int32]bool)

	for date := range incomeAmounts {
		allDates[date] = true
	}

	for date := range expenseAmounts {
		allDates[date] = true
	}

	dates := make([]int32, 0, len(allDates))

	for date := range allDates {
		dates = append(dates, date)
	}

	slices.Sort(dates)

	result := make([]*models.TransactionDailyAmountsResponseItem, 0, len(dates))

	for _, date := range dates {
		amountsByCurrency := make(map[string]*models.TransactionAmountsAndCurrency)

		for accountId, amount := range incomeAmounts[date] {
			account, exists := accountMap[accountId]

			if !exists {
				log.Warnf(c, "[transactions.TransactionDailyAmountsHandler] cannot find account for account \"id:%d\" of user \"uid:%d\"", accountId, uid)
				continue
			}

			totalAmounts, exists := amountsByCurrency[account.Currency]

			if !exists {
				totalAmounts = &models.TransactionAmountsAndCurrency{
					Currency:      account.Currency,
					IncomeAmount:  big.NewInt(0),
					ExpenseAmount: big.NewInt(0),
				}
			}

			totalAmounts.IncomeAmount.Add(totalAmounts.IncomeAmount, amount)
			amountsByCurrency[account.Currency] = totalAmounts
		}

		for accountId, amount := range expenseAmounts[date] {
			account, exists := accountMap[accountId]

			if !exists {
				log.Warnf(c, "[transactions.TransactionDailyAmountsHandler] cannot find account for account \"id:%d\" of user \"uid:%d\"", accountId, uid)
				continue
			}

			totalAmounts, exists := amountsByCurrency[account.Currency]

			if !exists {
				totalAmounts = &models.TransactionAmountsAndCurrency{
					Currency:      account.Currency,
					IncomeAmount:  big.NewInt(0),
					ExpenseAmount: big.NewInt(0),
				}
			}

			totalAmounts.ExpenseAmount.Add(totalAmounts.ExpenseAmount, amount)
			amountsByCurrency[account.Currency] = totalAmounts
		}

		dailyTotalAmounts := make(models.TransactionAmountsResponseItemAmountInfoSlice, 0, len(amountsByCurrency))

		for _, total := range amountsByCurrency {
			dailyTotalAmounts = append(dailyTotalAmounts, &models.TransactionAmountsResponseItemAmountInfo{
				Currency:      total.Currency,
				IncomeAmount:  total.IncomeAmount.String(),
				ExpenseAmount: total.ExpenseAmount.String(),
			})
		}

		sort.Sort(dailyTotalAmounts)

		result = append(result, &models.TransactionDailyAmountsResponseItem{
			Date:    utils.FormatNumericYearMonthDayToLongDate(date),
			Amounts: dailyTotalAmounts,
		})
	}

	return result, nil
}
