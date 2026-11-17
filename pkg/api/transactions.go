package api

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/duplicatechecker"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// TransactionsApi represents transaction api
type TransactionsApi struct {
	ApiUsingConfig
	ApiUsingDuplicateChecker
	transactions          *services.TransactionService
	transactionCategories *services.TransactionCategoryService
	transactionTags       *services.TransactionTagService
	transactionPictures   *services.TransactionPictureService
	accounts              *services.AccountService
	tokens                *services.TokenService
	users                 *services.UserService
}

// Initialize a transaction api singleton instance
var (
	Transactions = &TransactionsApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		ApiUsingDuplicateChecker: ApiUsingDuplicateChecker{
			ApiUsingConfig: ApiUsingConfig{
				container: settings.Container,
			},
			duplicateChecker: duplicatechecker.Container,
		},
		transactions:          services.Transactions,
		transactionCategories: services.TransactionCategories,
		transactionTags:       services.TransactionTags,
		transactionPictures:   services.TransactionPictures,
		accounts:              services.Accounts,
		tokens:                services.Tokens,
		users:                 services.Users,
	}
)

// TransactionCountHandler returns transaction total count of current user
func (a *TransactionsApi) TransactionCountHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionCountReq models.TransactionCountRequest
	err := c.ShouldBindQuery(&transactionCountReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCountHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	allAccountIds, err := a.accounts.GetAccountOrSubAccountIds(c, transactionCountReq.AccountIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCountHandler] get account error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allCategoryIds, err := a.transactionCategories.GetCategoryOrSubCategoryIds(c, transactionCountReq.CategoryIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCountHandler] get transaction category error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	noTags := transactionCountReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(transactionCountReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionCountHandler] parse transaction filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	totalCount, err := a.transactions.GetTransactionCount(c, uid, transactionCountReq.MaxTime, transactionCountReq.MinTime, transactionCountReq.Type, allCategoryIds, allAccountIds, tagFilters, noTags, transactionCountReq.AmountFilter, transactionCountReq.Keyword, transactionCountReq.MatchMode, transactionCountReq.MustHavePictures)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionCountHandler] failed to get transaction count for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	countResp := &models.TransactionCountResponse{
		TotalCount: totalCount,
	}

	return countResp, nil
}

// TransactionListHandler returns transaction list of current user
func (a *TransactionsApi) TransactionListHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionListReq models.TransactionListByMaxTimeRequest
	err := c.ShouldBindQuery(&transactionListReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionListHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	allAccountIds, err := a.accounts.GetAccountOrSubAccountIds(c, transactionListReq.AccountIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListHandler] get account error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allCategoryIds, err := a.transactionCategories.GetCategoryOrSubCategoryIds(c, transactionListReq.CategoryIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListHandler] get transaction category error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	noTags := transactionListReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(transactionListReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionListHandler] parse transaction tag filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	var totalCount int64

	if transactionListReq.WithCount {
		totalCount, err = a.transactions.GetTransactionCount(c, uid, transactionListReq.MaxTime, transactionListReq.MinTime, transactionListReq.Type, allCategoryIds, allAccountIds, tagFilters, noTags, transactionListReq.AmountFilter, transactionListReq.Keyword, transactionListReq.MatchMode, transactionListReq.MustHavePictures)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionListHandler] failed to get transaction count for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	transactions, err := a.transactions.GetTransactionsByMaxTime(c, uid, transactionListReq.MaxTime, transactionListReq.MinTime, transactionListReq.Type, allCategoryIds, allAccountIds, tagFilters, noTags, transactionListReq.AmountFilter, transactionListReq.Keyword, transactionListReq.MatchMode, transactionListReq.MustHavePictures, transactionListReq.Page, transactionListReq.Count, true, true)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListHandler] failed to get transactions earlier than \"%d\" for user \"uid:%d\", because %s", transactionListReq.MaxTime, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	hasMore := false
	var nextTimeSequenceId *int64

	if len(transactions) > int(transactionListReq.Count) {
		hasMore = true
		nextTimeSequenceId = &transactions[transactionListReq.Count].TransactionTime
		transactions = transactions[:transactionListReq.Count]
	}

	accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, err := a.getTransactionEssentialDataByTransactionIds(c, user, transactions, transactionListReq.WithPictures, transactionListReq.TrimCategory, transactionListReq.TrimTag)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListHandler] failed to get essential data for assembling transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactions = a.filterTransactions(c, uid, transactions, accountMap)
	transactionResult, err := a.getTransactionResponseListResult(c, user, transactions, accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, clientTimezone, transactionListReq.WithPictures, transactionListReq.TrimAccount, transactionListReq.TrimCategory, transactionListReq.TrimTag)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListHandler] failed to assemble transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionResps := &models.TransactionInfoPageWrapperResponse{
		Items: transactionResult,
	}

	if hasMore {
		transactionResps.NextTimeSequenceId = nextTimeSequenceId
	}

	if transactionListReq.WithCount {
		transactionResps.TotalCount = &totalCount
	}

	return transactionResps, nil
}

