package services

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"xorm.io/builder"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// GetAccountsTotalIncomeAndExpense returns the every accounts total income and expense amount by specific date range
func (s *TransactionService) GetAccountsTotalIncomeAndExpense(c core.Context, uid int64, startUnixTime int64, endUnixTime int64, excludeAccountIds []int64, excludeCategoryIds []int64, tagFilters []*models.TransactionTagFilter, clientTimezone *time.Location, useTransactionTimezone bool) (map[int64]*big.Int, map[int64]*big.Int, error) {
	if uid <= 0 {
		return nil, nil, errs.ErrUserIdInvalid
	}

	startLocalDateTime := utils.FormatUnixTimeToNumericLocalDateTime(startUnixTime, clientTimezone)
	endLocalDateTime := utils.FormatUnixTimeToNumericLocalDateTime(endUnixTime, clientTimezone)

	allTransactions, err := s.getAllTransactionsInSpecifiedDateRange(c, uid, startUnixTime, endUnixTime, excludeAccountIds, excludeCategoryIds, tagFilters, clientTimezone)

	if err != nil {
		return nil, nil, err
	}

	incomeAmounts := make(map[int64]*big.Int)
	expenseAmounts := make(map[int64]*big.Int)

	for i := 0; i < len(allTransactions); i++ {
		transaction := allTransactions[i]
		timeZone := clientTimezone

		if useTransactionTimezone {
			timeZone = time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
		}

		localDateTime := utils.FormatUnixTimeToNumericLocalDateTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime), timeZone)

		if localDateTime < startLocalDateTime || localDateTime > endLocalDateTime {
			continue
		}

		var amountsMap map[int64]*big.Int

		if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			amountsMap = incomeAmounts
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			amountsMap = expenseAmounts
		}

		totalAmounts, exists := amountsMap[transaction.AccountId]

		if !exists {
			totalAmounts = big.NewInt(0)
		}

		totalAmounts.Add(totalAmounts, big.NewInt(transaction.Amount))
		amountsMap[transaction.AccountId] = totalAmounts
	}

	return incomeAmounts, expenseAmounts, nil
}

// GetAccountsDailyIncomeAndExpense returns daily income and expense amounts grouped by account
func (s *TransactionService) GetAccountsDailyIncomeAndExpense(c core.Context, uid int64, startUnixTime int64, endUnixTime int64, excludeAccountIds []int64, excludeCategoryIds []int64, tagFilters []*models.TransactionTagFilter, clientTimezone *time.Location, useTransactionTimezone bool) (map[int32]map[int64]*big.Int, map[int32]map[int64]*big.Int, error) {
	if uid <= 0 {
		return nil, nil, errs.ErrUserIdInvalid
	}

	startLocalDateTime := utils.FormatUnixTimeToNumericLocalDateTime(startUnixTime, clientTimezone)
	endLocalDateTime := utils.FormatUnixTimeToNumericLocalDateTime(endUnixTime, clientTimezone)

	allTransactions, err := s.getAllTransactionsInSpecifiedDateRange(c, uid, startUnixTime, endUnixTime, excludeAccountIds, excludeCategoryIds, tagFilters, clientTimezone)

	if err != nil {
		return nil, nil, err
	}

	incomeAmounts := make(map[int32]map[int64]*big.Int)
	expenseAmounts := make(map[int32]map[int64]*big.Int)

	for i := 0; i < len(allTransactions); i++ {
		transaction := allTransactions[i]
		timeZone := clientTimezone

		if useTransactionTimezone {
			timeZone = time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
		}

		unixTime := utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime)
		localDateTime := utils.FormatUnixTimeToNumericLocalDateTime(unixTime, timeZone)

		if localDateTime < startLocalDateTime || localDateTime > endLocalDateTime {
			continue
		}

		var allAmounts map[int32]map[int64]*big.Int

		if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			allAmounts = incomeAmounts
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			allAmounts = expenseAmounts
		}

		yearMonthDay := utils.FormatUnixTimeToNumericYearMonthDay(unixTime, timeZone)
		dailyAmounts, exists := allAmounts[yearMonthDay]

		if !exists {
			dailyAmounts = make(map[int64]*big.Int)
		}

		amount, exists := dailyAmounts[transaction.AccountId]

		if !exists {
			amount = big.NewInt(0)
		}

		amount.Add(amount, big.NewInt(transaction.Amount))
		dailyAmounts[transaction.AccountId] = amount
		allAmounts[yearMonthDay] = dailyAmounts
	}

	return incomeAmounts, expenseAmounts, nil
}

