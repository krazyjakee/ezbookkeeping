package services

import (
	"fmt"
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

const pageCountForLoadTransactionAmounts = 1000

// TransactionService represents transaction service
type TransactionService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a transaction service singleton instance
var (
	Transactions = &TransactionService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// CreateTransaction saves a new transaction to database
func (s *TransactionService) CreateTransaction(c core.Context, transaction *models.Transaction, tagIds []int64, pictureIds []int64) error {
	if transaction.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	err := s.isAccountIdValid(transaction)

	if err != nil {
		return err
	}

	now := time.Now().Unix()

	needTransactionUuidCount := 1

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		needTransactionUuidCount = 2
	}

	transactionUuids := s.GenerateUuids(uuid.UUID_TYPE_TRANSACTION, uint16(needTransactionUuidCount))

	if len(transactionUuids) < needTransactionUuidCount {
		return errs.ErrSystemIsBusy
	}

	tagIds = utils.ToUniqueInt64Slice(tagIds)
	needTagIndexUuidCount := uint16(len(tagIds))
	tagIndexUuids := s.GenerateUuids(uuid.UUID_TYPE_TAG_INDEX, needTagIndexUuidCount)

	if len(tagIndexUuids) < int(needTagIndexUuidCount) {
		return errs.ErrSystemIsBusy
	}

	transaction.TransactionId = transactionUuids[0]

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		transaction.RelatedId = transactionUuids[1]
	}

	transaction.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))

	transaction.CreatedUnixTime = now
	transaction.UpdatedUnixTime = now

	transactionTagIndexes := make([]*models.TransactionTagIndex, len(tagIds))

	for i := 0; i < len(tagIds); i++ {
		transactionTagIndexes[i] = &models.TransactionTagIndex{
			TagIndexId:      tagIndexUuids[i],
			Uid:             transaction.Uid,
			Deleted:         false,
			TagId:           tagIds[i],
			TransactionId:   transaction.TransactionId,
			CreatedUnixTime: now,
			UpdatedUnixTime: now,
		}
	}

	pictureUpdateModel := &models.TransactionPictureInfo{
		TransactionId:   transaction.TransactionId,
		UpdatedUnixTime: now,
	}

	userDataDb := s.UserDataDB(transaction.Uid)

	return userDataDb.DoTransaction(c, func(sess datastore.DatabaseSession) error {
		return s.doCreateTransaction(c, userDataDb, sess, transaction, transactionTagIndexes, tagIds, pictureIds, pictureUpdateModel)
	})
}