// TransactionMonthListHandler returns all transaction list of current user by month
func (a *TransactionsApi) TransactionMonthListHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionListReq models.TransactionListInMonthByPageRequest
	err := c.ShouldBindQuery(&transactionListReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMonthListHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMonthListHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionMonthListHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	allAccountIds, err := a.accounts.GetAccountOrSubAccountIds(c, transactionListReq.AccountIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMonthListHandler] get account error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allCategoryIds, err := a.transactionCategories.GetCategoryOrSubCategoryIds(c, transactionListReq.CategoryIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionMonthListHandler] get transaction category error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	noTags := transactionListReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(transactionListReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionMonthListHandler] parse transaction tag filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	transactions, err := a.transactions.GetTransactionsInMonthByPage(c, uid, transactionListReq.Year, transactionListReq.Month, transactionListReq.Type, allCategoryIds, allAccountIds, tagFilters, noTags, transactionListReq.AmountFilter, transactionListReq.Keyword, transactionListReq.MatchMode, transactionListReq.MustHavePictures)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMonthListHandler] failed to get transactions in month \"%d-%d\" for user \"uid:%d\", because %s", transactionListReq.Year, transactionListReq.Month, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, err := a.getTransactionEssentialDataByTransactionIds(c, user, transactions, transactionListReq.WithPictures, transactionListReq.TrimCategory, transactionListReq.TrimTag)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMonthListHandler] failed to get essential data for assembling transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactions = a.filterTransactions(c, uid, transactions, accountMap)
	transactionResult, err := a.getTransactionResponseListResult(c, user, transactions, accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, clientTimezone, transactionListReq.WithPictures, transactionListReq.TrimAccount, transactionListReq.TrimCategory, transactionListReq.TrimTag)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionMonthListHandler] failed to assemble transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionResps := &models.TransactionInfoPageWrapperResponse2{
		Items:      transactionResult,
		TotalCount: int64(transactionResult.Len()),
	}

	return transactionResps, nil
}

// TransactionListAllHandler returns all transaction list of current user
func (a *TransactionsApi) TransactionListAllHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionAllListReq models.TransactionAllListRequest
	err := c.ShouldBindQuery(&transactionAllListReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListAllHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListAllHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionListAllHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	allAccountIds, err := a.accounts.GetAccountOrSubAccountIds(c, transactionAllListReq.AccountIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListAllHandler] get account error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allCategoryIds, err := a.transactionCategories.GetCategoryOrSubCategoryIds(c, transactionAllListReq.CategoryIds, uid)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionListAllHandler] get transaction category error, because %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	noTags := transactionAllListReq.TagFilter == models.TransactionNoTagFilterValue
	var tagFilters []*models.TransactionTagFilter

	if !noTags {
		tagFilters, err = models.ParseTransactionTagFilter(transactionAllListReq.TagFilter)

		if err != nil {
			log.Warnf(c, "[transactions.TransactionListAllHandler] parse transaction tag filters error, because %s", err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	maxTransactionTime := int64(math.MaxInt64)
	minTransactionTime := int64(0)

	if transactionAllListReq.EndTime > 0 {
		maxTransactionTime = utils.GetMaxTransactionTimeFromUnixTime(transactionAllListReq.EndTime)
	}

	if transactionAllListReq.StartTime > 0 {
		minTransactionTime = utils.GetMinTransactionTimeFromUnixTime(transactionAllListReq.StartTime)
	}

	allTransactions, err := a.transactions.GetAllSpecifiedTransactions(c, uid, maxTransactionTime, minTransactionTime, transactionAllListReq.Type, allCategoryIds, allAccountIds, tagFilters, noTags, transactionAllListReq.AmountFilter, transactionAllListReq.Keyword, transactionAllListReq.MatchMode, transactionAllListReq.MustHavePictures, pageCountForDataExport, true)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListAllHandler] failed to get all transactions for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	var accountMap map[int64]*models.Account
	var categoryMap map[int64]*models.TransactionCategory
	var tagMap map[int64]*models.TransactionTag
	var allTransactionTagIds map[int64][]int64
	var pictureInfoMap map[int64][]*models.TransactionPictureInfo

	if minTransactionTime == 0 && maxTransactionTime == math.MaxInt64 && len(allCategoryIds) < 1 && len(allAccountIds) < 1 && len(tagFilters) < 1 && transactionAllListReq.AmountFilter == "" && transactionAllListReq.Keyword == "" {
		accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, err = a.getTransactionAllEssentialData(c, user, transactionAllListReq.WithPictures, transactionAllListReq.TrimCategory, transactionAllListReq.TrimTag)
	} else {
		accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, err = a.getTransactionEssentialDataByTransactionIds(c, user, allTransactions, transactionAllListReq.WithPictures, transactionAllListReq.TrimCategory, transactionAllListReq.TrimTag)
	}

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListAllHandler] failed to get essential data for assembling transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	allTransactions = a.filterTransactions(c, uid, allTransactions, accountMap)
	transactionResult, err := a.getTransactionResponseListResult(c, user, allTransactions, accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, clientTimezone, transactionAllListReq.WithPictures, transactionAllListReq.TrimAccount, transactionAllListReq.TrimCategory, transactionAllListReq.TrimTag)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionListAllHandler] failed to assemble transaction result for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return transactionResult, nil
}