// GetAccountsAndCategoriesTotalInflowAndOutflow returns the every accounts and categories total inflows and outflows amount by specific date range
func (s *TransactionService) GetAccountsAndCategoriesTotalInflowAndOutflow(c core.Context, uid int64, startUnixTime int64, endUnixTime int64, tagFilters []*models.TransactionTagFilter, noTags bool, keyword string, matchMode core.MatchMode, clientTimezone *time.Location, useTransactionTimezone bool) ([]*models.TransactionTotalAmount, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var startLocalDateTime, endLocalDateTime, startTransactionTime, endTransactionTime int64

	if startUnixTime > 0 {
		startLocalDateTime = utils.FormatUnixTimeToNumericLocalDateTime(startUnixTime, clientTimezone)
		startUnixTime = utils.GetMinUnixTimeWithSameLocalDateTime(startUnixTime, utils.GetTimezoneOffsetMinutes(startUnixTime, clientTimezone))
		startTransactionTime = utils.GetMinTransactionTimeFromUnixTime(startUnixTime)
	}

	if endUnixTime > 0 {
		endLocalDateTime = utils.FormatUnixTimeToNumericLocalDateTime(endUnixTime, clientTimezone)
		endUnixTime = utils.GetMaxUnixTimeWithSameLocalDateTime(endUnixTime, utils.GetTimezoneOffsetMinutes(endUnixTime, clientTimezone))
		endTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(endUnixTime)
	}

	allTransactions, err := s.getAllInflowAndOutflowTransactions(c, uid, startTransactionTime, endTransactionTime, tagFilters, noTags, keyword, matchMode)

	if err != nil {
		return nil, err
	}

	transactionTotalAmountsMap := make(map[string]*models.TransactionTotalAmount)

	for i := 0; i < len(allTransactions); i++ {
		transaction := allTransactions[i]
		timeZone := clientTimezone

		if useTransactionTimezone {
			timeZone = time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
		}

		localDateTime := utils.FormatUnixTimeToNumericLocalDateTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime), timeZone)

		if (startLocalDateTime > 0 && localDateTime < startLocalDateTime) || (endLocalDateTime > 0 && localDateTime > endLocalDateTime) {
			continue
		}

		groupKey := fmt.Sprintf("%d_%d", transaction.CategoryId, transaction.AccountId)

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			groupKey = fmt.Sprintf("%d_%d_%d_%d", transaction.CategoryId, transaction.AccountId, transaction.RelatedAccountId, transaction.Type)
		}

		totalAmounts, exists := transactionTotalAmountsMap[groupKey]

		if !exists {
			totalAmounts = &models.TransactionTotalAmount{
				Type:             transaction.Type,
				CategoryId:       transaction.CategoryId,
				AccountId:        transaction.AccountId,
				RelatedAccountId: transaction.RelatedAccountId,
				Amount:           big.NewInt(0),
			}

			transactionTotalAmountsMap[groupKey] = totalAmounts
		}

		totalAmounts.Amount.Add(totalAmounts.Amount, big.NewInt(transaction.Amount))
	}

	transactionTotalAmounts := make([]*models.TransactionTotalAmount, 0, len(transactionTotalAmountsMap))

	for _, totalAmounts := range transactionTotalAmountsMap {
		transactionTotalAmounts = append(transactionTotalAmounts, totalAmounts)
	}

	return transactionTotalAmounts, nil
}

// GetUnreconciledTransactionCounts returns transaction counts after each account's last reconciled time
func (s *TransactionService) GetUnreconciledTransactionCounts(c core.Context, uid int64, accountLastReconciledTimes map[int64]int64) ([]*models.TransactionUnreconciledCountItem, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	accountConditions := make([]builder.Cond, 0, len(accountLastReconciledTimes))

	for accountId, lastReconciledTime := range accountLastReconciledTimes {
		accountConditions = append(accountConditions, builder.And(
			builder.Eq{"account_id": accountId},
			builder.Gt{"transaction_time": utils.GetMaxTransactionTimeFromUnixTime(lastReconciledTime)},
		))
	}

	counts := make([]*models.TransactionUnreconciledCountItem, 0)

	if len(accountConditions) < 1 {
		return counts, nil
	}

	err := s.UserDataDB(uid).NewSession(c).Table(&models.Transaction{}).Select("account_id, SUM(1) AS count").Where(builder.And(builder.Eq{"uid": uid, "deleted": false}, builder.Or(accountConditions...))).GroupBy("account_id").Having("SUM(1) > 0").OrderBy("account_id asc").Find(&counts)

	return counts, err
}

