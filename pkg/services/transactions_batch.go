package services

import (
	"math"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// BatchCreateTransactions saves new transactions to database
func (s *TransactionService) BatchCreateTransactions(c core.Context, uid int64, transactions []*models.Transaction, allTagIds map[int][]int64, processHandler core.TaskProcessUpdateHandler) error {
	now := time.Now().Unix()
	currentProcess := float64(0)
	processUpdateStep := int(math.Max(100.0, float64(len(transactions)/100.0)))

	needTransactionUuidCount := uint16(0)
	needTagIndexUuidCount := uint16(0)

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Uid != uid {
			return errs.ErrUserIdInvalid
		}

		err := s.isAccountIdValid(transaction)

		if err != nil {
			return err
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			needTransactionUuidCount += 2
		} else {
			needTransactionUuidCount++
		}

		transaction.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))

		transaction.CreatedUnixTime = now
		transaction.UpdatedUnixTime = now
	}

	for index, tagIds := range allTagIds {
		if index < 0 || index >= len(transactions) {
			return errs.ErrOperationFailed
		}

		uniqueTagIds := utils.ToUniqueInt64Slice(tagIds)
		needTagIndexUuidCount += uint16(len(uniqueTagIds))
	}

	if needTransactionUuidCount > uint16(65535) || needTagIndexUuidCount > uint16(65535) {
		return errs.ErrImportTooManyTransaction
	}

	transactionUuids := s.GenerateUuids(uuid.UUID_TYPE_TRANSACTION, needTransactionUuidCount)
	transactionUuidIndex := 0

	if len(transactionUuids) < int(needTransactionUuidCount) {
		return errs.ErrSystemIsBusy
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		transaction.TransactionId = transactionUuids[transactionUuidIndex]
		transactionUuidIndex++

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			transaction.RelatedId = transactionUuids[transactionUuidIndex]
			transactionUuidIndex++
		}
	}

	tagIndexUuids := s.GenerateUuids(uuid.UUID_TYPE_TAG_INDEX, needTagIndexUuidCount)
	tagIndexUuidIndex := 0

	if len(tagIndexUuids) < int(needTagIndexUuidCount) {
		return errs.ErrSystemIsBusy
	}

	allTransactionTagIndexes := make(map[int64][]*models.TransactionTagIndex)
	allTransactionTagIds := make(map[int64][]int64)

	for index, tagIds := range allTagIds {
		transaction := transactions[index]
		uniqueTagIds := utils.ToUniqueInt64Slice(tagIds)

		transactionTagIndexes := make([]*models.TransactionTagIndex, len(uniqueTagIds))

		for i := 0; i < len(uniqueTagIds); i++ {
			transactionTagIndexes[i] = &models.TransactionTagIndex{
				TagIndexId:      tagIndexUuids[tagIndexUuidIndex],
				Uid:             transaction.Uid,
				Deleted:         false,
				TagId:           uniqueTagIds[i],
				TransactionId:   transaction.TransactionId,
				CreatedUnixTime: now,
				UpdatedUnixTime: now,
			}

			tagIndexUuidIndex++
		}

		allTransactionTagIndexes[transaction.TransactionId] = transactionTagIndexes
		allTransactionTagIds[transaction.TransactionId] = uniqueTagIds
	}

	userDataDb := s.UserDataDB(uid)

	return userDataDb.DoTransaction(c, func(sess datastore.DatabaseSession) error {
		for i := 0; i < len(transactions); i++ {
			transaction := transactions[i]
			transactionTagIndexes := allTransactionTagIndexes[transaction.TransactionId]
			transactionTagIds := allTransactionTagIds[transaction.TransactionId]
			err := s.doCreateTransaction(c, userDataDb, sess, transaction, transactionTagIndexes, transactionTagIds, nil, nil)

			currentProcess = float64(i) / float64(len(transactions)) * 100

			if processHandler != nil && i%processUpdateStep == 0 {
				processHandler(currentProcess)
			}

			if err != nil {
				transactionUnixTime := utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime)
				transactionTimeZone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
				log.Errorf(c, "[transactions.BatchCreateTransactions] failed to create trasaction (datetime: %s, type: %s, amount: %d)", utils.FormatUnixTimeToLongDateTime(transactionUnixTime, transactionTimeZone), transaction.Type, transaction.Amount)
				return err
			}
		}

		return nil
	})
}