// TransactionGetHandler returns one specific transaction of current user
func (a *TransactionsApi) TransactionGetHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionGetReq models.TransactionGetRequest
	err := c.ShouldBindQuery(&transactionGetReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionGetHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionGetHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionGetHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transaction, err := a.transactions.GetTransactionByTransactionId(c, uid, transactionGetReq.Id)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionGetReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		transaction = a.transactions.GetRelatedTransferTransaction(transaction)
	}

	accountIds := make([]int64, 0, 2)
	accountIds = append(accountIds, transaction.AccountId)

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		accountIds = append(accountIds, transaction.RelatedAccountId)
		accountIds = utils.ToUniqueInt64Slice(accountIds)
	}

	accountMap, err := a.accounts.GetAccountsByAccountIds(c, uid, accountIds)

	if _, exists := accountMap[transaction.AccountId]; !exists {
		log.Warnf(c, "[transactions.TransactionGetHandler] account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
		return nil, errs.ErrTransactionNotFound
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if _, exists := accountMap[transaction.RelatedAccountId]; !exists {
			log.Warnf(c, "[transactions.TransactionGetHandler] related account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrTransactionNotFound
		}
	}

	allTransactionTagIds, err := a.transactionTags.GetAllTagIdsOfTransactions(c, uid, []int64{transaction.TransactionId})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	var category *models.TransactionCategory
	var tagMap map[int64]*models.TransactionTag
	var pictureInfos []*models.TransactionPictureInfo

	if !transactionGetReq.TrimCategory && transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		category, err = a.transactionCategories.GetCategoryByCategoryId(c, uid, transaction.CategoryId)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transactions category for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	if !transactionGetReq.TrimTag {
		tagMap, err = a.transactionTags.GetTagsByTagIds(c, uid, utils.ToUniqueInt64Slice(a.transactionTags.GetTransactionTagIds(allTransactionTagIds)))

		if err != nil {
			log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transactions tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	if transactionGetReq.WithPictures && a.CurrentConfig().EnableTransactionPictures {
		pictureInfos, err = a.transactionPictures.GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transactions pictures for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	transactionEditable := transaction.IsEditable(user, clientTimezone, accountMap[transaction.AccountId], accountMap[transaction.RelatedAccountId])
	transactionTagIds := allTransactionTagIds[transaction.TransactionId]
	transactionResp := transaction.ToTransactionInfoResponse(transactionTagIds, transactionEditable)

	if transactionResp == nil {
		log.Warnf(c, "[transactions.TransactionGetHandler] failed to assemble transaction response for transaction \"id:%d\" of user \"uid:%d\"", transaction.TransactionId, uid)
		return nil, errs.ErrSystemError
	}

	if !transactionGetReq.TrimAccount {
		if sourceAccount := accountMap[transaction.AccountId]; sourceAccount != nil {
			transactionResp.SourceAccount = sourceAccount.ToAccountInfoResponse()
		}

		if destinationAccount := accountMap[transaction.RelatedAccountId]; destinationAccount != nil {
			transactionResp.DestinationAccount = destinationAccount.ToAccountInfoResponse()
		}
	}

	if !transactionGetReq.TrimCategory {
		if category != nil {
			transactionResp.Category = category.ToTransactionCategoryInfoResponse()
		}
	}

	if !transactionGetReq.TrimTag {
		transactionResp.Tags = a.getTransactionTagInfoResponses(transactionTagIds, tagMap)
	}

	if transactionGetReq.WithPictures && a.CurrentConfig().EnableTransactionPictures && len(pictureInfos) > 0 {
		pictureToken, err := a.GetTransactionPictureToken(c, uid)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionGetHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrTokenGenerating)
		}

		transactionResp.Pictures = a.GetTransactionPictureInfoResponseList(pictureInfos, pictureToken)
	}

	return transactionResp, nil
}

// TransactionCreateHandler saves a new transaction by request parameters for current user
func (a *TransactionsApi) TransactionCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionCreateReq models.TransactionCreateRequest
	err := c.ShouldBindJSON(&transactionCreateReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	tagIds, err := utils.StringArrayToInt64Array(transactionCreateReq.TagIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateHandler] parse tag ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionTagIdInvalid
	}

	if len(tagIds) > models.MaximumTagsCountOfTransaction {
		return nil, errs.ErrTransactionHasTooManyTags
	}

	pictureIds, err := utils.StringArrayToInt64Array(transactionCreateReq.PictureIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionCreateHandler] parse picture ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionPictureIdInvalid
	}

	if len(pictureIds) > models.MaximumPicturesCountOfTransaction {
		return nil, errs.ErrTransactionHasTooManyPictures
	}

	if transactionCreateReq.Type < models.TRANSACTION_TYPE_MODIFY_BALANCE || transactionCreateReq.Type > models.TRANSACTION_TYPE_TRANSFER {
		log.Warnf(c, "[transactions.TransactionCreateHandler] transaction type is invalid")
		return nil, errs.ErrTransactionTypeInvalid
	}

	if transactionCreateReq.Type == models.TRANSACTION_TYPE_MODIFY_BALANCE && transactionCreateReq.CategoryId != 0 {
		log.Warnf(c, "[transactions.TransactionCreateHandler] balance modification transaction cannot set category id")
		return nil, errs.ErrBalanceModificationTransactionCannotSetCategory
	}

	if transactionCreateReq.Type != models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.DestinationAccountId != 0 {
		log.Warnf(c, "[transactions.TransactionCreateHandler] non-transfer transaction destination account cannot be set")
		return nil, errs.ErrTransactionDestinationAccountCannotBeSet
	} else if transactionCreateReq.Type == models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.SourceAccountId == transactionCreateReq.DestinationAccountId {
		log.Warnf(c, "[transactions.TransactionCreateHandler] transfer transaction source account must not be destination account")
		return nil, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
	}

	if transactionCreateReq.Type != models.TRANSACTION_TYPE_TRANSFER && transactionCreateReq.DestinationAmount != 0 {
		log.Warnf(c, "[transactions.TransactionCreateHandler] non-transfer transaction destination amount cannot be set")
		return nil, errs.ErrTransactionDestinationAmountCannotBeSet
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transaction := a.createNewTransactionModel(uid, &transactionCreateReq, c.ClientIP())

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, []*models.Transaction{transaction})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

	if !transactionEditable {
		return nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
	}

	var pictureInfos []*models.TransactionPictureInfo

	if len(pictureIds) > 0 {
		pictureInfos, err = a.transactionPictures.GetNewPictureInfosByPictureIds(c, uid, pictureIds)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get transactions pictures for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		notExistsPictureIds := utils.Int64SliceMinus(pictureIds, a.transactionPictures.GetTransactionPictureIds(pictureInfos))

		if len(notExistsPictureIds) > 0 {
			log.Errorf(c, "[transactions.TransactionCreateHandler] some pictures \"ids:%s\" does not exists for user \"uid:%d\"", strings.Join(utils.Int64ArrayToStringArray(notExistsPictureIds), ","), uid)
			return nil, errs.ErrTransactionPictureNotFound
		}
	}

	if a.CurrentConfig().EnableDuplicateSubmissionsCheck && transactionCreateReq.ClientSessionId != "" {
		found, remark := a.GetSubmissionRemark(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_TRANSACTION, uid, transactionCreateReq.ClientSessionId)

		if found {
			log.Infof(c, "[transactions.TransactionCreateHandler] another transaction \"id:%s\" has been created for user \"uid:%d\"", remark, uid)
			transactionId, err := utils.StringToInt64(remark)

			if err == nil {
				transaction, err = a.transactions.GetTransactionByTransactionId(c, uid, transactionId)

				if err != nil {
					log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get existed transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())
					return nil, errs.Or(err, errs.ErrOperationFailed)
				}

				transactionResp := transaction.ToTransactionInfoResponse(tagIds, transactionEditable)

				if len(pictureInfos) > 0 {
					pictureToken, err := a.GetTransactionPictureToken(c, uid)

					if err != nil {
						log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
						return nil, errs.Or(err, errs.ErrTokenGenerating)
					}

					transactionResp.Pictures = a.GetTransactionPictureInfoResponseList(pictureInfos, pictureToken)
				}

				return transactionResp, nil
			}
		}
	}

	err = a.transactions.CreateTransaction(c, transaction, tagIds, pictureIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionCreateHandler] failed to create transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionCreateHandler] user \"uid:%d\" has created a new transaction \"id:%d\" successfully", uid, transaction.TransactionId)

	a.SetSubmissionRemarkIfEnable(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_TRANSACTION, uid, transactionCreateReq.ClientSessionId, utils.Int64ToString(transaction.TransactionId))
	transactionResp := transaction.ToTransactionInfoResponse(tagIds, transactionEditable)

	if len(pictureInfos) > 0 {
		pictureToken, err := a.GetTransactionPictureToken(c, uid)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionCreateHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrTokenGenerating)
		}

		transactionResp.Pictures = a.GetTransactionPictureInfoResponseList(pictureInfos, pictureToken)
	}

	return transactionResp, nil
}