// ModifyTransaction saves an existed transaction to database
func (s *TransactionService) ModifyTransaction(c core.Context, transaction *models.Transaction, changeToTransfer bool, currentTagIdsCount int, addTagIds []int64, removeTagIds []int64, addPictureIds []int64, removePictureIds []int64) error {
	if transaction.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE &&
		transaction.Type != models.TRANSACTION_DB_TYPE_INCOME &&
		transaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE &&
		transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		log.Warnf(c, "[transactions.ModifyTransaction] not allowed to modify to transaction type %d", transaction.Type)
		return errs.ErrTransactionTypeInvalid
	}

	var newRelatedTransactionId int64

	if changeToTransfer {
		newRelatedTransactionId = s.GenerateUuid(uuid.UUID_TYPE_TRANSACTION)

		if newRelatedTransactionId < 1 {
			return errs.ErrSystemIsBusy
		}
	}

	needTagIndexUuidCount := uint16(len(addTagIds))
	tagIndexUuids := s.GenerateUuids(uuid.UUID_TYPE_TAG_INDEX, needTagIndexUuidCount)

	if len(tagIndexUuids) < int(needTagIndexUuidCount) {
		return errs.ErrSystemIsBusy
	}

	updateCols := make([]string, 0, 16)

	now := time.Now().Unix()

	transaction.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))
	transaction.UpdatedUnixTime = now
	updateCols = append(updateCols, "updated_unix_time")

	addTagIds = utils.ToUniqueInt64Slice(addTagIds)
	removeTagIds = utils.ToUniqueInt64Slice(removeTagIds)

	transactionTagIndexes := make([]*models.TransactionTagIndex, len(addTagIds))

	for i := 0; i < len(addTagIds); i++ {
		transactionTagIndexes[i] = &models.TransactionTagIndex{
			TagIndexId:      tagIndexUuids[i],
			Uid:             transaction.Uid,
			Deleted:         false,
			TagId:           addTagIds[i],
			TransactionId:   transaction.TransactionId,
			CreatedUnixTime: now,
			UpdatedUnixTime: now,
		}
	}

	err := s.UserDataDB(transaction.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		oldTransaction := &models.Transaction{}
		has, err := sess.ID(transaction.TransactionId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(oldTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.ModifyTransaction] failed to get current transaction, because %s", err.Error())
			return err
		} else if !has {
			return errs.ErrTransactionNotFound
		}

		if oldTransaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE &&
			oldTransaction.Type != models.TRANSACTION_DB_TYPE_INCOME &&
			oldTransaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE &&
			oldTransaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			log.Warnf(c, "[transactions.ModifyTransaction] transaction type %d is not allowed to modify", oldTransaction.Type)
			return errs.ErrTransactionTypeInvalid
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE && oldTransaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			return errs.ErrTransactionTypeInvalid
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			transaction.RelatedId = oldTransaction.RelatedId
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && oldTransaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			if newRelatedTransactionId < 1 {
				return errs.ErrSystemIsBusy
			}

			transaction.RelatedId = newRelatedTransactionId
			updateCols = append(updateCols, "related_id")
		}

		err = s.isAccountIdValid(transaction)

		if err != nil {
			return err
		}

		sourceAccount, destinationAccount, err := s.getAccountModels(sess, transaction)

		if err != nil {
			log.Errorf(c, "[transactions.ModifyTransaction] failed to get account, because %s", err.Error())
			return err
		}

		if sourceAccount.Hidden || (destinationAccount != nil && destinationAccount.Hidden) {
			return errs.ErrCannotModifyTransactionInHiddenAccount
		}

		if sourceAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS || (destinationAccount != nil && destinationAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS) {
			return errs.ErrCannotModifyTransactionInParentAccount
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && sourceAccount.Currency == destinationAccount.Currency && transaction.Amount != transaction.RelatedAccountAmount {
			return errs.ErrTransactionSourceAndDestinationAmountNotEqual
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && (transaction.Amount < 0 || transaction.RelatedAccountAmount < 0) {
			return errs.ErrTransferTransactionAmountCannotBeLessThanZero
		}

		oldSourceAccount, oldDestinationAccount, err := s.getOldAccountModels(sess, transaction, oldTransaction, sourceAccount, destinationAccount)

		if err != nil {
			log.Errorf(c, "[transactions.ModifyTransaction] failed to get old account, because %s", err.Error())
			return err
		}

		if oldSourceAccount.Hidden || (oldDestinationAccount != nil && oldDestinationAccount.Hidden) {
			return errs.ErrCannotAddTransactionToHiddenAccount
		}

		if transaction.Type != oldTransaction.Type || transaction.CategoryId != oldTransaction.CategoryId {
			err = s.isCategoryValid(sess, transaction)

			if err != nil {
				return err
			}

			updateCols = append(updateCols, "category_id")
		}

		if transaction.Type != oldTransaction.Type {
			updateCols = append(updateCols, "type")
		}

		modifyTransactionTime := false

		if utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) != utils.GetUnixTimeFromTransactionTime(oldTransaction.TransactionTime) {
			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
				return errs.ErrBalanceModificationTransactionCannotModifyTime
			}

			sameSecondLatestTransaction := &models.Transaction{}
			minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))
			maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))

			has, err = sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", transaction.Uid, minTransactionTime, maxTransactionTime).OrderBy("transaction_time desc").Limit(1).Get(sameSecondLatestTransaction)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to get trasaction time, because %s", err.Error())
				return err
			}

			if has && sameSecondLatestTransaction.TransactionTime < maxTransactionTime-1 {
				transaction.TransactionTime = sameSecondLatestTransaction.TransactionTime + 1
			} else if has && sameSecondLatestTransaction.TransactionTime == maxTransactionTime-1 {
				return errs.ErrTooMuchTransactionInOneSecond
			}

			updateCols = append(updateCols, "transaction_time")
			modifyTransactionTime = true
		}

		if transaction.TimezoneUtcOffset != oldTransaction.TimezoneUtcOffset {
			updateCols = append(updateCols, "timezone_utc_offset")
		}

		if transaction.AccountId != oldTransaction.AccountId {
			updateCols = append(updateCols, "account_id")
		}

		if transaction.Amount != oldTransaction.Amount {
			if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
				transaction.RelatedAccountAmount = oldTransaction.RelatedAccountAmount + transaction.Amount - oldTransaction.Amount
				updateCols = append(updateCols, "related_account_amount")
			}

			updateCols = append(updateCols, "amount")
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			if transaction.RelatedAccountId != oldTransaction.RelatedAccountId || oldTransaction.Type == models.TRANSACTION_DB_TYPE_INCOME || oldTransaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
				updateCols = append(updateCols, "related_account_id")
			}

			if transaction.RelatedAccountAmount != oldTransaction.RelatedAccountAmount || oldTransaction.Type == models.TRANSACTION_DB_TYPE_INCOME || oldTransaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
				updateCols = append(updateCols, "related_account_amount")
			}
		} else if (transaction.Type == models.TRANSACTION_DB_TYPE_INCOME || transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE) && oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			transaction.RelatedAccountId = 0
			transaction.RelatedAccountAmount = 0
			updateCols = append(updateCols, "related_account_id")
			updateCols = append(updateCols, "related_account_amount")
		}

		if transaction.HideAmount != oldTransaction.HideAmount {
			updateCols = append(updateCols, "hide_amount")
		}

		if transaction.Comment != oldTransaction.Comment {
			updateCols = append(updateCols, "comment")
		}

		if transaction.GeoLongitude != oldTransaction.GeoLongitude {
			updateCols = append(updateCols, "geo_longitude")
		}

		if transaction.GeoLatitude != oldTransaction.GeoLatitude {
			updateCols = append(updateCols, "geo_latitude")
		}

		err = s.isTagsValid(sess, transaction.Uid, transactionTagIndexes, addTagIds)

		if err != nil {
			return err
		}

		err = s.isPicturesValid(sess, transaction, addPictureIds)

		if err != nil {
			return err
		}

		// Not allow to add transaction before balance modification transaction
		if transaction.Type != models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			otherTransactionExists := false

			if destinationAccount != nil && sourceAccount.AccountId != destinationAccount.AccountId {
				otherTransactionExists, err = sess.Cols("uid", "deleted", "account_id").Where("uid=? AND deleted=? AND type=? AND (account_id=? OR account_id=?) AND transaction_time>=?", transaction.Uid, false, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, sourceAccount.AccountId, destinationAccount.AccountId, transaction.TransactionTime).Limit(1).Exist(&models.Transaction{})
			} else {
				otherTransactionExists, err = sess.Cols("uid", "deleted", "account_id").Where("uid=? AND deleted=? AND type=? AND account_id=? AND transaction_time>=?", transaction.Uid, false, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, sourceAccount.AccountId, transaction.TransactionTime).Limit(1).Exist(&models.Transaction{})
			}

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to get whether other transactions exist, because %s", err.Error())
				return err
			} else if otherTransactionExists {
				return errs.ErrCannotAddTransactionBeforeBalanceModificationTransaction
			}
		}

		updatedRows, err := sess.ID(transaction.TransactionId).Cols(updateCols...).Where("uid=? AND deleted=?", transaction.Uid, false).Update(transaction)

		if err != nil {
			log.Errorf(c, "[transactions.ModifyTransaction] failed to update transaction, because %s", err.Error())
			return err
		} else if updatedRows < 1 {
			return errs.ErrTransactionNotFound
		}

		if transaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT && oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			deleteUpdateModel := &models.Transaction{
				Deleted:         true,
				DeletedUnixTime: now,
			}

			deletedRows, err := sess.ID(oldTransaction.RelatedId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", transaction.Uid, false).Update(deleteUpdateModel)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to delete old related transaction, because %s", err.Error())
				return err
			} else if deletedRows < 1 {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to delete old related transaction")
				return errs.ErrDatabaseOperationFailed
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && oldTransaction.Type != models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			relatedTransaction := s.GetRelatedTransferTransaction(transaction)

			sameSecondLatestTransaction := &models.Transaction{}
			minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))
			maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))

			has, err := sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", transaction.Uid, minTransactionTime, maxTransactionTime).OrderBy("transaction_time desc").Limit(1).Get(sameSecondLatestTransaction)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to get trasaction time, because %s", err.Error())
				return err
			} else if !has {
				log.Errorf(c, "[transactions.ModifyTransaction] it should have transactions in %d - %d, but result is empty", minTransactionTime, maxTransactionTime)
				return errs.ErrDatabaseOperationFailed
			} else if sameSecondLatestTransaction.TransactionTime == maxTransactionTime-1 {
				return errs.ErrTooMuchTransactionInOneSecond
			}

			relatedTransaction.TransactionTime = sameSecondLatestTransaction.TransactionTime + 1
			createdRows, err := sess.Insert(relatedTransaction)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to create related transaction, because %s", err.Error())
				return err
			} else if createdRows < 1 {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to create related transaction")
				return errs.ErrDatabaseOperationFailed
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			relatedTransaction := s.GetRelatedTransferTransaction(transaction)

			if utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) != utils.GetUnixTimeFromTransactionTime(relatedTransaction.TransactionTime) {
				return errs.ErrTooMuchTransactionInOneSecond
			}

			relatedUpdateCols := s.getRelatedUpdateColumns(updateCols)
			updatedRows, err := sess.ID(relatedTransaction.TransactionId).Cols(relatedUpdateCols...).Where("uid=? AND deleted=?", relatedTransaction.Uid, false).Update(relatedTransaction)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to update related transaction, because %s", err.Error())
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to update related transaction")
				return errs.ErrDatabaseOperationFailed
			}
		}

		if len(removeTagIds) > 0 {
			tagIndexUpdateModel := &models.TransactionTagIndex{
				Deleted:         true,
				DeletedUnixTime: now,
			}

			deletedRows, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", transaction.Uid, false, transaction.TransactionId).In("tag_id", removeTagIds).Update(tagIndexUpdateModel)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to remove old transaction tag index, because %s", err.Error())
				return err
			} else if deletedRows < 1 {
				return errs.ErrTransactionTagNotFound
			}
		}

		if len(transactionTagIndexes) > 0 {
			for i := 0; i < len(transactionTagIndexes); i++ {
				transactionTagIndex := transactionTagIndexes[i]
				transactionTagIndex.TransactionTime = transaction.TransactionTime

				_, err := sess.Insert(transactionTagIndex)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to add new transaction tag index, because %s", err.Error())
					return err
				}
			}
		} else if len(transactionTagIndexes) == 0 && currentTagIdsCount > 0 && modifyTransactionTime {
			tagIndexUpdateModel := &models.TransactionTagIndex{
				TransactionTime: transaction.TransactionTime,
			}

			_, err := sess.Where("uid=? AND deleted=? AND transaction_id=?", transaction.Uid, false, transaction.TransactionId).Update(tagIndexUpdateModel)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to update transaction tag index, because %s", err.Error())
				return err
			}
		}

		if len(removePictureIds) > 0 {
			pictureUpdateModel := &models.TransactionPictureInfo{
				Deleted:         true,
				DeletedUnixTime: now,
			}

			deletedRows, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", transaction.Uid, false, transaction.TransactionId).In("picture_id", removePictureIds).Update(pictureUpdateModel)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to remove old transaction picture info, because %s", err.Error())
				return err
			} else if deletedRows < 1 {
				return errs.ErrTransactionPictureNotFound
			}
		}

		if len(addPictureIds) > 0 {
			pictureUpdateModel := &models.TransactionPictureInfo{
				TransactionId:   transaction.TransactionId,
				UpdatedUnixTime: now,
			}

			_, err = sess.Cols("transaction_id", "updated_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", transaction.Uid, false, models.TransactionPictureNewPictureTransactionId).In("picture_id", addPictureIds).Update(pictureUpdateModel)

			if err != nil {
				log.Errorf(c, "[transactions.ModifyTransaction] failed to update new transaction picture info, because %s", err.Error())
				return err
			}
		}

		if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			if transaction.AccountId != oldTransaction.AccountId {
				return errs.ErrBalanceModificationTransactionCannotChangeAccountId
			}

			if transaction.Amount != oldTransaction.Amount && transaction.RelatedAccountAmount != oldTransaction.RelatedAccountAmount {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, transaction.RelatedAccountAmount-oldTransaction.RelatedAccountAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			var oldAccountOldAmount = oldTransaction.Amount
			var oldAccountNewAmount int64 = 0
			var newAccountNewAmount int64 = 0

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE || oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				oldAccountOldAmount = -oldTransaction.Amount
			}

			if transaction.AccountId == oldTransaction.AccountId {
				oldAccountNewAmount = transaction.Amount
			} else if transaction.AccountId != oldTransaction.AccountId {
				newAccountNewAmount = transaction.Amount
			}

			if oldAccountNewAmount != oldAccountOldAmount {
				oldSourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, oldSourceAccount, oldAccountNewAmount-oldAccountOldAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if newAccountNewAmount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, newAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				oldDestinationAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, oldDestinationAccount, -oldTransaction.RelatedAccountAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update related account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update related account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			var oldAccountOldAmount = oldTransaction.Amount
			var oldAccountNewAmount int64 = 0
			var newAccountNewAmount int64 = 0

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
				oldAccountOldAmount = -oldTransaction.Amount
			}

			if transaction.AccountId == oldTransaction.AccountId {
				oldAccountNewAmount = transaction.Amount
			} else if transaction.AccountId != oldTransaction.AccountId {
				newAccountNewAmount = transaction.Amount
			}

			if oldAccountNewAmount != oldAccountOldAmount {
				oldSourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, oldSourceAccount, oldAccountOldAmount-oldAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if newAccountNewAmount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, -newAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				oldDestinationAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, oldDestinationAccount, -oldTransaction.RelatedAccountAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update related account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update related account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			var oldSourceAccountOldAmount = oldTransaction.Amount
			var oldSourceAccountNewAmount int64 = 0
			var newSourceAccountNewAmount int64 = 0

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
				oldSourceAccountOldAmount = -oldTransaction.Amount
			}

			if transaction.AccountId == oldTransaction.AccountId {
				oldSourceAccountNewAmount = transaction.Amount
			} else if transaction.AccountId != oldTransaction.AccountId {
				newSourceAccountNewAmount = transaction.Amount
			}

			if oldSourceAccountNewAmount != oldSourceAccountOldAmount {
				oldSourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, oldSourceAccount, oldSourceAccountOldAmount-oldSourceAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if newSourceAccountNewAmount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, -newSourceAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			var oldDestinationAccountNewAmount int64 = 0
			var newDestinationAccountNewAmount int64 = 0

			if transaction.RelatedAccountId == oldTransaction.RelatedAccountId {
				oldDestinationAccountNewAmount = transaction.RelatedAccountAmount
			} else if transaction.RelatedAccountId != oldTransaction.RelatedAccountId {
				newDestinationAccountNewAmount = transaction.RelatedAccountAmount
			}

			if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				if oldDestinationAccountNewAmount != oldTransaction.RelatedAccountAmount {
					oldDestinationAccount.UpdatedUnixTime = time.Now().Unix()
					updatedRows, err := s.updateAccountBalance(sess, oldDestinationAccount, oldDestinationAccountNewAmount-oldTransaction.RelatedAccountAmount)

					if err != nil {
						log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
						return err
					} else if updatedRows < 1 {
						log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
						return errs.ErrDatabaseOperationFailed
					}
				}
			}

			if newDestinationAccountNewAmount != 0 {
				destinationAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, destinationAccount, newDestinationAccountNewAmount)

				if err != nil {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance, because %s", err.Error())
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.ModifyTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			return errs.ErrTransactionTypeInvalid
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// MoveAllTransactionsBetweenAccounts moves all transactions from one account to another account, and combine balance modification transactions if necessary
func (s *TransactionService) MoveAllTransactionsBetweenAccounts(c core.Context, uid int64, fromAccountId int64, toAccountId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if fromAccountId <= 0 || toAccountId <= 0 {
		return errs.ErrAccountIdInvalid
	}

	if fromAccountId == toAccountId {
		return errs.ErrCannotMoveTransactionToSameAccount
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		fromAccount := &models.Account{}
		has, err := sess.ID(fromAccountId).Where("uid=? AND deleted=?", uid, false).Get(fromAccount)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrAccountNotFound
		}

		toAccount := &models.Account{}
		has, err = sess.ID(toAccountId).Where("uid=? AND deleted=?", uid, false).Get(toAccount)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrAccountNotFound
		}

		if fromAccount.Hidden || toAccount.Hidden {
			return errs.ErrCannotMoveTransactionFromOrToHiddenAccount
		}

		if fromAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS || toAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			return errs.ErrCannotMoveTransactionFromOrToParentAccount
		}

		if fromAccount.Currency != toAccount.Currency {
			return errs.ErrCannotMoveTransactionBetweenAccountsWithDifferentCurrencies
		}

		// combine balance modification transaction
		var balanceModificationTransactions []*models.Transaction
		err = sess.Where("uid=? AND deleted=? AND type=? AND (account_id=? OR account_id=?)", uid, false, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, fromAccountId, toAccountId).Find(&balanceModificationTransactions)

		if err != nil {
			return err
		}

		if len(balanceModificationTransactions) > 2 {
			log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has more than 2 balance modification transactions in account \"id:%d\" and account \"id:%d\", cannot combine balance modification transaction", uid, fromAccountId, toAccountId)
			return errs.ErrOperationFailed
		} else if len(balanceModificationTransactions) == 2 && balanceModificationTransactions[0].AccountId != balanceModificationTransactions[1].AccountId {
			// if two balance modification transactions exist, merge the amounts into the earlier one and delete the later transaction
			var earlierTransaction *models.Transaction
			var laterTransaction *models.Transaction

			if balanceModificationTransactions[0].TransactionTime < balanceModificationTransactions[1].TransactionTime {
				earlierTransaction = balanceModificationTransactions[0]
				laterTransaction = balanceModificationTransactions[1]
			} else {
				earlierTransaction = balanceModificationTransactions[1]
				laterTransaction = balanceModificationTransactions[0]
			}

			mergedAmount, validAmount := utils.AddInt64(earlierTransaction.Amount, laterTransaction.Amount)
			mergedRelatedAccountAmount, validRelatedAmount := utils.AddInt64(earlierTransaction.RelatedAccountAmount, laterTransaction.RelatedAccountAmount)

			if !validAmount || !validRelatedAmount ||
				(mergedAmount <= models.MinimumTransactionAmount || mergedAmount >= models.MaximumTransactionAmount) ||
				(mergedRelatedAccountAmount <= models.MinimumTransactionAmount || mergedRelatedAccountAmount >= models.MaximumTransactionAmount) {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" cannot combine balance modification transactions \"id:%d\" and \"id:%d\", because the merged amount exceeds the supported range", uid, earlierTransaction.TransactionId, laterTransaction.TransactionId)
				return errs.ErrMergedBalanceModificationTransactionAmountOverflow
			}

			earlierTransaction.Amount = mergedAmount
			earlierTransaction.RelatedAccountAmount = mergedRelatedAccountAmount
			earlierTransaction.UpdatedUnixTime = time.Now().Unix()

			updatedRows, err := sess.ID(earlierTransaction.TransactionId).Cols("amount", "related_account_amount", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(earlierTransaction)

			if err != nil {
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to update earlier balance modification transaction")
				return errs.ErrDatabaseOperationFailed
			}

			laterTransaction.Deleted = true
			laterTransaction.DeletedUnixTime = time.Now().Unix()

			deletedRows, err := sess.ID(laterTransaction.TransactionId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(laterTransaction)

			if err != nil {
				return err
			} else if deletedRows < 1 {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to delete later balance modification transaction")
				return errs.ErrDatabaseOperationFailed
			}

			log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has combined two balance modification transactions \"id:%d\" and \"id:%d\", retained transaction is \"id:%d\"", uid, earlierTransaction.TransactionId, laterTransaction.TransactionId, earlierTransaction.TransactionId)
		} else if len(balanceModificationTransactions) == 1 {
			// when merging a new balance modification transaction, if its date is later than the account's earliest transaction, update the balance modification transaction time accordingly
			anotherAccountId := int64(0)

			if balanceModificationTransactions[0].AccountId == fromAccountId {
				anotherAccountId = toAccountId
			} else if balanceModificationTransactions[0].AccountId == toAccountId {
				anotherAccountId = fromAccountId
			} else {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has a balance modification transaction \"id:%d\" which account id is neither \"%d\" nor \"%d\"", uid, balanceModificationTransactions[0].TransactionId, fromAccountId, toAccountId)
				return errs.ErrOperationFailed
			}

			earliestTransaction := &models.Transaction{}
			has, err := sess.Where("uid=? AND deleted=? AND account_id=?", uid, false, anotherAccountId).OrderBy("transaction_time asc").Limit(1).Get(earliestTransaction)

			if err != nil {
				return err
			} else if has && balanceModificationTransactions[0].TransactionTime > earliestTransaction.TransactionTime {
				balanceModificationTransaction := balanceModificationTransactions[0]
				earliestTransactionUnixTime := utils.GetUnixTimeFromTransactionTime(earliestTransaction.TransactionTime)

				newBalanceModificationMinTransactionTime := utils.GetMinTransactionTimeFromUnixTime(earliestTransactionUnixTime - 1)
				newBalanceModificationMaxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(earliestTransactionUnixTime - 1)
				balanceModificationTransaction.TransactionTime = newBalanceModificationMinTransactionTime

				sameSecondLatestTransaction := &models.Transaction{}
				has, err = sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", balanceModificationTransaction.Uid, newBalanceModificationMinTransactionTime, newBalanceModificationMaxTransactionTime).OrderBy("transaction_time desc").Limit(1).Get(sameSecondLatestTransaction)

				if err != nil {
					log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to get trasaction time, because %s", err.Error())
					return err
				}

				if has && sameSecondLatestTransaction.TransactionTime < newBalanceModificationMaxTransactionTime-1 {
					balanceModificationTransaction.TransactionTime = sameSecondLatestTransaction.TransactionTime + 1
				} else if has && sameSecondLatestTransaction.TransactionTime == newBalanceModificationMaxTransactionTime-1 {
					return errs.ErrTooMuchTransactionInOneSecond
				}

				balanceModificationTransaction.UpdatedUnixTime = time.Now().Unix()

				if balanceModificationTransaction.TransactionTime < 0 {
					balanceModificationTransaction.TransactionTime = 0
				}

				updatedRows, err := sess.ID(balanceModificationTransaction.TransactionId).Cols("transaction_time", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(balanceModificationTransaction)

				if err != nil {
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to update balance modification transaction time")
					return errs.ErrDatabaseOperationFailed
				}

				log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has updated balance modification transaction \"id:%d\" time to %d, because earliest transaction time in account \"id:%d\" is %d", uid, balanceModificationTransaction.TransactionId, balanceModificationTransaction.TransactionTime, toAccountId, earliestTransaction.TransactionTime)
			}
		}

		updateModel := &models.Transaction{
			AccountId:       toAccountId,
			UpdatedUnixTime: time.Now().Unix(),
		}

		updatedRows, err := sess.Cols("account_id", "updated_unix_time").Where("uid=? AND deleted=? AND account_id=?", uid, false, fromAccountId).Update(updateModel)

		if err != nil {
			return err
		}

		if updatedRows > 0 {
			log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has moved %d transactions from account \"id:%d\" to account \"id:%d\"", uid, updatedRows, fromAccountId, toAccountId)
		}

		updateRelatedModel := &models.Transaction{
			RelatedAccountId: toAccountId,
			UpdatedUnixTime:  time.Now().Unix(),
		}

		relatedUpdatedRows, err := sess.Cols("related_account_id", "updated_unix_time").Where("uid=? AND deleted=? AND related_account_id=?", uid, false, fromAccountId).Update(updateRelatedModel)

		if err != nil {
			return err
		}

		if updatedRows > 0 {
			log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has moved %d related transactions from account \"id:%d\" to account \"id:%d\"", uid, relatedUpdatedRows, fromAccountId, toAccountId)
		}

		// delete all transfer transactions which related account id and account id are both
		deletedModel := &models.Transaction{
			Deleted:         true,
			DeletedUnixTime: time.Now().Unix(),
		}

		deletedRows, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=? AND (type=? OR type=?) AND account_id=? AND related_account_id=?", uid, false, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, models.TRANSACTION_DB_TYPE_TRANSFER_IN, toAccountId, toAccountId).Update(deletedModel)

		if err != nil {
			return err
		}

		if deletedRows > 0 {
			log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has deleted %d transactions which account id and related account id are both \"%d\"", uid, deletedRows, toAccountId)
		}

		if fromAccount.Balance != 0 {
			newToAccountBalance, ok := utils.AddInt64(toAccount.Balance, fromAccount.Balance)

			if !ok {
				return errs.ErrAccountBalanceOverflow
			}

			toAccount.UpdatedUnixTime = time.Now().Unix()
			updatedRows, err := s.updateAccountBalance(sess, toAccount, fromAccount.Balance)

			if err != nil {
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to update to account balance")
				return errs.ErrDatabaseOperationFailed
			}

			log.Infof(c, "[transactions.MoveAllTransactionsBetweenAccounts] user \"uid:%d\" has updated account \"id:%d\" balance from %d to %d", uid, toAccountId, toAccount.Balance, newToAccountBalance)

			fromAccount.Balance = 0
			fromAccount.UpdatedUnixTime = time.Now().Unix()
			updatedRows, err = sess.ID(fromAccount.AccountId).Cols("balance", "updated_unix_time").Where("uid=? AND deleted=?", fromAccount.Uid, false).Update(fromAccount)

			if err != nil {
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.MoveAllTransactionsBetweenAccounts] failed to update from account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}

		return nil
	})
}

