package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const pageCountForMovingAccountTransactions = 1000

// TransactionCreateBatchHandler saves a batch of new transactions by request parameters for current user
func (a *TransactionsApi) TransactionCreateBatchHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionCreateBatchReq models.TransactionCreateBatchRequest
	err := c.ShouldBindJSON(&transactionCreateBatchReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateBatchHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateBatchHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	if len(transactionCreateBatchReq.Transactions) < 1 {
		return nil, errs.ErrTransactionBatchCreateNoTransactions
	}

	if len(transactionCreateBatchReq.Transactions) > 10 {
		return nil, errs.ErrTransactionBatchCreateTooManyTransactions
	}

	newTransactionTagIdsMap := make(map[int][]int64, len(transactionCreateBatchReq.Transactions))

	for i := 0; i < len(transactionCreateBatchReq.Transactions); i++ {
		transactionCreateReq := transactionCreateBatchReq.Transactions[i]
		tagIds, err := utils.StringArrayToInt64Array(transactionCreateReq.TagIds)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionCreateBatchHandler] parse tag ids failed of transaction \"index:%d\", because %s", i, err.Error())
			return nil, errs.ErrTransactionTagIdInvalid
		}

		if len(tagIds) > models.MaximumTagsCountOfTransaction {
			return nil, errs.ErrTransactionHasTooManyTags
		}

		if transactionCreateReq.Type != models.TRANSACTION_TYPE_INCOME && transactionCreateReq.Type != models.TRANSACTION_TYPE_EXPENSE {
			log.Warnf(c, "[transactions.TransactionCreateBatchHandler] transaction type of transaction \"index:%d\" is invalid", i)
			return nil, errs.ErrTransactionTypeInvalid
		}

		if transactionCreateReq.DestinationAccountId != 0 {
			log.Warnf(c, "[transactions.TransactionCreateBatchHandler] non-transfer transaction \"index:%d\" destination account cannot be set", i)
			return nil, errs.ErrTransactionDestinationAccountCannotBeSet
		}

		if transactionCreateReq.DestinationAmount != 0 {
			log.Warnf(c, "[transactions.TransactionCreateBatchHandler] non-transfer transaction \"index:%d\" destination amount cannot be set", i)
			return nil, errs.ErrTransactionDestinationAmountCannotBeSet
		}

		newTransactionTagIdsMap[i] = tagIds
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionCreateBatchHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	newTransactions := make([]*models.Transaction, len(transactionCreateBatchReq.Transactions))

	for i := 0; i < len(transactionCreateBatchReq.Transactions); i++ {
		transactionCreateReq := transactionCreateBatchReq.Transactions[i]
		transaction := a.createNewTransactionModel(uid, transactionCreateReq, c.ClientIP())
		newTransactions[i] = transaction
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, newTransactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionCreateBatchHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(newTransactions); i++ {
		transaction := newTransactions[i]
		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionCreateBatchHandler] transaction \"index:%d\" is not editable for user \"uid:%d\"", i, uid)
			return nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
		}
	}

	err = a.transactions.BatchCreateTransactions(c, user.Uid, newTransactions, newTransactionTagIdsMap, nil)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionCreateBatchHandler] failed to batch create transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionCreateBatchHandler] user \"uid:%d\" has created %d transactions", uid, len(newTransactions))

	transactionResps := make([]*models.TransactionInfoResponse, len(newTransactions))

	for i := 0; i < len(newTransactions); i++ {
		transaction := newTransactions[i]
		transactionResp := transaction.ToTransactionInfoResponse(newTransactionTagIdsMap[i], true)
		transactionResps[i] = transactionResp
	}

	return transactionResps, nil
}