// TransactionModifyHandler saves an existed transaction by request parameters for current user
func (a *TransactionsApi) TransactionModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionModifyReq models.TransactionModifyRequest
	err := c.ShouldBindJSON(&transactionModifyReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionModifyHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	tagIds, err := utils.StringArrayToInt64Array(transactionModifyReq.TagIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionModifyHandler] parse tag ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionTagIdInvalid
	}

	if len(tagIds) > models.MaximumTagsCountOfTransaction {
		return nil, errs.ErrTransactionHasTooManyTags
	}

	pictureIds, err := utils.StringArrayToInt64Array(transactionModifyReq.PictureIds)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionModifyHandler] parse picture ids failed, because %s", err.Error())
		return nil, errs.ErrTransactionPictureIdInvalid
	}

	if len(pictureIds) > models.MaximumPicturesCountOfTransaction {
		return nil, errs.ErrTransactionHasTooManyPictures
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transaction, err := a.transactions.GetTransactionByTransactionId(c, uid, transactionModifyReq.Id)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		log.Warnf(c, "[transactions.TransactionModifyHandler] cannot modify transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transactionModifyReq.Id, uid)
		return nil, errs.ErrTransactionTypeInvalid
	}

	newTransactionType, err := transactionModifyReq.Type.ToTransactionDbType()

	if err != nil {
		return nil, errs.ErrTransactionTypeInvalid
	}

	if (transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && newTransactionType != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE) ||
		(transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && newTransactionType == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE) {
		log.Warnf(c, "[transactions.TransactionModifyHandler] cannot modify transaction type from \"%d\" to \"%d\"", transaction.Type, newTransactionType)
		return nil, errs.ErrTransactionTypeInvalid
	}

	changeToTransfer := newTransactionType == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT

	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && transactionModifyReq.CategoryId != 0 {
		log.Warnf(c, "[transactions.TransactionModifyHandler] balance modification transaction cannot set category id")
		return nil, errs.ErrBalanceModificationTransactionCannotSetCategory
	} else if transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && transactionModifyReq.CategoryId == 0 {
		log.Warnf(c, "[transactions.TransactionModifyHandler] non-balance modification transaction must set category id")
		return nil, errs.ErrIncompleteOrIncorrectSubmission
	}

	allTransactionTagIds, err := a.transactionTags.GetAllTagIdsOfTransactions(c, uid, []int64{transaction.TransactionId})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionTagIds := allTransactionTagIds[transaction.TransactionId]

	if transactionTagIds == nil {
		transactionTagIds = make([]int64, 0, 0)
	}

	transactionPictureInfos, err := a.transactionPictures.GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transaction picture infos for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionPictureIds := a.transactionPictures.GetTransactionPictureIds(transactionPictureInfos)

	newTransaction := &models.Transaction{
		TransactionId:     transaction.TransactionId,
		Uid:               uid,
		Type:              newTransactionType,
		CategoryId:        transactionModifyReq.CategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(transactionModifyReq.Time),
		TimezoneUtcOffset: transactionModifyReq.UtcOffset,
		AccountId:         transactionModifyReq.SourceAccountId,
		Amount:            transactionModifyReq.SourceAmount,
		HideAmount:        transactionModifyReq.HideAmount,
		Comment:           transactionModifyReq.Comment,
	}

	if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		newTransaction.RelatedAccountId = transactionModifyReq.DestinationAccountId
		newTransaction.RelatedAccountAmount = transactionModifyReq.DestinationAmount
	}

	if transactionModifyReq.GeoLocation != nil {
		newTransaction.GeoLongitude = transactionModifyReq.GeoLocation.Longitude
		newTransaction.GeoLatitude = transactionModifyReq.GeoLocation.Latitude
	}

	if newTransaction.Type == transaction.Type &&
		newTransaction.CategoryId == transaction.CategoryId &&
		utils.GetUnixTimeFromTransactionTime(newTransaction.TransactionTime) == utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) &&
		newTransaction.TimezoneUtcOffset == transaction.TimezoneUtcOffset &&
		newTransaction.AccountId == transaction.AccountId &&
		newTransaction.Amount == transaction.Amount &&
		(newTransaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT || newTransaction.RelatedAccountId == transaction.RelatedAccountId) &&
		(newTransaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT || newTransaction.RelatedAccountAmount == transaction.RelatedAccountAmount) &&
		newTransaction.HideAmount == transaction.HideAmount &&
		newTransaction.Comment == transaction.Comment &&
		newTransaction.GeoLongitude == transaction.GeoLongitude &&
		newTransaction.GeoLatitude == transaction.GeoLatitude &&
		utils.Int64SliceEquals(tagIds, transactionTagIds) &&
		utils.Int64SliceEquals(pictureIds, transactionPictureIds) {
		return nil, errs.ErrNothingWillBeUpdated
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, []*models.Transaction{transaction, newTransaction})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])
	newTransactionEditable := user.CanEditTransactionByTransactionTime(newTransaction.TransactionTime, clientTimezone, allUsedAccounts[newTransaction.AccountId], allUsedAccounts[newTransaction.RelatedAccountId])

	if !transactionEditable || !newTransactionEditable {
		return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
	}

	var addTransactionTagIds []int64
	var removeTransactionTagIds []int64

	if !utils.Int64SliceEquals(tagIds, transactionTagIds) {
		removeTransactionTagIds = transactionTagIds
		addTransactionTagIds = tagIds
	}

	addTransactionPictureIds := utils.Int64SliceMinus(pictureIds, transactionPictureIds)
	removeTransactionPictureIds := utils.Int64SliceMinus(transactionPictureIds, pictureIds)
	var newPictureInfos []*models.TransactionPictureInfo

	if !utils.Int64SliceEquals(pictureIds, transactionPictureIds) {
		oldAndNewPictureIds := transactionPictureIds
		oldAndNewPictureInfoMap := a.transactionPictures.GetPictureInfoMapByList(transactionPictureInfos)

		if len(addTransactionPictureIds) > 0 {
			addPictureInfos, err := a.transactionPictures.GetNewPictureInfosByPictureIds(c, uid, addTransactionPictureIds)

			if err != nil {
				log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transactions pictures for user \"uid:%d\", because %s", uid, err.Error())
				return nil, errs.Or(err, errs.ErrOperationFailed)
			}

			oldAndNewPictureIds = append(oldAndNewPictureIds, a.transactionPictures.GetTransactionPictureIds(addPictureInfos)...)
			notExistsPictureIds := utils.Int64SliceMinus(pictureIds, oldAndNewPictureIds)

			if len(notExistsPictureIds) > 0 {
				log.Errorf(c, "[transactions.TransactionModifyHandler] some pictures \"ids:%s\" does not exists for user \"uid:%d\"", strings.Join(utils.Int64ArrayToStringArray(notExistsPictureIds), ","), uid)
				return nil, errs.ErrTransactionPictureNotFound
			}

			for i := 0; i < len(addPictureInfos); i++ {
				oldAndNewPictureInfoMap[addPictureInfos[i].PictureId] = addPictureInfos[i]
			}
		}

		for i := 0; i < len(pictureIds); i++ {
			pictureId := pictureIds[i]
			pictureInfo, exists := oldAndNewPictureInfoMap[pictureId]

			if exists {
				newPictureInfos = append(newPictureInfos, pictureInfo)
			}
		}
	}

	err = a.transactions.ModifyTransaction(c, newTransaction, changeToTransfer, len(transactionTagIds), addTransactionTagIds, removeTransactionTagIds, addTransactionPictureIds, removeTransactionPictureIds)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionModifyHandler] failed to update transaction \"id:%d\" for user \"uid:%d\", because %s", transactionModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionModifyHandler] user \"uid:%d\" has updated transaction \"id:%d\" successfully", uid, transactionModifyReq.Id)

	newTransactionResp := newTransaction.ToTransactionInfoResponse(tagIds, transactionEditable)

	if len(newPictureInfos) > 0 {
		pictureToken, err := a.GetTransactionPictureToken(c, uid)

		if err != nil {
			log.Errorf(c, "[transactions.TransactionModifyHandler] failed to get transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrTokenGenerating)
		}

		newTransactionResp.Pictures = a.GetTransactionPictureInfoResponseList(newPictureInfos, pictureToken)
	}

	return newTransactionResp, nil
}