// GetAccountsAndCategoriesMonthlyInflowAndOutflow returns the every accounts monthly inflows and outflows amount by specific date range
func (s *TransactionService) GetAccountsAndCategoriesMonthlyInflowAndOutflow(c core.Context, uid int64, startYear int32, startMonth int32, endYear int32, endMonth int32, tagFilters []*models.TransactionTagFilter, noTags bool, keyword string, matchMode core.MatchMode, clientTimezone *time.Location, useTransactionTimezone bool) (map[int32][]*models.TransactionTotalAmount, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var startTransactionTime, endTransactionTime int64
	var err error

	if startYear > 0 && startMonth > 0 {
		startTransactionTime, _, err = utils.GetTransactionTimeRangeByYearMonth(startYear, startMonth)

		if err != nil {
			return nil, errs.ErrSystemError
		}
	}

	if endYear > 0 && endMonth > 0 {
		_, endTransactionTime, err = utils.GetTransactionTimeRangeByYearMonth(endYear, endMonth)

		if err != nil {
			return nil, errs.ErrSystemError
		}
	}

	allTransactions, err := s.getAllInflowAndOutflowTransactions(c, uid, startTransactionTime, endTransactionTime, tagFilters, noTags, keyword, matchMode)

	if err != nil {
		return nil, err
	}

	startYearMonth := startYear*100 + startMonth
	endYearMonth := endYear*100 + endMonth
	transactionsMonthlyAmountsMap := make(map[string]*models.TransactionTotalAmount)
	transactionsMonthlyAmounts := make(map[int32][]*models.TransactionTotalAmount)

	for i := 0; i < len(allTransactions); i++ {
		transaction := allTransactions[i]
		timeZone := clientTimezone

		if useTransactionTimezone {
			timeZone = time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
		}

		yearMonth := utils.FormatUnixTimeToNumericYearMonth(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime), timeZone)

		if (startYearMonth > 0 && yearMonth < startYearMonth) || (endYearMonth > 0 && yearMonth > endYearMonth) {
			continue
		}

		groupKey := fmt.Sprintf("%d_%d_%d", yearMonth, transaction.CategoryId, transaction.AccountId)

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			groupKey = fmt.Sprintf("%d_%d_%d_%d_%d", yearMonth, transaction.CategoryId, transaction.AccountId, transaction.RelatedAccountId, transaction.Type)
		}

		transactionAmounts, exists := transactionsMonthlyAmountsMap[groupKey]

		if !exists {
			transactionAmounts = &models.TransactionTotalAmount{
				Type:             transaction.Type,
				CategoryId:       transaction.CategoryId,
				AccountId:        transaction.AccountId,
				RelatedAccountId: transaction.RelatedAccountId,
				Amount:           big.NewInt(0),
			}

			transactionsMonthlyAmountsMap[groupKey] = transactionAmounts
		}

		transactionAmounts.Amount.Add(transactionAmounts.Amount, big.NewInt(transaction.Amount))
	}

	for groupKey, transaction := range transactionsMonthlyAmountsMap {
		groupKeyParts := strings.Split(groupKey, "_")
		yearMonth, _ := utils.StringToInt32(groupKeyParts[0])
		monthlyAmounts, exists := transactionsMonthlyAmounts[yearMonth]

		if !exists {
			monthlyAmounts = make([]*models.TransactionTotalAmount, 0)
		}

		monthlyAmounts = append(monthlyAmounts, transaction)
		transactionsMonthlyAmounts[yearMonth] = monthlyAmounts
	}

	return transactionsMonthlyAmounts, nil
}

