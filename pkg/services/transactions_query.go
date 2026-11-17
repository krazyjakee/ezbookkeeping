package services

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"xorm.io/builder"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// GetTotalTransactionCountByUid returns total transaction count of user
func (s *TransactionService) GetTotalTransactionCountByUid(c core.Context, uid int64) (int64, error) {
	if uid <= 0 {
		return 0, errs.ErrUserIdInvalid
	}

	count, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Count(&models.Transaction{})

	return count, err
}

// GetAllTransactions returns all transactions
func (s *TransactionService) GetAllTransactions(c core.Context, uid int64, pageCount int32, noDuplicated bool) ([]*models.Transaction, error) {
	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(time.Now().Unix())
	var allTransactions []*models.Transaction

	for maxTransactionTime > 0 {
		transactions, err := s.GetTransactionsByMaxTime(c, uid, maxTransactionTime, 0, 0, nil, nil, nil, false, "", "", core.MATCH_MODE_DEFAULT, false, 1, pageCount, false, noDuplicated)

		if err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < int(pageCount) {
			maxTransactionTime = 0
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	return allTransactions, nil
}

// GetAllSpecifiedTransactions returns all transactions that match given conditions
func (s *TransactionService) GetAllSpecifiedTransactions(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool, pageCount int32, noDuplicated bool) ([]*models.Transaction, error) {
	if maxTransactionTime <= 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(time.Now().Unix())
	}

	var allTransactions []*models.Transaction

	for maxTransactionTime > 0 {
		transactions, err := s.GetTransactionsByMaxTime(c, uid, maxTransactionTime, minTransactionTime, transactionType, categoryIds, accountIds, tagFilters, noTags, amountFilter, keyword, matchMode, mustHavePictures, 1, pageCount, false, noDuplicated)

		if err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < int(pageCount) {
			maxTransactionTime = 0
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	return allTransactions, nil
}

// GetAllTransactionsInOneAccountWithAccountBalanceByMaxTime returns account statement within time range
func (s *TransactionService) GetAllTransactionsInOneAccountWithAccountBalanceByMaxTime(c core.Context, uid int64, pageCount int32, maxTransactionTime int64, minTransactionTime int64, accountId int64, accountCategory models.AccountCategory) ([]*models.TransactionWithAccountBalance, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	if maxTransactionTime <= 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(time.Now().Unix())
	}

	var allTransactions []*models.Transaction

	for maxTransactionTime > 0 {
		var transactions []*models.Transaction
		var err error

		// transactions before the requested range only contribute to the opening balance and are never returned
		if maxTransactionTime < minTransactionTime {
			transactions, err = s.getTransactionBalanceFieldsByMaxTime(c, uid, maxTransactionTime, []int64{accountId}, pageCount, true)
		} else {
			transactions, err = s.GetTransactionsByMaxTime(c, uid, maxTransactionTime, 0, 0, nil, []int64{accountId}, nil, false, "", "", core.MATCH_MODE_DEFAULT, false, 1, pageCount, false, true)
		}

		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < int(pageCount) {
			maxTransactionTime = 0
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	allTransactionsAndAccountBalance := make([]*models.TransactionWithAccountBalance, 0, len(allTransactions))

	if len(allTransactions) < 1 {
		return allTransactionsAndAccountBalance, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil
	}

	totalInflows := big.NewInt(0)
	totalOutflows := big.NewInt(0)
	openingBalance := big.NewInt(0)
	accumulatedBalance := big.NewInt(0)
	lastAccumulatedBalance := big.NewInt(0)

	for i := len(allTransactions) - 1; i >= 0; i-- {
		transaction := allTransactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.RelatedAccountAmount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			accumulatedBalance.Sub(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			accumulatedBalance.Sub(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.Amount))
		} else {
			log.Errorf(c, "[transactions.GetAllTransactionsInOneAccountWithAccountBalanceByMaxTime] trasaction type (%d) is invalid (id:%d)", transaction.TransactionId, transaction.Type)
			return nil, nil, nil, nil, nil, errs.ErrTransactionTypeInvalid
		}

		if transaction.TransactionTime < minTransactionTime {
			openingBalance.Set(accumulatedBalance)
			lastAccumulatedBalance.Set(accumulatedBalance)
			continue
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			if accountCategory.IsAsset() {
				totalInflows.Add(totalInflows, big.NewInt(transaction.RelatedAccountAmount))
			} else if accountCategory.IsLiability() {
				totalOutflows.Sub(totalOutflows, big.NewInt(transaction.RelatedAccountAmount))
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			totalInflows.Add(totalInflows, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			totalOutflows.Add(totalOutflows, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			totalOutflows.Add(totalOutflows, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			totalInflows.Add(totalInflows, big.NewInt(transaction.Amount))
		}

		transactionsAndAccountBalance := &models.TransactionWithAccountBalance{
			Transaction:           transaction,
			AccountOpeningBalance: new(big.Int).Set(lastAccumulatedBalance),
			AccountClosingBalance: new(big.Int).Set(accumulatedBalance),
		}

		lastAccumulatedBalance.Set(accumulatedBalance)
		allTransactionsAndAccountBalance = append(allTransactionsAndAccountBalance, transactionsAndAccountBalance)
	}

	return allTransactionsAndAccountBalance, totalInflows, totalOutflows, openingBalance, new(big.Int).Set(accumulatedBalance), nil
}

// GetAllAccountsDailyOpeningAndClosingBalance returns daily opening and closing balance of all accounts within time range
func (s *TransactionService) GetAllAccountsDailyOpeningAndClosingBalance(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, clientTimezone *time.Location) (map[int32][]*models.TransactionWithAccountBalance, error) {
	if maxTransactionTime <= 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(time.Now().Unix())
	}

	var allTransactions []*models.Transaction

	for maxTransactionTime > 0 {
		transactions, err := s.getTransactionBalanceFieldsByMaxTime(c, uid, maxTransactionTime, nil, pageCountForLoadTransactionAmounts, false)

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

	accountDailyLastBalances := make(map[string]*models.TransactionWithAccountBalance)
	accountDailyBalances := make(map[int32][]*models.TransactionWithAccountBalance)

	if len(allTransactions) < 1 {
		return accountDailyBalances, nil
	}

	accumulatedBalances := make(map[int64]*big.Int)
	accumulatedBalancesBeforeStartTime := make(map[int64]*big.Int)

	for i := len(allTransactions) - 1; i >= 0; i-- {
		transaction := allTransactions[i]
		accumulatedBalance := accumulatedBalances[transaction.AccountId]

		if accumulatedBalance == nil {
			accumulatedBalance = big.NewInt(0)
		}

		lastAccumulatedBalance := new(big.Int).Set(accumulatedBalance)

		if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.RelatedAccountAmount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			accumulatedBalance.Sub(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			accumulatedBalance.Sub(accumulatedBalance, big.NewInt(transaction.Amount))
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			accumulatedBalance.Add(accumulatedBalance, big.NewInt(transaction.Amount))
		} else {
			log.Errorf(c, "[transactions.GetAllTransactionsWithAccountBalanceByMaxTime] trasaction type (%d) is invalid (id:%d)", transaction.TransactionId, transaction.Type)
			return nil, errs.ErrTransactionTypeInvalid
		}

		accumulatedBalances[transaction.AccountId] = accumulatedBalance

		if transaction.TransactionTime < minTransactionTime {
			accumulatedBalancesBeforeStartTime[transaction.AccountId] = new(big.Int).Set(accumulatedBalance)
			continue
		}

		yearMonthDay := utils.FormatUnixTimeToNumericYearMonthDay(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime), clientTimezone)
		groupKey := fmt.Sprintf("%d_%d", yearMonthDay, transaction.AccountId)
		dailyAccountBalance, exists := accountDailyLastBalances[groupKey]

		if exists {
			dailyAccountBalance.AccountClosingBalance.Set(accumulatedBalance)
		} else {
			dailyAccountBalance = &models.TransactionWithAccountBalance{
				Transaction: &models.Transaction{
					AccountId: transaction.AccountId,
				},
				AccountOpeningBalance: lastAccumulatedBalance,
				AccountClosingBalance: new(big.Int).Set(accumulatedBalance),
			}
			accountDailyLastBalances[groupKey] = dailyAccountBalance
		}
	}

	firstTransactionTime := allTransactions[len(allTransactions)-1].TransactionTime

	if minTransactionTime > firstTransactionTime {
		firstTransactionTime = minTransactionTime
	}

	firstYearMonthDay := utils.FormatUnixTimeToNumericYearMonthDay(utils.GetUnixTimeFromTransactionTime(firstTransactionTime), clientTimezone)

	// fill in the opening balance for accounts that do not have transactions on the first day
	for accountId, accumulatedBalance := range accumulatedBalancesBeforeStartTime {
		if accumulatedBalance.Sign() == 0 {
			continue
		}

		groupKey := fmt.Sprintf("%d_%d", firstYearMonthDay, accountId)

		if _, exists := accountDailyLastBalances[groupKey]; exists {
			continue
		}

		accountDailyLastBalances[groupKey] = &models.TransactionWithAccountBalance{
			Transaction: &models.Transaction{
				AccountId: accountId,
			},
			AccountOpeningBalance: new(big.Int).Set(accumulatedBalance),
			AccountClosingBalance: new(big.Int).Set(accumulatedBalance),
		}
	}

	for groupKey, transactionWithAccountBalance := range accountDailyLastBalances {
		groupKeyParts := strings.Split(groupKey, "_")
		yearMonthDay, _ := utils.StringToInt32(groupKeyParts[0])
		dailyAccountBalances, exists := accountDailyBalances[yearMonthDay]

		if !exists {
			dailyAccountBalances = make([]*models.TransactionWithAccountBalance, 0)
		}

		dailyAccountBalances = append(dailyAccountBalances, transactionWithAccountBalance)
		accountDailyBalances[yearMonthDay] = dailyAccountBalances
	}

	return accountDailyBalances, nil
}

// GetTransactionsByMaxTimeUpToCount returns transactions before given time and up to given count
func (s *TransactionService) GetTransactionsByMaxTimeUpToCount(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool, page int32, count int32, pageCount int32, needOneMoreItem bool, noDuplicated bool) ([]*models.Transaction, error) {
	if maxTransactionTime <= 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(time.Now().Unix())
	}

	if page < 0 {
		return nil, errs.ErrPageIndexInvalid
	} else if page == 0 {
		page = 1
	}

	if count < 1 {
		return nil, errs.ErrPageCountInvalid
	}

	finalExpectedCount := int(count)

	if needOneMoreItem {
		finalExpectedCount++
	}

	var allTransactions []*models.Transaction
	startOffset := int((page - 1) * count)
	firstFetchCount := int(pageCount)

	if finalExpectedCount < firstFetchCount {
		firstFetchCount = finalExpectedCount
	}

	transactions, err := s.getTransactionsByMaxTimeWithOffset(c, uid, maxTransactionTime, minTransactionTime, transactionType, categoryIds, accountIds, tagFilters, noTags, amountFilter, keyword, matchMode, mustHavePictures, startOffset, firstFetchCount, noDuplicated)

	if err != nil {
		return nil, err
	}

	allTransactions = append(allTransactions, transactions...)

	if len(transactions) < firstFetchCount {
		return allTransactions, nil
	}

	maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1

	for len(allTransactions) < finalExpectedCount && maxTransactionTime > 0 {
		remainingCount := finalExpectedCount - len(allTransactions)
		fetchCount := int(pageCount)

		if remainingCount < fetchCount {
			fetchCount = remainingCount
		}

		transactions, err := s.GetTransactionsByMaxTime(c, uid, maxTransactionTime, minTransactionTime, transactionType, categoryIds, accountIds, tagFilters, noTags, amountFilter, keyword, matchMode, mustHavePictures, 1, int32(fetchCount), false, noDuplicated)

		if err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, transactions...)

		if len(transactions) < fetchCount {
			break
		}

		maxTransactionTime = transactions[len(transactions)-1].TransactionTime - 1
	}

	return allTransactions, nil
}

// GetTransactionsByMaxTime returns transactions before given time
func (s *TransactionService) GetTransactionsByMaxTime(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool, page int32, count int32, needOneMoreItem bool, noDuplicated bool) ([]*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if page < 0 {
		return nil, errs.ErrPageIndexInvalid
	} else if page == 0 {
		page = 1
	}

	if count < 1 {
		return nil, errs.ErrPageCountInvalid
	}

	finalCount := int(count)

	if needOneMoreItem {
		finalCount++
	}

	return s.getTransactionsByMaxTimeWithOffset(c, uid, maxTransactionTime, minTransactionTime, transactionType, categoryIds, accountIds, tagFilters, noTags, amountFilter, keyword, matchMode, mustHavePictures, int(count*(page-1)), finalCount, noDuplicated)
}

// getTransactionsByMaxTimeWithOffset returns transactions before given time with explicit offset and limit
func (s *TransactionService) getTransactionsByMaxTimeWithOffset(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool, offset int, limit int, noDuplicated bool) ([]*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var err error
	var transactionDbType models.TransactionDbType = 0

	if transactionType > 0 {
		transactionDbType, err = transactionType.ToTransactionDbType()

		if err != nil {
			return nil, err
		}
	}

	var transactions []*models.Transaction

	condition, conditionParams := s.buildTransactionQueryCondition(uid, maxTransactionTime, minTransactionTime, transactionDbType, categoryIds, accountIds, amountFilter, keyword, matchMode, noDuplicated)
	sess := s.UserDataDB(uid).NewSession(c).Where(condition, conditionParams...)
	sess = s.appendFilterTagIdsConditionToQuery(sess, uid, maxTransactionTime, minTransactionTime, tagFilters, noTags)
	sess = s.appendFilterPicturesConditionToQuery(sess, uid, mustHavePictures)

	err = sess.Limit(limit, offset).OrderBy("transaction_time desc").Find(&transactions)

	return transactions, err
}

// getTransactionBalanceFieldsByMaxTime returns the same transactions as GetTransactionsByMaxTime filtered only by account ids, but loads only the columns needed to accumulate account balances
func (s *TransactionService) getTransactionBalanceFieldsByMaxTime(c core.Context, uid int64, maxTransactionTime int64, accountIds []int64, count int32, noDuplicated bool) ([]*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if count < 1 {
		return nil, errs.ErrPageCountInvalid
	}

	var transactions []*models.Transaction
	condition, conditionParams := s.buildTransactionQueryCondition(uid, maxTransactionTime, 0, 0, nil, accountIds, "", "", core.MATCH_MODE_DEFAULT, noDuplicated)
	err := s.UserDataDB(uid).NewSession(c).Cols("transaction_id", "type", "account_id", "transaction_time", "amount", "related_account_amount").Where(condition, conditionParams...).Limit(int(count), 0).OrderBy("transaction_time desc").Find(&transactions)

	return transactions, err
}

// GetTransactionsInMonthByPage returns all transactions in given year and month
func (s *TransactionService) GetTransactionsInMonthByPage(c core.Context, uid int64, year int32, month int32, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool) ([]*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var err error
	var transactionDbType models.TransactionDbType = 0

	if transactionType > 0 {
		transactionDbType, err = transactionType.ToTransactionDbType()

		if err != nil {
			return nil, err
		}
	}

	minTransactionTime, maxTransactionTime, err := utils.GetTransactionTimeRangeByYearMonth(year, month)

	if err != nil {
		return nil, errs.ErrSystemError
	}

	var transactions []*models.Transaction

	condition, conditionParams := s.buildTransactionQueryCondition(uid, maxTransactionTime, minTransactionTime, transactionDbType, categoryIds, accountIds, amountFilter, keyword, matchMode, true)
	sess := s.UserDataDB(uid).NewSession(c).Where(condition, conditionParams...)
	sess = s.appendFilterTagIdsConditionToQuery(sess, uid, maxTransactionTime, minTransactionTime, tagFilters, noTags)
	sess = s.appendFilterPicturesConditionToQuery(sess, uid, mustHavePictures)

	err = sess.OrderBy("transaction_time desc").Find(&transactions)

	transactionsInMonth := make([]*models.Transaction, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		transactionUnixTime := utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime)
		transactionTimeZone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)

		if utils.IsUnixTimeEqualsYearAndMonth(transactionUnixTime, transactionTimeZone, year, month) {
			transactionsInMonth = append(transactionsInMonth, transaction)
		}
	}

	return transactionsInMonth, err
}

// GetTransactionByTransactionId returns a transaction model according to transaction id
func (s *TransactionService) GetTransactionByTransactionId(c core.Context, uid int64, transactionId int64) (*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if transactionId <= 0 {
		return nil, errs.ErrTransactionIdInvalid
	}

	transaction := &models.Transaction{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(transactionId).Where("uid=? AND deleted=?", uid, false).Get(transaction)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrTransactionNotFound
	}

	return transaction, nil
}

// GetTransactionsByTransactionIds returns transaction models according to transaction ids
func (s *TransactionService) GetTransactionsByTransactionIds(c core.Context, uid int64, transactionIds []int64) ([]*models.Transaction, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if len(transactionIds) <= 0 {
		return nil, errs.ErrTransactionIdInvalid
	}

	var transactions []*models.Transaction
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("transaction_id", transactionIds).Find(&transactions)

	return transactions, err
}

// GetAllTransactionCount returns total count of transactions
func (s *TransactionService) GetAllTransactionCount(c core.Context, uid int64) (int64, error) {
	return s.GetTransactionCount(c, uid, 0, 0, 0, nil, nil, nil, false, "", "", core.MATCH_MODE_DEFAULT, false)
}

// GetTransactionCount returns count of transactions
func (s *TransactionService) GetTransactionCount(c core.Context, uid int64, maxTransactionTime int64, minTransactionTime int64, transactionType models.TransactionType, categoryIds []int64, accountIds []int64, tagFilters []*models.TransactionTagFilter, noTags bool, amountFilter string, keyword string, matchMode core.MatchMode, mustHavePictures bool) (int64, error) {
	if uid <= 0 {
		return 0, errs.ErrUserIdInvalid
	}

	var err error
	var transactionDbType models.TransactionDbType = 0

	if transactionType > 0 {
		transactionDbType, err = transactionType.ToTransactionDbType()

		if err != nil {
			return 0, err
		}
	}

	condition, conditionParams := s.buildTransactionQueryCondition(uid, maxTransactionTime, minTransactionTime, transactionDbType, categoryIds, accountIds, amountFilter, keyword, matchMode, true)
	sess := s.UserDataDB(uid).NewSession(c).Where(condition, conditionParams...)
	sess = s.appendFilterTagIdsConditionToQuery(sess, uid, maxTransactionTime, minTransactionTime, tagFilters, noTags)
	sess = s.appendFilterPicturesConditionToQuery(sess, uid, mustHavePictures)

	return sess.Count(&models.Transaction{})
}

func (s *TransactionService) buildTransactionQueryCondition(uid int64, maxTransactionTime int64, minTransactionTime int64, transactionDbType models.TransactionDbType, categoryIds []int64, accountIds []int64, amountFilter string, keyword string, matchMode core.MatchMode, noDuplicated bool) (string, []any) {
	condition := "uid=? AND deleted=?"
	conditionParams := make([]any, 0, 16)
	conditionParams = append(conditionParams, uid)
	conditionParams = append(conditionParams, false)

	if maxTransactionTime > 0 {
		condition = condition + " AND transaction_time<=?"
		conditionParams = append(conditionParams, maxTransactionTime)
	}

	if minTransactionTime > 0 {
		condition = condition + " AND transaction_time>=?"
		conditionParams = append(conditionParams, minTransactionTime)
	}

	var accountIdsCondition strings.Builder
	accountIdConditionParams := make([]any, 0, len(accountIds))

	for i := 0; i < len(accountIds); i++ {
		if i > 0 {
			accountIdsCondition.WriteString(",")
		}

		accountIdsCondition.WriteString("?")
		accountIdConditionParams = append(accountIdConditionParams, accountIds[i])
	}

	if models.TRANSACTION_DB_TYPE_MODIFY_BALANCE <= transactionDbType && transactionDbType <= models.TRANSACTION_DB_TYPE_EXPENSE {
		condition = condition + " AND type=?"
		conditionParams = append(conditionParams, transactionDbType)
	} else if transactionDbType == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transactionDbType == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		if len(accountIds) == 0 {
			condition = condition + " AND type=?"
			conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_OUT)
		} else if len(accountIds) == 1 {
			condition = condition + " AND (type=? OR type=?)"
			conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_OUT)
			conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_IN)
		} else { // len(accountsIds) > 1
			condition = condition + " AND (type=? OR (type=? AND related_account_id NOT IN (" + accountIdsCondition.String() + ")))"
			conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_OUT)
			conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_IN)
			conditionParams = append(conditionParams, accountIdConditionParams...)
		}
	} else {
		if noDuplicated {
			if len(accountIds) == 0 {
				condition = condition + " AND (type=? OR type=? OR type=? OR type=?)"
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_INCOME)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_EXPENSE)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_OUT)
			} else if len(accountIds) == 1 {
				// Do Nothing
			} else { // len(accountsIds) > 1
				condition = condition + " AND (type=? OR type=? OR type=? OR type=? OR (type=? AND related_account_id NOT IN (" + accountIdsCondition.String() + ")))"
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_INCOME)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_EXPENSE)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_OUT)
				conditionParams = append(conditionParams, models.TRANSACTION_DB_TYPE_TRANSFER_IN)
				conditionParams = append(conditionParams, accountIdConditionParams...)
			}
		}
	}

	if len(categoryIds) > 0 {
		var conditions strings.Builder

		for i := 0; i < len(categoryIds); i++ {
			if i > 0 {
				conditions.WriteString(",")
			}

			conditions.WriteString("?")
			conditionParams = append(conditionParams, categoryIds[i])
		}

		if conditions.Len() > 1 {
			condition = condition + " AND category_id IN (" + conditions.String() + ")"
		} else {
			condition = condition + " AND category_id = " + conditions.String()
		}
	}

	if len(accountIds) > 0 {
		if accountIdsCondition.Len() > 1 {
			condition = condition + " AND account_id IN (" + accountIdsCondition.String() + ")"
		} else {
			condition = condition + " AND account_id = " + accountIdsCondition.String()
		}

		conditionParams = append(conditionParams, accountIdConditionParams...)
	}

	if amountFilter != "" {
		amountFilterItems := strings.Split(amountFilter, ":")

		if len(amountFilterItems) == 2 && amountFilterItems[0] == "gt" {
			value, err := utils.StringToInt64(amountFilterItems[1])

			if err == nil {
				condition = condition + " AND amount > ?"
				conditionParams = append(conditionParams, value)
			}
		} else if len(amountFilterItems) == 2 && amountFilterItems[0] == "lt" {
			value, err := utils.StringToInt64(amountFilterItems[1])

			if err == nil {
				condition = condition + " AND amount < ?"
				conditionParams = append(conditionParams, value)
			}
		} else if len(amountFilterItems) == 2 && amountFilterItems[0] == "eq" {
			value, err := utils.StringToInt64(amountFilterItems[1])

			if err == nil {
				condition = condition + " AND amount = ?"
				conditionParams = append(conditionParams, value)
			}
		} else if len(amountFilterItems) == 2 && amountFilterItems[0] == "ne" {
			value, err := utils.StringToInt64(amountFilterItems[1])

			if err == nil {
				condition = condition + " AND amount <> ?"
				conditionParams = append(conditionParams, value)
			}
		} else if len(amountFilterItems) == 3 && amountFilterItems[0] == "bt" {
			value1, err := utils.StringToInt64(amountFilterItems[1])
			value2, err := utils.StringToInt64(amountFilterItems[2])

			if err == nil {
				condition = condition + " AND amount >= ? AND amount <= ?"
				conditionParams = append(conditionParams, value1)
				conditionParams = append(conditionParams, value2)
			}
		} else if len(amountFilterItems) == 3 && amountFilterItems[0] == "nb" {
			value1, err := utils.StringToInt64(amountFilterItems[1])
			value2, err := utils.StringToInt64(amountFilterItems[2])

			if err == nil {
				condition = condition + " AND (amount < ? OR amount > ?)"
				conditionParams = append(conditionParams, value1)
				conditionParams = append(conditionParams, value2)
			}
		}
	}

	if keyword != "" {
		if matchMode == core.MATCH_MODE_IGNORE_CASE {
			condition = condition + " AND LOWER(comment) LIKE LOWER(?)"
		} else {
			condition = condition + " AND comment LIKE ?"
		}
		conditionParams = append(conditionParams, "%%"+keyword+"%%")
	}

	return condition, conditionParams
}