// TransactionDeleteHandler deletes an existed transaction by request parameters for current user
func (a *TransactionsApi) TransactionDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionDeleteReq models.TransactionDeleteRequest
	err := c.ShouldBindJSON(&transactionDeleteReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionDeleteHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionDeleteHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transaction, err := a.transactions.GetTransactionByTransactionId(c, uid, transactionDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionDeleteHandler] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionDeleteReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		log.Warnf(c, "[transactions.TransactionDeleteHandler] cannot delete transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transactionDeleteReq.Id, uid)
		return nil, errs.ErrTransactionTypeInvalid
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, []*models.Transaction{transaction})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionDeleteHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[transaction.RelatedAccountId])

	if !transactionEditable {
		return nil, errs.ErrCannotDeleteTransactionWithThisTransactionTime
	}

	err = a.transactions.DeleteTransaction(c, uid, transactionDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionDeleteHandler] failed to delete transaction \"id:%d\" for user \"uid:%d\", because %s", transactionDeleteReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionDeleteHandler] user \"uid:%d\" has deleted transaction \"id:%d\"", uid, transactionDeleteReq.Id)
	return true, nil
}

func (a *TransactionsApi) filterTransactions(c *core.WebContext, uid int64, transactions []*models.Transaction, accountMap map[int64]*models.Account) []*models.Transaction {
	finalTransactions := make([]*models.Transaction, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if _, exists := accountMap[transaction.AccountId]; !exists {
			log.Warnf(c, "[transactions.filterTransactions] account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
			continue
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			if _, exists := accountMap[transaction.RelatedAccountId]; !exists {
				log.Warnf(c, "[transactions.filterTransactions] related account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
				continue
			}
		}

		finalTransactions = append(finalTransactions, transaction)
	}

	return finalTransactions
}

func (a *TransactionsApi) getTransactionTagInfoResponses(tagIds []int64, allTransactionTags map[int64]*models.TransactionTag) []*models.TransactionTagInfoResponse {
	allTags := make([]*models.TransactionTagInfoResponse, 0, len(tagIds))

	for i := 0; i < len(tagIds); i++ {
		tag := allTransactionTags[tagIds[i]]

		if tag == nil {
			continue
		}

		allTags = append(allTags, tag.ToTransactionTagInfoResponse())
	}

	return allTags
}

func (a *TransactionsApi) getTransactionAllEssentialData(c *core.WebContext, user *models.User, withPictures bool, trimCategory bool, trimTag bool) (accountMap map[int64]*models.Account, categoryMap map[int64]*models.TransactionCategory, tagMap map[int64]*models.TransactionTag, allTransactionTagIds map[int64][]int64, pictureInfoMap map[int64][]*models.TransactionPictureInfo, err error) {
	uid := user.Uid
	allAccounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[transactions.getTransactionAllEssentialData] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, nil, nil, nil, err
	}

	accountMap = a.accounts.GetAccountMapByList(allAccounts)

	allTagIndexes, err := a.transactionTags.GetAllTagIdsOfAllTransactions(c, uid)

	if err != nil {
		log.Errorf(c, "[transactions.getTransactionAllEssentialData] failed to get all transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, nil, nil, nil, err
	}

	allTransactionTagIds = a.transactionTags.GetGroupedTransactionTagIds(allTagIndexes)

	if !trimCategory {
		allCategories, err := a.transactionCategories.GetAllCategoriesByUid(c, uid, 0, -1)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionAllEssentialData] failed to get all transactions categories for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}

		categoryMap = a.transactionCategories.GetCategoryMapByList(allCategories)
	}

	if !trimTag {
		allTags, err := a.transactionTags.GetAllTagsByUid(c, uid)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionAllEssentialData] failed to get all transactions tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}

		tagMap = a.transactionTags.GetTagMapByList(allTags)
	}

	if withPictures && a.CurrentConfig().EnableTransactionPictures {
		pictureInfoMap, err = a.transactionPictures.GetAllPictureInfosOfAllTransactions(c, uid)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionAllEssentialData] failed to get all transactions pictures for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}
	}

	return accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, nil
}

