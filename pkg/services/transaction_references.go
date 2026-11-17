package services

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// transactionReferenceQueryChunkSize is the maximum count of values in one "IN" condition when querying transaction references
const transactionReferenceQueryChunkSize = 500

// TransactionReferenceService represents transaction reference (evidence / source reference) service
type TransactionReferenceService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a transaction reference service singleton instance
var (
	TransactionReferences = &TransactionReferenceService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetReferencesByTransactionIds returns the references of the given transactions, grouped by transaction id and ordered by display order
func (s *TransactionReferenceService) GetReferencesByTransactionIds(c core.Context, uid int64, transactionIds []int64) (map[int64][]*models.TransactionReference, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	result := make(map[int64][]*models.TransactionReference)

	for start := 0; start < len(transactionIds); start += transactionReferenceQueryChunkSize {
		end := start + transactionReferenceQueryChunkSize

		if end > len(transactionIds) {
			end = len(transactionIds)
		}

		var references []*models.TransactionReference
		err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).In("transaction_id", transactionIds[start:end]).OrderBy("transaction_id asc, display_order asc").Find(&references)

		if err != nil {
			return nil, err
		}

		for i := 0; i < len(references); i++ {
			reference := references[i]
			result[reference.TransactionId] = append(result[reference.TransactionId], reference)
		}
	}

	return result, nil
}

// GetTransactionIdsByReference returns the ids of transactions (at most limit) which have a reference with the given value (and the given type if not empty)
func (s *TransactionReferenceService) GetTransactionIdsByReference(c core.Context, uid int64, referenceType string, referenceValue string, limit int) ([]int64, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if limit < 1 {
		return nil, errs.ErrPageCountInvalid
	}

	var references []*models.TransactionReference
	sess := s.UserDataDB(uid).NewSession(c)

	if referenceType != "" {
		sess = sess.Where("uid=? AND reference_type=? AND reference_value=?", uid, referenceType, referenceValue)
	} else {
		sess = sess.Where("uid=? AND reference_value=?", uid, referenceValue)
	}

	err := sess.Cols("transaction_id").OrderBy("transaction_id desc").Limit(limit).Find(&references)

	if err != nil {
		return nil, err
	}

	transactionIds := make([]int64, 0, len(references))
	transactionIdExists := make(map[int64]bool, len(references))

	for i := 0; i < len(references); i++ {
		if transactionIdExists[references[i].TransactionId] {
			continue
		}

		transactionIdExists[references[i].TransactionId] = true
		transactionIds = append(transactionIds, references[i].TransactionId)
	}

	return transactionIds, nil
}

// GetReferencesByTypesAndValues returns all references which have one of the given types and one of the given values
func (s *TransactionReferenceService) GetReferencesByTypesAndValues(c core.Context, uid int64, referenceTypes []string, referenceValues []string) ([]*models.TransactionReference, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	allReferences := make([]*models.TransactionReference, 0)

	if len(referenceTypes) < 1 || len(referenceValues) < 1 {
		return allReferences, nil
	}

	for start := 0; start < len(referenceValues); start += transactionReferenceQueryChunkSize {
		end := start + transactionReferenceQueryChunkSize

		if end > len(referenceValues) {
			end = len(referenceValues)
		}

		var references []*models.TransactionReference
		err := s.UserDataDB(uid).NewSession(c).Where("uid=?", uid).In("reference_type", referenceTypes).In("reference_value", referenceValues[start:end]).Find(&references)

		if err != nil {
			return nil, err
		}

		allReferences = append(allReferences, references...)
	}

	return allReferences, nil
}

// SetTransactionReferences replaces all references of the given transaction of the user with the given references,
// the transaction must exist and belong to the user
func (s *TransactionReferenceService) SetTransactionReferences(c core.Context, uid int64, transactionId int64, references []*models.TransactionReference) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if transactionId <= 0 {
		return errs.ErrTransactionIdInvalid
	}

	var referenceIds []int64

	if len(references) > 0 {
		referenceIds = s.GenerateUuids(uuid.UUID_TYPE_DEFAULT, uint16(len(references)))

		if len(referenceIds) < len(references) {
			return errs.ErrSystemIsBusy
		}
	}

	now := time.Now().Unix()

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		exists, err := sess.Cols("uid", "deleted", "transaction_id").Where("uid=? AND deleted=? AND transaction_id=?", uid, false, transactionId).Limit(1).Exist(&models.Transaction{})

		if err != nil {
			return err
		} else if !exists {
			return errs.ErrTransactionNotFound
		}

		_, err = sess.Where("uid=? AND transaction_id=?", uid, transactionId).Delete(&models.TransactionReference{})

		if err != nil {
			return err
		}

		for i := 0; i < len(references); i++ {
			reference := &models.TransactionReference{
				ReferenceId:     referenceIds[i],
				Uid:             uid,
				TransactionId:   transactionId,
				ReferenceType:   references[i].ReferenceType,
				ReferenceValue:  references[i].ReferenceValue,
				DisplayOrder:    int32(i + 1),
				CreatedUnixTime: now,
				UpdatedUnixTime: now,
			}

			_, err = sess.Insert(reference)

			if err != nil {
				return err
			}

			references[i].ReferenceId = reference.ReferenceId
			references[i].Uid = uid
			references[i].TransactionId = transactionId
			references[i].DisplayOrder = reference.DisplayOrder
			references[i].CreatedUnixTime = now
			references[i].UpdatedUnixTime = now
		}

		return nil
	})
}

// DeleteTransactionReferences deletes all references of the given transaction of the user
func (s *TransactionReferenceService) DeleteTransactionReferences(c core.Context, uid int64, transactionId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if transactionId <= 0 {
		return errs.ErrTransactionIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Where("uid=? AND transaction_id=?", uid, transactionId).Delete(&models.TransactionReference{})
		return err
	})
}

// DeleteAllTransactionReferences deletes all transaction references of the user
func (s *TransactionReferenceService) DeleteAllTransactionReferences(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Where("uid=?", uid).Delete(&models.TransactionReference{})
		return err
	})
}