func (s *TransactionService) getAllTransactionsInSpecifiedDateRange(c core.Context, uid int64, startUnixTime int64, endUnixTime int64, excludeAccountIds []int64, excludeCategoryIds []int64, tagFilters []*models.TransactionTagFilter, clientTimezone *time.Location) ([]*models.Transaction, error) {
	startUnixTime = utils.GetMinUnixTimeWithSameLocalDateTime(startUnixTime, utils.GetTimezoneOffsetMinutes(startUnixTime, clientTimezone))
	endUnixTime = utils.GetMaxUnixTimeWithSameLocalDateTime(endUnixTime, utils.GetTimezoneOffsetMinutes(endUnixTime, clientTimezone))

	startTransactionTime := utils.GetMinTransactionTimeFromUnixTime(startUnixTime)
	endTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(endUnixTime)

	condition := "uid=? AND deleted=? AND (type=? OR type=?)"
	conditionParams := make([]any, 0, 4+len(excludeAccountIds)+len(excludeCategoryIds))
	conditionParams = append(conditionParams, uid)
	conditionParams = append(conditionParams, false)
	conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_INCOME)
	conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_EXPENSE)

	if len(excludeAccountIds) > 0 {
		var accountIdsCondition strings.Builder
		accountIdConditionParams := make([]any, 0, len(excludeAccountIds))

		for i := 0; i < len(excludeAccountIds); i++ {
			if i > 0 {
				accountIdsCondition.WriteString(",")
			}

			accountIdsCondition.WriteString("?")
			accountIdConditionParams = append(accountIdConditionParams, excludeAccountIds[i])
		}

		condition = condition + " AND account_id NOT IN (" + accountIdsCondition.String() + ")"
		conditionParams = append(conditionParams, accountIdConditionParams...)
	}

	if len(excludeCategoryIds) > 0 {
		var categoryIdsCondition strings.Builder
		categoryIdConditionParams := make([]any, 0, len(excludeCategoryIds))

		for i := 0; i < len(excludeCategoryIds); i++ {
			if i > 0 {
				categoryIdsCondition.WriteString(",")
			}

			categoryIdsCondition.WriteString("?")
			categoryIdConditionParams = append(categoryIdConditionParams, excludeCategoryIds[i])
		}

		condition = condition + " AND category_id NOT IN (" + categoryIdsCondition.String() + ")"
		conditionParams = append(conditionParams, categoryIdConditionParams...)
	}

	condition = condition + " AND transaction_time>=? AND transaction_time<=?"

	minTransactionTime := startTransactionTime
	maxTransactionTime := endTransactionTime
	var allTransactions []*models.Transaction

	for maxTransactionTime > 0 {
		var transactions []*models.Transaction

		finalConditionParams := make([]any, 0, 6)
		finalConditionParams = append(finalConditionParams, conditionParams...)
		finalConditionParams = append(finalConditionParams, minTransactionTime)
		finalConditionParams = append(finalConditionParams, maxTransactionTime)

		sess := s.UserDataDB(uid).NewSession(c).Select("type, account_id, transaction_time, timezone_utc_offset, amount").Where(condition, finalConditionParams...)
		sess = s.appendFilterTagIdsConditionToQuery(sess, uid, maxTransactionTime, minTransactionTime, tagFilters, false)

		err := sess.Limit(pageCountForLoadTransactionAmounts, 0).OrderBy("transaction_time desc").Find(&transactions)

		if err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < pageCountForLoadTransactionAmounts {
			maxTransactionTime = 0
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	return allTransactions, nil
}

// getAllInflowAndOutflowTransactions returns all income, expense and transfer transactions in the given transaction time range (0 means unbounded), loading only the columns needed for statistics
func (s *TransactionService) getAllInflowAndOutflowTransactions(c core.Context, uid int64, minTransactionTime int64, maxTransactionTime int64, tagFilters []*models.TransactionTagFilter, noTags bool, keyword string, matchMode core.MatchMode) ([]*models.Transaction, error) {
	condition := "uid=? AND deleted=? AND (type=? OR type=? OR type=? OR type=?)"
	conditionParams := []any{uid, false, models.TRANSACTION_DB_TYPE_INCOME, models.TRANSACTION_DB_TYPE_EXPENSE, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.TRANSACTION_DB_TYPE_TRANSFER_IN}

	var allTransactions []*models.Transaction

	for maxTransactionTime >= 0 {
		var transactions []*models.Transaction

		finalCondition := condition
		finalConditionParams := slices.Clone(conditionParams)

		if minTransactionTime > 0 {
			finalCondition = finalCondition + " AND transaction_time>=?"
			finalConditionParams = append(finalConditionParams, minTransactionTime)
		}

		if maxTransactionTime > 0 {
			finalCondition = finalCondition + " AND transaction_time<=?"
			finalConditionParams = append(finalConditionParams, maxTransactionTime)
		}

		if keyword != "" {
			if matchMode == core.MATCH_MODE_IGNORE_CASE {
				finalCondition = finalCondition + " AND LOWER(comment) LIKE LOWER(?)"
			} else {
				finalCondition = finalCondition + " AND comment LIKE ?"
			}
			finalConditionParams = append(finalConditionParams, "%%"+keyword+"%%")
		}

		sess := s.UserDataDB(uid).NewSession(c).Select("type, category_id, account_id, related_account_id, transaction_time, timezone_utc_offset, amount").Where(finalCondition, finalConditionParams...)
		sess = s.appendFilterTagIdsConditionToQuery(sess, uid, maxTransactionTime, minTransactionTime, tagFilters, noTags)

		err := sess.Limit(pageCountForLoadTransactionAmounts, 0).OrderBy("transaction_time desc").Find(&transactions)

		if err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < pageCountForLoadTransactionAmounts {
			maxTransactionTime = -1
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	return allTransactions, nil
}