func (a *TransactionsApi) getTransactionEssentialDataByTransactionIds(c *core.WebContext, user *models.User, transactions []*models.Transaction, withPictures bool, trimCategory bool, trimTag bool) (accountMap map[int64]*models.Account, categoryMap map[int64]*models.TransactionCategory, tagMap map[int64]*models.TransactionTag, allTransactionTagIds map[int64][]int64, pictureInfoMap map[int64][]*models.TransactionPictureInfo, err error) {
	uid := user.Uid
	transactionIds := make([]int64, len(transactions))
	accountIds := make([]int64, 0, len(transactions)*2)
	categoryIds := make([]int64, 0, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transactionId := transactions[i].TransactionId

		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			transactionId = transactions[i].RelatedId
		}

		transactionIds[i] = transactionId
		accountIds = append(accountIds, transactions[i].AccountId)

		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			accountIds = append(accountIds, transactions[i].RelatedAccountId)
		}

		categoryIds = append(categoryIds, transactions[i].CategoryId)
	}

	transactionIds = utils.ToUniqueInt64Slice(transactionIds)
	accountIds = utils.ToUniqueInt64Slice(accountIds)
	categoryIds = utils.ToUniqueInt64Slice(categoryIds)

	accountMap, err = a.accounts.GetAccountsByAccountIds(c, uid, accountIds)

	if err != nil {
		log.Errorf(c, "[transactions.getTransactionEssentialDataByTransactionIds] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, nil, nil, nil, err
	}

	allTransactionTagIds, err = a.transactionTags.GetAllTagIdsOfTransactions(c, uid, transactionIds)

	if err != nil {
		log.Errorf(c, "[transactions.getTransactionEssentialDataByTransactionIds] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, nil, nil, nil, err
	}

	if !trimCategory {
		categoryMap, err = a.transactionCategories.GetCategoriesByCategoryIds(c, uid, categoryIds)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionEssentialDataByTransactionIds] failed to get transactions categories for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}
	}

	if !trimTag {
		tagMap, err = a.transactionTags.GetTagsByTagIds(c, uid, utils.ToUniqueInt64Slice(a.transactionTags.GetTransactionTagIds(allTransactionTagIds)))

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionEssentialDataByTransactionIds] failed to get transactions tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}
	}

	if withPictures && a.CurrentConfig().EnableTransactionPictures {
		pictureInfoMap, err = a.transactionPictures.GetPictureInfosByTransactionIds(c, uid, transactionIds)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionEssentialDataByTransactionIds] failed to get transactions pictures for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, nil, nil, nil, err
		}
	}

	return accountMap, categoryMap, tagMap, allTransactionTagIds, pictureInfoMap, nil
}