// DeleteTransaction deletes an existed transaction from database
func (s *TransactionService) DeleteTransaction(c core.Context, uid int64, transactionId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.Transaction{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	tagIndexUpdateModel := &models.TransactionTagIndex{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	pictureUpdateModel := &models.TransactionPictureInfo{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		oldTransaction := &models.Transaction{}
		has, err := sess.ID(transactionId).Where("uid=? AND deleted=?", uid, false).Get(oldTransaction)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrTransactionNotFound
		}

		sourceAccount, destinationAccount, err := s.getAccountModels(sess, oldTransaction)

		if err != nil {
			return err
		}

		if sourceAccount.Hidden || (destinationAccount != nil && destinationAccount.Hidden) {
			return errs.ErrCannotDeleteTransactionInHiddenAccount
		}

		if sourceAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS || (destinationAccount != nil && destinationAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS) {
			return errs.ErrCannotDeleteTransactionInParentAccount
		}

		deletedRows, err := sess.ID(oldTransaction.TransactionId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrTransactionNotFound
		}

		if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			deletedRows, err = sess.ID(oldTransaction.RelatedId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

			if err != nil {
				return err
			} else if deletedRows < 1 {
				return errs.ErrTransactionNotFound
			}
		}

		_, err = sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", uid, false, oldTransaction.TransactionId).Update(tagIndexUpdateModel)

		if err != nil {
			return err
		}

		_, err = sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", uid, false, oldTransaction.TransactionId).Update(pictureUpdateModel)

		if err != nil {
			return err
		}

		if oldTransaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
			if oldTransaction.RelatedAccountAmount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, -oldTransaction.RelatedAccountAmount)

				if err != nil {
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.DeleteTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if oldTransaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			if oldTransaction.Amount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, -oldTransaction.Amount)

				if err != nil {
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.DeleteTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if oldTransaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			if oldTransaction.Amount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedRows, err := s.updateAccountBalance(sess, sourceAccount, oldTransaction.Amount)

				if err != nil {
					return err
				} else if updatedRows < 1 {
					log.Errorf(c, "[transactions.DeleteTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			if oldTransaction.Amount != 0 {
				sourceAccount.UpdatedUnixTime = time.Now().Unix()
				updatedSourceRows, err := s.updateAccountBalance(sess, sourceAccount, oldTransaction.Amount)

				if err != nil {
					return err
				} else if updatedSourceRows < 1 {
					log.Errorf(c, "[transactions.DeleteTransaction] failed to update account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}

			if oldTransaction.RelatedAccountAmount != 0 {
				destinationAccount.UpdatedUnixTime = time.Now().Unix()
				updatedDestinationRows, err := s.updateAccountBalance(sess, destinationAccount, -oldTransaction.RelatedAccountAmount)

				if err != nil {
					return err
				} else if updatedDestinationRows < 1 {
					log.Errorf(c, "[transactions.DeleteTransaction] failed to update related account balance")
					return errs.ErrDatabaseOperationFailed
				}
			}
		} else if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			return errs.ErrTransactionTypeInvalid
		}

		return err
	})
}

// DeleteAllTransactions deletes all existed transactions from database
func (s *TransactionService) DeleteAllTransactions(c core.Context, uid int64, deleteAccount bool) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.Transaction{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	tagIndexUpdateModel := &models.TransactionTagIndex{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	pictureUpdateModel := &models.TransactionPictureInfo{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	accountUpdateModel := &models.Account{
		Balance:         0,
		Deleted:         deleteAccount,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		}

		_, err = sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(tagIndexUpdateModel)

		if err != nil {
			return err
		}

		_, err = sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(pictureUpdateModel)

		if err != nil {
			return err
		}

		// Update all accounts to deleted or set amount to zero
		_, err = sess.Cols("balance", "deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(accountUpdateModel)

		if err != nil {
			return err
		}

		return nil
	})
}

// DeleteAllTransactionsOfAccount deletes all existed transactions of specific account from database
func (s *TransactionService) DeleteAllTransactionsOfAccount(c core.Context, uid int64, accountId int64, pageCount int32) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if accountId <= 0 {
		return errs.ErrAccountIdInvalid
	}

	transactions, err := s.GetAllSpecifiedTransactions(c, uid, 0, 0, 0, nil, []int64{accountId}, nil, false, "", "", core.MATCH_MODE_DEFAULT, false, pageCount, true)

	if err != nil {
		return err
	}

	if len(transactions) < 1 {
		return nil
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]

		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			err = s.DeleteTransaction(c, uid, transaction.RelatedId)
		} else {
			err = s.DeleteTransaction(c, uid, transaction.TransactionId)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// GetRelatedTransferTransaction returns the related transaction for transfer transaction
func (s *TransactionService) GetRelatedTransferTransaction(originalTransaction *models.Transaction) *models.Transaction {
	var relatedType models.TransactionDbType
	var relatedTransactionTime int64

	if originalTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		relatedType = models.TRANSACTION_DB_TYPE_TRANSFER_IN
		relatedTransactionTime = originalTransaction.TransactionTime + 1
	} else if originalTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		relatedType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
		relatedTransactionTime = originalTransaction.TransactionTime - 1
	} else {
		return nil
	}

	relatedTransaction := &models.Transaction{
		TransactionId:        originalTransaction.RelatedId,
		Uid:                  originalTransaction.Uid,
		Deleted:              originalTransaction.Deleted,
		Type:                 relatedType,
		CategoryId:           originalTransaction.CategoryId,
		TransactionTime:      relatedTransactionTime,
		TimezoneUtcOffset:    originalTransaction.TimezoneUtcOffset,
		AccountId:            originalTransaction.RelatedAccountId,
		Amount:               originalTransaction.RelatedAccountAmount,
		RelatedId:            originalTransaction.TransactionId,
		RelatedAccountId:     originalTransaction.AccountId,
		RelatedAccountAmount: originalTransaction.Amount,
		Comment:              originalTransaction.Comment,
		GeoLongitude:         originalTransaction.GeoLongitude,
		GeoLatitude:          originalTransaction.GeoLatitude,
		CreatedIp:            originalTransaction.CreatedIp,
		CreatedUnixTime:      originalTransaction.CreatedUnixTime,
		UpdatedUnixTime:      originalTransaction.UpdatedUnixTime,
		DeletedUnixTime:      originalTransaction.DeletedUnixTime,
	}

	return relatedTransaction
}

func (s *TransactionService) doCreateTransaction(c core.Context, database datastore.Database, sess datastore.DatabaseSession, transaction *models.Transaction, transactionTagIndexes []*models.TransactionTagIndex, tagIds []int64, pictureIds []int64, pictureUpdateModel *models.TransactionPictureInfo) error {
	sourceAccount, destinationAccount, err := s.getAccountModels(sess, transaction)

	if err != nil {
		return err
	}

	if sourceAccount.Hidden || (destinationAccount != nil && destinationAccount.Hidden) {
		return errs.ErrCannotAddTransactionToHiddenAccount
	}

	if sourceAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS || (destinationAccount != nil && destinationAccount.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS) {
		return errs.ErrCannotAddTransactionToParentAccount
	}

	if (transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN) &&
		sourceAccount.Currency == destinationAccount.Currency && transaction.Amount != transaction.RelatedAccountAmount {
		return errs.ErrTransactionSourceAndDestinationAmountNotEqual
	}

	if (transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN) &&
		(transaction.Amount < 0 || transaction.RelatedAccountAmount < 0) {
		return errs.ErrTransferTransactionAmountCannotBeLessThanZero
	}

	err = s.isCategoryValid(sess, transaction)

	if err != nil {
		return err
	}

	err = s.isTagsValid(sess, transaction.Uid, transactionTagIndexes, tagIds)

	if err != nil {
		return err
	}

	err = s.isPicturesValid(sess, transaction, pictureIds)

	if err != nil {
		return err
	}

	// Verify balance modification transaction and calculate real amount
	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		otherTransactionExists, err := sess.Cols("uid", "deleted", "account_id").Where("uid=? AND deleted=? AND account_id=?", transaction.Uid, false, sourceAccount.AccountId).Limit(1).Exist(&models.Transaction{})

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to get whether other transactions exist, because %s", err.Error())
			return err
		} else if otherTransactionExists {
			return errs.ErrBalanceModificationTransactionCannotAddWhenNotEmpty
		}

		relatedAccountAmount, ok := utils.SubtractInt64(transaction.Amount, sourceAccount.Balance)

		if !ok {
			return errs.ErrAccountBalanceOverflow
		}

		transaction.RelatedAccountId = transaction.AccountId
		transaction.RelatedAccountAmount = relatedAccountAmount
	} else { // Not allow to add transaction before balance modification transaction
		otherTransactionExists := false

		if destinationAccount != nil && sourceAccount.AccountId != destinationAccount.AccountId {
			otherTransactionExists, err = sess.Cols("uid", "deleted", "account_id").Where("uid=? AND deleted=? AND type=? AND (account_id=? OR account_id=?) AND transaction_time>=?", transaction.Uid, false, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, sourceAccount.AccountId, destinationAccount.AccountId, transaction.TransactionTime).Limit(1).Exist(&models.Transaction{})
		} else {
			otherTransactionExists, err = sess.Cols("uid", "deleted", "account_id").Where("uid=? AND deleted=? AND type=? AND account_id=? AND transaction_time>=?", transaction.Uid, false, models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, sourceAccount.AccountId, transaction.TransactionTime).Limit(1).Exist(&models.Transaction{})
		}

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to get whether other transactions exist, because %s", err.Error())
			return err
		} else if otherTransactionExists {
			return errs.ErrCannotAddTransactionBeforeBalanceModificationTransaction
		}
	}

	var relatedTransaction *models.Transaction

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		relatedTransaction = s.GetRelatedTransferTransaction(transaction)
	}

	insertTransactionSavePointName := "insert_transaction"
	err = database.SetSavePoint(sess, insertTransactionSavePointName)

	if err != nil {
		log.Errorf(c, "[transactions.doCreateTransaction] failed to set save point \"%s\", because %s", insertTransactionSavePointName, err.Error())
		return err
	}

	createdRows, err := sess.Insert(transaction)

	if err != nil || createdRows < 1 { // maybe another transaction has same time
		if err != nil {
			log.Warnf(c, "[transactions.doCreateTransaction] cannot create trasaction, because %s, regenerate transaction time value", err.Error())
		} else {
			log.Warnf(c, "[transactions.doCreateTransaction] cannot create trasaction, regenerate transaction time value")
		}

		err = database.RollbackToSavePoint(sess, insertTransactionSavePointName)

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to rollback to save point \"%s\", because %s", insertTransactionSavePointName, err.Error())
			return err
		}

		sameSecondLatestTransaction := &models.Transaction{}
		minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))
		maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime))

		has, err := sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", transaction.Uid, minTransactionTime, maxTransactionTime).OrderBy("transaction_time desc").Limit(1).Get(sameSecondLatestTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to get trasaction time, because %s", err.Error())
			return err
		} else if !has {
			log.Errorf(c, "[transactions.doCreateTransaction] it should have transactions in %d - %d, but result is empty", minTransactionTime, maxTransactionTime)
			return errs.ErrDatabaseOperationFailed
		} else if sameSecondLatestTransaction.TransactionTime == maxTransactionTime-1 {
			return errs.ErrTooMuchTransactionInOneSecond
		}

		transaction.TransactionTime = sameSecondLatestTransaction.TransactionTime + 1
		createdRows, err := sess.Insert(transaction)

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to add transaction again, because %s", err.Error())
			return err
		} else if createdRows < 1 {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to add transaction again")
			return errs.ErrDatabaseOperationFailed
		}
	}

	if relatedTransaction != nil {
		relatedTransaction.TransactionTime = transaction.TransactionTime + 1

		if utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) != utils.GetUnixTimeFromTransactionTime(relatedTransaction.TransactionTime) {
			return errs.ErrTooMuchTransactionInOneSecond
		}

		createdRows, err := sess.Insert(relatedTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to add related transaction, because %s", err.Error())
			return err
		} else if createdRows < 1 {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to add related transaction")
			return errs.ErrDatabaseOperationFailed
		}
	}

	err = nil

	if len(transactionTagIndexes) > 0 {
		for i := 0; i < len(transactionTagIndexes); i++ {
			transactionTagIndex := transactionTagIndexes[i]
			transactionTagIndex.TransactionTime = transaction.TransactionTime

			_, err := sess.Insert(transactionTagIndex)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to add transaction tag index, because %s", err.Error())
				return err
			}
		}
	}

	if len(pictureIds) > 0 {
		_, err = sess.Cols("transaction_id", "updated_unix_time").Where("uid=? AND deleted=? AND transaction_id=?", transaction.Uid, false, models.TransactionPictureNewPictureTransactionId).In("picture_id", pictureIds).Update(pictureUpdateModel)

		if err != nil {
			log.Errorf(c, "[transactions.doCreateTransaction] failed to update transaction picture info, because %s", err.Error())
			return err
		}
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		if transaction.RelatedAccountAmount != 0 {
			sourceAccount.UpdatedUnixTime = time.Now().Unix()
			updatedRows, err := s.updateAccountBalance(sess, sourceAccount, transaction.RelatedAccountAmount)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance, because %s", err.Error())
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
		if transaction.Amount != 0 {
			sourceAccount.UpdatedUnixTime = time.Now().Unix()
			updatedRows, err := s.updateAccountBalance(sess, sourceAccount, transaction.Amount)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance, because %s", err.Error())
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
		if transaction.Amount != 0 {
			sourceAccount.UpdatedUnixTime = time.Now().Unix()
			updatedRows, err := s.updateAccountBalance(sess, sourceAccount, -transaction.Amount)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance, because %s", err.Error())
				return err
			} else if updatedRows < 1 {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if transaction.Amount != 0 {
			sourceAccount.UpdatedUnixTime = time.Now().Unix()
			updatedSourceRows, err := s.updateAccountBalance(sess, sourceAccount, -transaction.Amount)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance, because %s", err.Error())
				return err
			} else if updatedSourceRows < 1 {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}

		if transaction.RelatedAccountAmount != 0 {
			destinationAccount.UpdatedUnixTime = time.Now().Unix()
			updatedDestinationRows, err := s.updateAccountBalance(sess, destinationAccount, transaction.RelatedAccountAmount)

			if err != nil {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance, because %s", err.Error())
				return err
			} else if updatedDestinationRows < 1 {
				log.Errorf(c, "[transactions.doCreateTransaction] failed to update account balance")
				return errs.ErrDatabaseOperationFailed
			}
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		return errs.ErrTransactionTypeInvalid
	}

	return err
}

func (s *TransactionService) updateAccountBalance(sess datastore.DatabaseSession, account *models.Account, delta int64) (int64, error) {
	if delta == 0 {
		return 1, nil
	}

	updateSession := sess.ID(account.AccountId).Cols("updated_unix_time").Where("uid=? AND deleted=?", account.Uid, false)

	if delta > 0 {
		updateSession = updateSession.Where("balance<=?", int64(math.MaxInt64)-delta)
	} else if delta == math.MinInt64 {
		updateSession = updateSession.Where("balance>=?", int64(0))
	} else {
		updateSession = updateSession.Where("balance>=?", int64(math.MinInt64)-delta)
	}

	updatedRows, err := updateSession.SetExpr("balance", fmt.Sprintf("balance+(%d)", delta)).Update(account)

	if err == nil && updatedRows < 1 {
		return 0, errs.ErrAccountBalanceOverflow
	}

	return updatedRows, err
}

func (s *TransactionService) getAccountModels(sess datastore.DatabaseSession, transaction *models.Transaction) (sourceAccount *models.Account, destinationAccount *models.Account, err error) {
	sourceAccount = &models.Account{}
	destinationAccount = &models.Account{}

	has, err := sess.ID(transaction.AccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(sourceAccount)

	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, errs.ErrSourceAccountNotFound
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		if transaction.RelatedAccountId != 0 && transaction.RelatedAccountId != transaction.AccountId {
			return nil, nil, errs.ErrAccountIdInvalid
		} else {
			destinationAccount = sourceAccount
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME || transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
		if transaction.RelatedAccountId != 0 {
			return nil, nil, errs.ErrAccountIdInvalid
		}

		destinationAccount = nil
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		if transaction.RelatedAccountId <= 0 {
			return nil, nil, errs.ErrAccountIdInvalid
		} else {
			has, err = sess.ID(transaction.RelatedAccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(destinationAccount)

			if err != nil {
				return nil, nil, err
			} else if !has {
				return nil, nil, errs.ErrDestinationAccountNotFound
			}
		}
	}

	// check whether the parent accounts are valid
	if sourceAccount.ParentAccountId > 0 && destinationAccount != nil && sourceAccount.ParentAccountId != destinationAccount.ParentAccountId && destinationAccount.ParentAccountId > 0 {
		var accounts []*models.Account
		err := sess.Where("uid=? AND deleted=? and (account_id=? or account_id=?)", transaction.Uid, false, sourceAccount.ParentAccountId, destinationAccount.ParentAccountId).Find(&accounts)

		if err != nil {
			return nil, nil, err
		}

		if len(accounts) < 2 {
			return nil, nil, errs.ErrAccountNotFound
		}

		for i := 0; i < len(accounts); i++ {
			account := accounts[i]

			if account.Hidden {
				return nil, nil, errs.ErrCannotUseHiddenAccount
			}
		}
	} else if sourceAccount.ParentAccountId > 0 && (destinationAccount == nil || sourceAccount.ParentAccountId == destinationAccount.ParentAccountId || destinationAccount.ParentAccountId == 0) {
		sourceParentAccount := &models.Account{}
		has, err = sess.ID(sourceAccount.ParentAccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(sourceParentAccount)

		if err != nil {
			return nil, nil, err
		} else if !has {
			return nil, nil, errs.ErrSourceAccountNotFound
		}

		if sourceParentAccount.Hidden {
			return nil, nil, errs.ErrCannotUseHiddenAccount
		}
	} else if sourceAccount.ParentAccountId == 0 && destinationAccount != nil && destinationAccount.ParentAccountId > 0 {
		destinationParentAccount := &models.Account{}
		has, err = sess.ID(destinationAccount.ParentAccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(destinationParentAccount)

		if err != nil {
			return nil, nil, err
		} else if !has {
			return nil, nil, errs.ErrDestinationAccountNotFound
		}

		if destinationParentAccount.Hidden {
			return nil, nil, errs.ErrCannotUseHiddenAccount
		}
	}

	return sourceAccount, destinationAccount, nil
}

func (s *TransactionService) getOldAccountModels(sess datastore.DatabaseSession, transaction *models.Transaction, oldTransaction *models.Transaction, sourceAccount *models.Account, destinationAccount *models.Account) (oldSourceAccount *models.Account, oldDestinationAccount *models.Account, err error) {
	oldSourceAccount = &models.Account{}
	oldDestinationAccount = &models.Account{}

	if transaction.AccountId == oldTransaction.AccountId {
		oldSourceAccount = sourceAccount
	} else {
		has, err := sess.ID(oldTransaction.AccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(oldSourceAccount)

		if err != nil {
			return nil, nil, err
		} else if !has {
			return nil, nil, errs.ErrSourceAccountNotFound
		}
	}

	if transaction.RelatedAccountId == oldTransaction.RelatedAccountId {
		oldDestinationAccount = destinationAccount
	} else if oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || oldTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		has, err := sess.ID(oldTransaction.RelatedAccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(oldDestinationAccount)

		if err != nil {
			return nil, nil, err
		} else if !has {
			return nil, nil, errs.ErrDestinationAccountNotFound
		}
	}

	return oldSourceAccount, oldDestinationAccount, nil
}

func (s *TransactionService) getRelatedUpdateColumns(updateCols []string) []string {
	relatedUpdateCols := make([]string, len(updateCols))

	for i := 0; i < len(updateCols); i++ {
		if updateCols[i] == "account_id" {
			relatedUpdateCols[i] = "related_account_id"
		} else if updateCols[i] == "related_account_id" {
			relatedUpdateCols[i] = "account_id"
		} else if updateCols[i] == "amount" {
			relatedUpdateCols[i] = "related_account_amount"
		} else if updateCols[i] == "related_account_amount" {
			relatedUpdateCols[i] = "amount"
		} else {
			relatedUpdateCols[i] = updateCols[i]
		}
	}

	return relatedUpdateCols
}

func (s *TransactionService) isAccountIdValid(transaction *models.Transaction) error {
	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		if transaction.RelatedAccountId != 0 && transaction.RelatedAccountId != transaction.AccountId {
			return errs.ErrTransactionDestinationAccountCannotBeSet
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME ||
		transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
		if transaction.RelatedAccountId != 0 {
			return errs.ErrTransactionDestinationAccountCannotBeSet
		} else if transaction.RelatedAccountAmount != 0 {
			return errs.ErrTransactionDestinationAmountCannotBeSet
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if transaction.AccountId == transaction.RelatedAccountId {
			return errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
		}
	} else if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		return errs.ErrTransactionTypeInvalid
	} else {
		return errs.ErrTransactionTypeInvalid
	}

	return nil
}

func (s *TransactionService) isCategoryValid(sess datastore.DatabaseSession, transaction *models.Transaction) error {
	if transaction.Type == models.TRANSACTION_DB_TYPE_MODIFY_BALANCE {
		if transaction.CategoryId != 0 {
			return errs.ErrBalanceModificationTransactionCannotSetCategory
		}
	} else {
		category := &models.TransactionCategory{}
		has, err := sess.ID(transaction.CategoryId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(category)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrTransactionCategoryNotFound
		}

		if category.Hidden {
			return errs.ErrCannotUseHiddenTransactionCategory
		}

		if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			return errs.ErrCannotUsePrimaryCategoryForTransaction
		}

		if (transaction.Type == models.TRANSACTION_DB_TYPE_INCOME && category.Type != models.CATEGORY_TYPE_INCOME) ||
			(transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE && category.Type != models.CATEGORY_TYPE_EXPENSE) ||
			((transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN) && category.Type != models.CATEGORY_TYPE_TRANSFER) {
			return errs.ErrTransactionCategoryTypeInvalid
		}

		parentCategory := &models.TransactionCategory{}
		has, err = sess.ID(category.ParentCategoryId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(parentCategory)

		if err != nil {
			return err
		} else if !has {
			return errs.ErrTransactionCategoryNotFound
		}

		if parentCategory.Hidden {
			return errs.ErrCannotUseHiddenTransactionCategory
		}
	}

	return nil
}

func (s *TransactionService) isTagsValid(sess datastore.DatabaseSession, uid int64, transactionTagIndexes []*models.TransactionTagIndex, tagIds []int64) error {
	if len(transactionTagIndexes) > 0 {
		var tags []*models.TransactionTag
		err := sess.Where("uid=? AND deleted=?", uid, false).In("tag_id", tagIds).Find(&tags)

		if err != nil {
			return err
		}

		tagMap := make(map[int64]*models.TransactionTag)

		for i := 0; i < len(tags); i++ {
			if tags[i].Hidden {
				return errs.ErrCannotUseHiddenTransactionTag
			}

			tagMap[tags[i].TagId] = tags[i]
		}

		for i := 0; i < len(transactionTagIndexes); i++ {
			if _, exists := tagMap[transactionTagIndexes[i].TagId]; !exists {
				return errs.ErrTransactionTagNotFound
			}
		}
	}

	return nil
}

func (s *TransactionService) isPicturesValid(sess datastore.DatabaseSession, transaction *models.Transaction, pictureIds []int64) error {
	if len(pictureIds) > 0 {
		var pictureInfos []*models.TransactionPictureInfo
		err := sess.Where("uid=? AND deleted=?", transaction.Uid, false).In("picture_id", pictureIds).Find(&pictureInfos)

		if err != nil {
			return err
		}

		pictureInfoMap := make(map[int64]*models.TransactionPictureInfo)

		for i := 0; i < len(pictureInfos); i++ {
			if pictureInfos[i].TransactionId != models.TransactionPictureNewPictureTransactionId && pictureInfos[i].TransactionId != transaction.TransactionId {
				return errs.ErrTransactionPictureIdInvalid
			}

			pictureInfoMap[pictureInfos[i].PictureId] = pictureInfos[i]
		}

		for i := 0; i < len(pictureIds); i++ {
			if _, exists := pictureInfoMap[pictureIds[i]]; !exists {
				return errs.ErrTransactionPictureNotFound
			}
		}
	}

	return nil
}