// TransactionBatchUpdateCategoriesHandler batch updates categories of transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchUpdateCategoriesHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchUpdateReq models.TransactionBatchUpdateCategoryRequest
	err := c.ShouldBindJSON(&transactionBatchUpdateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TransactionIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	category, err := a.transactionCategories.GetCategoryByCategoryId(c, uid, transactionBatchUpdateReq.CategoryId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] failed to get category \"id:%d\" for user \"uid:%d\", because %s", transactionBatchUpdateReq.CategoryId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
		log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] transaction category \"id:%d\" is not a sub category", category.CategoryId)
		return nil, errs.ErrCannotUsePrimaryCategoryForTransaction
	}

	var expectedTransactionType models.TransactionDbType

	if category.Type == models.CATEGORY_TYPE_EXPENSE {
		expectedTransactionType = models.TRANSACTION_DB_TYPE_EXPENSE
	} else if category.Type == models.CATEGORY_TYPE_INCOME {
		expectedTransactionType = models.TRANSACTION_DB_TYPE_INCOME
	} else if category.Type == models.CATEGORY_TYPE_TRANSFER {
		expectedTransactionType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allTransactionIds := make([]int64, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type != expectedTransactionType {
			log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] transaction \"id:%d\" type is not expected type \"%d\" for user \"uid:%d\"", transaction.TransactionId, expectedTransactionType, uid)
			return nil, errs.ErrTransactionTypeInvalid
		}

		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}

		allTransactionIds = append(allTransactionIds, transaction.TransactionId)

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			allTransactionIds = append(allTransactionIds, transaction.RelatedId)
		}
	}

	err = a.transactions.BatchUpdateTransactionsCategory(c, uid, allTransactionIds, category.CategoryId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateCategoriesHandler] failed to batch update transactions category for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionBatchUpdateCategoriesHandler] user \"uid:%d\" has batch updated category of %d transactions successfully", uid, len(transactionBatchUpdateReq.TransactionIds))
	return true, nil
}

// TransactionBatchUpdateAccountsHandler batch updates accounts of transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchUpdateAccountsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchUpdateReq models.TransactionBatchUpdateAccountRequest
	err := c.ShouldBindJSON(&transactionBatchUpdateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TransactionIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchUpdateAccountsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	allAccounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateAccountsHandler] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountMap := a.accounts.GetAccountMapByList(allAccounts)
	account, exists := accountMap[transactionBatchUpdateReq.AccountId]

	if !exists || account == nil {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] account \"id:%d\" does not exist for user \"uid:%d\"", transactionBatchUpdateReq.AccountId, uid)
		return nil, errs.ErrAccountNotFound
	}

	if account.Hidden {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] account \"id:%d\" is hidden for user \"uid:%d\"", account.AccountId, uid)
		return nil, errs.ErrCannotMoveTransactionFromOrToHiddenAccount
	}

	if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] account \"id:%d\" is a parent account, cannot be used for transaction of user \"uid:%d\"", account.AccountId, uid)
		return nil, errs.ErrCannotModifyTransactionInParentAccount
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchUpdateAccountsHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot modify transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionTypeInvalid
		}

		if transactionBatchUpdateReq.IsDestinationAccount && transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot update destination account of non-transfer transaction \"id:%d\" for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionDestinationAccountCannotBeSet
		}

		if !transactionBatchUpdateReq.IsDestinationAccount && account.AccountId == transaction.RelatedAccountId {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot update account to same destination account of transaction \"id:%d\" for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
		} else if transactionBatchUpdateReq.IsDestinationAccount && account.AccountId == transaction.AccountId {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot update destination account to same source account of transaction \"id:%d\" for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
		}

		var oldAccount *models.Account

		if !transactionBatchUpdateReq.IsDestinationAccount {
			oldAccount = accountMap[transaction.AccountId]
		} else if transactionBatchUpdateReq.IsDestinationAccount {
			oldAccount = accountMap[transaction.RelatedAccountId]
		}

		if oldAccount == nil {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] the original account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrAccountNotFound
		}

		if oldAccount.Hidden {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] the original account of transaction \"id:%d\" is hidden for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotMoveTransactionFromOrToHiddenAccount
		}

		if oldAccount.Currency != account.Currency {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] cannot update account of transaction \"id:%d\", because the original account currency \"%s\" is different from updated account currency \"%s\" for user \"uid:%d\"", transaction.TransactionId, oldAccount.Currency, account.Currency, uid)
			return nil, errs.ErrCannotMoveTransactionBetweenAccountsWithDifferentCurrencies
		}

		newSourceAccount := accountMap[transaction.AccountId]
		newDestinationAccount := accountMap[transaction.RelatedAccountId]

		if !transactionBatchUpdateReq.IsDestinationAccount && transaction.AccountId != account.AccountId {
			newSourceAccount = account
		} else if transactionBatchUpdateReq.IsDestinationAccount && transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && transaction.RelatedAccountId != account.AccountId {
			newDestinationAccount = account
		}

		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, accountMap[transaction.AccountId], accountMap[transaction.RelatedAccountId])
		newTransactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, newSourceAccount, newDestinationAccount)

		if !transactionEditable || !newTransactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}
	}

	updatedCount := 0

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if !transactionBatchUpdateReq.IsDestinationAccount && transaction.AccountId != account.AccountId {
			transaction.AccountId = account.AccountId
		} else if transactionBatchUpdateReq.IsDestinationAccount && transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && transaction.RelatedAccountId != account.AccountId {
			transaction.RelatedAccountId = account.AccountId
		} else {
			log.Warnf(c, "[transactions.TransactionBatchUpdateAccountsHandler] skip updating transaction \"id:%d\", because the original account is same as updated account for user \"uid:%d\"", transaction.TransactionId, uid)
			continue
		}

		err = a.transactions.ModifyTransaction(c, transaction, false, 0, nil, nil, nil, nil)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionBatchUpdateAccountsHandler] failed to update transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		updatedCount++
	}

	if updatedCount < 1 {
		return nil, errs.ErrNothingWillBeUpdated
	}

	log.Infof(c, "[transactions.TransactionBatchUpdateAccountsHandler] user \"uid:%d\" has batch updated account of %d transactions successfully", uid, updatedCount)
	return true, nil
}