func (a *TransactionsApi) getTransactionUsedAccounts(c *core.WebContext, uid int64, transactions []*models.Transaction) (map[int64]*models.Account, error) {
	accountIds := make([]int64, 0, len(transactions)*2)

	for i := 0; i < len(transactions); i++ {
		accountIds = append(accountIds, transactions[i].AccountId)

		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			accountIds = append(accountIds, transactions[i].RelatedAccountId)
		}
	}

	accountMap, err := a.accounts.GetAccountsByAccountIds(c, uid, utils.ToUniqueInt64Slice(accountIds))

	if err != nil {
		log.Errorf(c, "[transactions.getTransactionUsedAccounts] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(transactions); i++ {
		if _, exists := accountMap[transactions[i].AccountId]; !exists {
			log.Warnf(c, "[transactions.getTransactionUsedAccounts] account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transactions[i].TransactionId, uid)
			return nil, errs.ErrSourceAccountNotFound
		}

		if transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN || transactions[i].Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			if _, exists := accountMap[transactions[i].RelatedAccountId]; !exists {
				log.Warnf(c, "[transactions.getTransactionUsedAccounts] related account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transactions[i].TransactionId, uid)
				return nil, errs.ErrDestinationAccountNotFound
			}
		}
	}

	return accountMap, nil
}

func (a *TransactionsApi) getTransactionResponseListResult(c *core.WebContext, user *models.User, transactions []*models.Transaction, allAccounts map[int64]*models.Account, categoryMap map[int64]*models.TransactionCategory, tagMap map[int64]*models.TransactionTag, allTransactionTagIds map[int64][]int64, pictureInfoMap map[int64][]*models.TransactionPictureInfo, clientTimezone *time.Location, withPictures bool, trimAccount bool, trimCategory bool, trimTag bool) (models.TransactionInfoResponseSlice, error) {
	var err error
	pictureToken := ""

	if withPictures && a.CurrentConfig().EnableTransactionPictures && len(pictureInfoMap) > 0 {
		pictureToken, err = a.GetTransactionPictureToken(c, user.Uid)

		if err != nil {
			log.Errorf(c, "[transactions.getTransactionResponseListResult] failed to get transaction picture token for user \"uid:%d\", because %s", user.Uid, err.Error())
			return nil, errs.Or(err, errs.ErrTokenGenerating)
		}
	}

	result := make(models.TransactionInfoResponseSlice, len(transactions))

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			transaction = a.transactions.GetRelatedTransferTransaction(transaction)
		}

		transactionEditable := transaction.IsEditable(user, clientTimezone, allAccounts[transaction.AccountId], allAccounts[transaction.RelatedAccountId])
		transactionTagIds := allTransactionTagIds[transaction.TransactionId]
		result[i] = transaction.ToTransactionInfoResponse(transactionTagIds, transactionEditable)

		if !trimAccount {
			if sourceAccount := allAccounts[transaction.AccountId]; sourceAccount != nil {
				result[i].SourceAccount = sourceAccount.ToAccountInfoResponse()
			}

			if destinationAccount := allAccounts[transaction.RelatedAccountId]; destinationAccount != nil {
				result[i].DestinationAccount = destinationAccount.ToAccountInfoResponse()
			}
		}

		if !trimCategory && categoryMap != nil {
			if category := categoryMap[transaction.CategoryId]; category != nil {
				result[i].Category = category.ToTransactionCategoryInfoResponse()
			}
		}

		if !trimTag && tagMap != nil && transactionTagIds != nil {
			result[i].Tags = a.getTransactionTagInfoResponses(transactionTagIds, tagMap)
		}

		if withPictures && a.CurrentConfig().EnableTransactionPictures && pictureInfoMap != nil {
			pictureInfos, exists := pictureInfoMap[transaction.TransactionId]

			if exists {
				result[i].Pictures = a.GetTransactionPictureInfoResponseList(pictureInfos, pictureToken)
			}
		}
	}

	sort.Sort(result)

	return result, nil
}