func (s *TransactionService) appendFilterTagIdsConditionToQuery(sess datastore.DatabaseSession, uid int64, maxTransactionTime int64, minTransactionTime int64, tagFilters []*models.TransactionTagFilter, noTags bool) datastore.DatabaseSession {
	if noTags {
		subQueryCondition := builder.And(builder.Eq{"uid": uid}, builder.Eq{"deleted": false})

		if maxTransactionTime > 0 {
			subQueryCondition = subQueryCondition.And(builder.Lte{"transaction_time": maxTransactionTime})
		}

		if minTransactionTime > 0 {
			subQueryCondition = subQueryCondition.And(builder.Gte{"transaction_time": minTransactionTime})
		}

		subQuery := builder.Select("transaction_id").From("transaction_tag_index").Where(subQueryCondition)
		sess.NotIn("transaction_id", subQuery).NotIn("related_id", subQuery)
		return sess
	}

	if len(tagFilters) < 1 {
		return sess
	}

	for i := 0; i < len(tagFilters); i++ {
		tagFilter := tagFilters[i]
		subQueryCondition := builder.And(builder.Eq{"uid": uid}, builder.Eq{"deleted": false})

		if maxTransactionTime > 0 {
			subQueryCondition = subQueryCondition.And(builder.Lte{"transaction_time": maxTransactionTime})
		}

		if minTransactionTime > 0 {
			subQueryCondition = subQueryCondition.And(builder.Gte{"transaction_time": minTransactionTime})
		}

		subQueryCondition = subQueryCondition.And(builder.In("tag_id", tagFilter.TagIds))
		subQuery := builder.Select("transaction_id").From("transaction_tag_index").Where(subQueryCondition)

		if tagFilter.Type == models.TRANSACTION_TAG_FILTER_HAS_ALL || tagFilter.Type == models.TRANSACTION_TAG_FILTER_NOT_HAS_ALL {
			subQuery = subQuery.GroupBy("transaction_id").Having(fmt.Sprintf("COUNT(DISTINCT tag_id) >= %d", len(tagFilter.TagIds)))
		}

		if tagFilter.Type == models.TRANSACTION_TAG_FILTER_HAS_ANY || tagFilter.Type == models.TRANSACTION_TAG_FILTER_HAS_ALL {
			sess.And(builder.Or(builder.In("transaction_id", subQuery), builder.In("related_id", subQuery)))
		} else if tagFilter.Type == models.TRANSACTION_TAG_FILTER_NOT_HAS_ANY || tagFilter.Type == models.TRANSACTION_TAG_FILTER_NOT_HAS_ALL {
			sess.NotIn("transaction_id", subQuery).NotIn("related_id", subQuery)
		}
	}

	return sess
}

func (s *TransactionService) appendFilterPicturesConditionToQuery(sess datastore.DatabaseSession, uid int64, mustHavePictures bool) datastore.DatabaseSession {
	if !mustHavePictures {
		return sess
	}

	subQuery := builder.Select("transaction_id").From("transaction_picture_info").Where(builder.And(builder.Eq{"uid": uid}, builder.Eq{"deleted": false}, builder.Neq{"transaction_id": models.TransactionPictureNewPictureTransactionId}))
	sess.And(builder.Or(builder.In("transaction_id", subQuery), builder.In("related_id", subQuery)))

	return sess
}

// GetTransactionMapByList returns a transaction map by a list
func (s *TransactionService) GetTransactionMapByList(transactions []*models.Transaction) map[int64]*models.Transaction {
	transactionMap := make(map[int64]*models.Transaction)

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		transactionMap[transaction.TransactionId] = transaction
	}

	return transactionMap
}

// GetTransactionIds returns transaction ids list
func (s *TransactionService) GetTransactionIds(transactions []*models.Transaction) []int64 {
	transactionIds := make([]int64, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transactionIds[i] = transactions[i].TransactionId
	}

	return transactionIds
}