// TransactionBatchAddTagsHandler batch add tags to transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchAddTagsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchUpdateReq models.TransactionBatchAddTagsRequest
	err := c.ShouldBindJSON(&transactionBatchUpdateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TransactionIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	tagIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TagIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] parse tag ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionTagIdInvalid
	}

	tagIds = utils.ToUniqueInt64Slice(tagIds)

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	tags, err := a.transactionTags.GetTagsByTagIds(c, uid, tagIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if len(tags) != len(tagIds) {
		log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] some tags do not exist for user \"uid:%d\"", uid)
		return nil, errs.ErrTransactionTagNotFound
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionTagIndexes, err := a.transactionTags.GetAllTagIdsOfTransactions(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to get transactions tag indexes for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allNewTransactionTagIndexes := make(map[int64][]int64, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] cannot modify transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionTypeInvalid
		}

		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchAddTagsHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}

		existedTagIds := transactionTagIndexes[transaction.TransactionId]
		existedTagIdsMap := make(map[int64]bool, len(existedTagIds))

		for j := 0; j < len(existedTagIds); j++ {
			existedTagIdsMap[existedTagIds[j]] = true
		}

		var newTagIds []int64

		for j := 0; j < len(tagIds); j++ {
			tagId := tagIds[j]

			if _, exists := existedTagIdsMap[tagId]; !exists {
				newTagIds = append(newTagIds, tagId)
			}
		}

		allNewTransactionTagIndexes[transaction.TransactionId] = newTagIds
	}

	err = a.transactions.BatchAddTagsToTransactions(c, uid, transactions, allNewTransactionTagIndexes)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchAddTagsHandler] failed to batch update transactions tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionBatchAddTagsHandler] user \"uid:%d\" has batch updated tag of %d transactions successfully", uid, len(allNewTransactionTagIndexes))
	return true, nil
}

// TransactionBatchRemoveTagsHandler batch remove tags from transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchRemoveTagsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchUpdateReq models.TransactionBatchRemoveTagsRequest
	err := c.ShouldBindJSON(&transactionBatchUpdateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TransactionIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	tagIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TagIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] parse tag ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionTagIdInvalid
	}

	tagIds = utils.ToUniqueInt64Slice(tagIds)

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchRemoveTagsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	tags, err := a.transactionTags.GetTagsByTagIds(c, uid, tagIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchRemoveTagsHandler] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if len(tags) != len(tagIds) {
		log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] some tags do not exist for user \"uid:%d\"", uid)
		return nil, errs.ErrTransactionTagNotFound
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchRemoveTagsHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchRemoveTagsHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allTransactionIds := make([]int64, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] cannot modify transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionTypeInvalid
		}

		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchRemoveTagsHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}

		allTransactionIds = append(allTransactionIds, transaction.TransactionId)
	}

	err = a.transactions.BatchRemoveTagsFromTransactions(c, uid, allTransactionIds, tagIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchRemoveTagsHandler] failed to batch update transactions tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionBatchRemoveTagsHandler] user \"uid:%d\" has batch updated tag of %d transactions successfully", uid, len(allTransactionIds))
	return true, nil
}

// TransactionBatchClearTagsHandler batch clear all tags from transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchClearTagsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchUpdateReq models.TransactionBatchClearTagsRequest
	err := c.ShouldBindJSON(&transactionBatchUpdateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchClearTagsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchClearTagsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchUpdateReq.TransactionIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchClearTagsHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchClearTagsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchClearTagsHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchClearTagsHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allTransactionIds := make([]int64, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			log.Warnf(c, "[transactions.TransactionBatchClearTagsHandler] cannot modify transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionTypeInvalid
		}

		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchClearTagsHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}

		allTransactionIds = append(allTransactionIds, transaction.TransactionId)
	}

	err = a.transactions.BatchClearAllTagsFromTransactions(c, uid, allTransactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchClearTagsHandler] failed to batch update transactions tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionBatchClearTagsHandler] user \"uid:%d\" has batch updated tag of %d transactions successfully", uid, len(allTransactionIds))
	return true, nil
}