// BatchUpdateTransactionsCategory batch updates the categories of transactions
func (s *TransactionService) BatchUpdateTransactionsCategory(c core.Context, uid int64, transactionIds []int64, newCategoryId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if len(transactionIds) < 1 {
		return errs.ErrTransactionIdInvalid
	}

	if newCategoryId <= 0 {
		return errs.ErrTransactionCategoryIdInvalid
	}

	uniqueTransactionIds := utils.ToUniqueInt64Slice(transactionIds)
	now := time.Now().Unix()

	updateModel := &models.Transaction{
		CategoryId:      newCategoryId,
		UpdatedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		updatedRows, err := sess.Cols("category_id", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).In("transaction_id", uniqueTransactionIds).Update(updateModel)

		if err != nil {
			return err
		} else if updatedRows < int64(len(uniqueTransactionIds)) {
			return errs.ErrTransactionNotFound
		}

		return err
	})
}

// BatchAddTagsToTransactions batch adds tags to transactions
func (s *TransactionService) BatchAddTagsToTransactions(c core.Context, uid int64, transactions []*models.Transaction, addTransactionTagIds map[int64][]int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if len(addTransactionTagIds) < 1 {
		return errs.ErrTransactionIdInvalid
	}

	now := time.Now().Unix()
	transactionTagIndexes := make([]*models.TransactionTagIndex, 0, len(addTransactionTagIds))
	transactionsMap := make(map[int64]*models.Transaction, len(transactions))
	transactionTagIdsMap := make(map[int64]bool, 0)

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		transactionsMap[transaction.TransactionId] = transaction
	}

	for transactionId, tagIds := range addTransactionTagIds {
		if transactionId <= 0 {
			return errs.ErrTransactionIdInvalid
		}

		transaction, exists := transactionsMap[transactionId]

		if !exists || transaction == nil {
			return errs.ErrTransactionNotFound
		}

		tagIds = utils.ToUniqueInt64Slice(tagIds)

		for i := 0; i < len(tagIds); i++ {
			tagId := tagIds[i]

			if tagId <= 0 {
				return errs.ErrTransactionTagIdInvalid
			}

			transactionTagIndexes = append(transactionTagIndexes, &models.TransactionTagIndex{
				Uid:             uid,
				Deleted:         false,
				TransactionTime: transaction.TransactionTime,
				TagId:           tagId,
				TransactionId:   transactionId,
				CreatedUnixTime: now,
				UpdatedUnixTime: now,
			})

			transactionTagIdsMap[tagId] = true
		}
	}

	tagIndexUuids := s.GenerateUuids(uuid.UUID_TYPE_TAG_INDEX, uint16(len(transactionTagIndexes)))

	if len(tagIndexUuids) < len(transactionTagIndexes) {
		return errs.ErrCannotAddTagsToTooManyTransactionsOneTime
	}

	for i := 0; i < len(transactionTagIndexes); i++ {
		transactionTagIndexes[i].TagIndexId = tagIndexUuids[i]
	}

	tagIds := make([]int64, 0, len(transactionTagIdsMap))

	for tagId := range transactionTagIdsMap {
		tagIds = append(tagIds, tagId)
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		err := s.isTagsValid(sess, uid, transactionTagIndexes, tagIds)

		if err != nil {
			return err
		}

		for i := 0; i < len(transactionTagIndexes); i++ {
			transactionTagIndex := transactionTagIndexes[i]
			_, err := sess.Insert(transactionTagIndex)

			if err != nil {
				return err
			}
		}

		return nil
	})
}

// BatchRemoveTagsFromTransactions batch removes tags from transactions
func (s *TransactionService) BatchRemoveTagsFromTransactions(c core.Context, uid int64, transactionIds []int64, tagIds []int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if len(transactionIds) < 1 {
		return errs.ErrTransactionIdInvalid
	}

	uniqueTransactionIds := utils.ToUniqueInt64Slice(transactionIds)
	uniqueTagIds := utils.ToUniqueInt64Slice(tagIds)
	now := time.Now().Unix()

	tagIndexUpdateModel := &models.TransactionTagIndex{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		deletedRows, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).In("transaction_id", uniqueTransactionIds).In("tag_id", uniqueTagIds).Update(tagIndexUpdateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrTransactionTagNotFound
		}

		return nil
	})
}

// BatchClearAllTagsFromTransactions batch clears all tags from transactions
func (s *TransactionService) BatchClearAllTagsFromTransactions(c core.Context, uid int64, transactionIds []int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if len(transactionIds) < 1 {
		return errs.ErrTransactionIdInvalid
	}

	uniqueTransactionIds := utils.ToUniqueInt64Slice(transactionIds)
	now := time.Now().Unix()

	tagIndexUpdateModel := &models.TransactionTagIndex{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		deletedRows, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).In("transaction_id", uniqueTransactionIds).Update(tagIndexUpdateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrTransactionTagNotFound
		}

		return nil
	})
}