func (a *TransactionsApi) GetTransactionPictureToken(c *core.WebContext, uid int64) (string, error) {
	claims := c.GetTokenClaims()

	if claims == nil {
		log.Warnf(c, "[transactions.GetTransactionPictureToken] token is invalid, because claims is nil or token type is not normal")
		return "", errs.ErrInvalidToken
	}

	if claims.Type != core.USER_TOKEN_TYPE_NORMAL {
		return "", nil
	}

	pictureToken, err := a.tokens.DeriveTransactionPictureToken(c, claims, c.GetTokenRecord())

	if err != nil {
		log.Errorf(c, "[transactions.GetTransactionPictureToken] failed to derive transaction picture token for user \"uid:%d\", because %s", uid, err.Error())
		return "", errs.Or(err, errs.ErrTokenGenerating)
	}

	return pictureToken, nil
}

func (a *TransactionsApi) createNewTransactionModel(uid int64, transactionCreateReq *models.TransactionCreateRequest, clientIp string) *models.Transaction {
	var transactionDbType models.TransactionDbType

	if transactionCreateReq.Type == models.TRANSACTION_TYPE_MODIFY_BALANCE {
		transactionDbType = models.TRANSACTION_DB_TYPE_MODIFY_BALANCE
	} else if transactionCreateReq.Type == models.TRANSACTION_TYPE_EXPENSE {
		transactionDbType = models.TRANSACTION_DB_TYPE_EXPENSE
	} else if transactionCreateReq.Type == models.TRANSACTION_TYPE_INCOME {
		transactionDbType = models.TRANSACTION_DB_TYPE_INCOME
	} else if transactionCreateReq.Type == models.TRANSACTION_TYPE_TRANSFER {
		transactionDbType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	}

	transaction := &models.Transaction{
		Uid:               uid,
		Type:              transactionDbType,
		CategoryId:        transactionCreateReq.CategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(transactionCreateReq.Time),
		TimezoneUtcOffset: transactionCreateReq.UtcOffset,
		AccountId:         transactionCreateReq.SourceAccountId,
		Amount:            transactionCreateReq.SourceAmount,
		HideAmount:        transactionCreateReq.HideAmount,
		Comment:           transactionCreateReq.Comment,
		CreatedIp:         clientIp,
	}

	if transactionCreateReq.Type == models.TRANSACTION_TYPE_TRANSFER {
		transaction.RelatedAccountId = transactionCreateReq.DestinationAccountId
		transaction.RelatedAccountAmount = transactionCreateReq.DestinationAmount
	}

	if transactionCreateReq.GeoLocation != nil {
		transaction.GeoLongitude = transactionCreateReq.GeoLocation.Longitude
		transaction.GeoLatitude = transactionCreateReq.GeoLocation.Latitude
	}

	return transaction
}