// TransactionMoveAllBetweenAccountsHandler moves all transactions from one account to another account for current user
func (a *TransactionsApi) TransactionMoveAllBetweenAccountsHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionMoveReq models.TransactionMoveBetweenAccountsRequest
	err := c.ShouldBindJSON(&transactionMoveReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	if transactionMoveReq.FromAccountId == transactionMoveReq.ToAccountId {
		return nil, errs.ErrCannotMoveTransactionToSameAccount
	}

	accountMap, err := a.accounts.GetAccountsByAccountIds(c, uid, []int64{transactionMoveReq.FromAccountId, transactionMoveReq.ToAccountId})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	fromAccount, exists := accountMap[transactionMoveReq.FromAccountId]

	if !exists {
		return nil, errs.ErrSourceAccountNotFound
	}

	toAccount, exists := accountMap[transactionMoveReq.ToAccountId]

	if !exists {
		return nil, errs.ErrDestinationAccountNotFound
	}

	if fromAccount.Hidden || toAccount.Hidden {
		return nil, errs.ErrCannotMoveTransactionFromOrToHiddenAccount
	}

	if fromAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS || toAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
		return nil, errs.ErrCannotMoveTransactionFromOrToParentAccount
	}

	if fromAccount.Currency != toAccount.Currency {
		return nil, errs.ErrCannotMoveTransactionBetweenAccountsWithDifferentCurrencies
	}

	transactions, err := a.transactions.GetAllSpecifiedTransactions(c, uid, 0, 0, 0, nil, []int64{fromAccount.AccountId}, nil, false, "", "", core.MATCH_MODE_DEFAULT, false, pageCountForMovingAccountTransactions, true)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] failed to get all transactions of account \"id:%d\" for user \"uid:%d\", because %s", fromAccount.AccountId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])
		newTransactionEditable := transactionEditable

		if transaction.AccountId == fromAccount.AccountId {
			newTransactionEditable = user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, toAccount, allUsedAccounts[transaction.RelatedAccountId])
		} else if transaction.RelatedAccountId == fromAccount.AccountId {
			newTransactionEditable = user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], toAccount)
		}

		if !transactionEditable || !newTransactionEditable {
			log.Warnf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
		}
	}

	err = a.transactions.MoveAllTransactionsBetweenAccounts(c, uid, fromAccount.AccountId, toAccount.AccountId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] failed to move all transactions from account \"id:%d\" to account \"id:%d\" for user \"uid:%d\", because %s", transactionMoveReq.FromAccountId, transactionMoveReq.ToAccountId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionMoveAllBetweenAccountsHandler] user \"uid:%d\" has moved all transactions from account \"id:%d\" to account \"id:%d\" successfully", uid, transactionMoveReq.FromAccountId, transactionMoveReq.ToAccountId)
	return true, nil
}

// TransactionBatchDeleteHandler deletes existed transactions by request parameters for current user
func (a *TransactionsApi) TransactionBatchDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionBatchDeleteReq models.TransactionBatchDeleteRequest
	err := c.ShouldBindJSON(&transactionBatchDeleteReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchDeleteHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	transactionIds, err := utils.StringArrayToInt64Array(transactionBatchDeleteReq.Ids)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionBatchDeleteHandler] parse transaction ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionIdInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionBatchDeleteHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	if !a.users.IsPasswordEqualsUserPassword(transactionBatchDeleteReq.Password, user) {
		return nil, errs.ErrUserPasswordWrong
	}

	transactions, err := a.transactions.GetTransactionsByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchDeleteHandler] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, transactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionBatchDeleteHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

		if !transactionEditable {
			log.Warnf(c, "[transactions.TransactionBatchDeleteHandler] transaction \"id:%d\" is not editable for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrCannotDeleteTransactionWithThisTransactionTime
		}
	}

	deletedCount := 0

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		err = a.transactions.DeleteTransaction(c, uid, transaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionBatchDeleteHandler] failed to delete transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		deletedCount++
	}

	log.Infof(c, "[transactions.TransactionBatchDeleteHandler] user \"uid:%d\" has deleted %d transactions", uid, deletedCount)
	return true, nil
}
