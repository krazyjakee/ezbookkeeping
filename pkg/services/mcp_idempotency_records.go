package services

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// MCPIdempotencyRecordService represents the idempotency record service of MCP write tools
type MCPIdempotencyRecordService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a mcp idempotency record service singleton instance
var (
	MCPIdempotencyRecords = &MCPIdempotencyRecordService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetRecordByKey returns the idempotency record of the user with the given key, or nil if not exists
func (s *MCPIdempotencyRecordService) GetRecordByKey(c core.Context, uid int64, idempotencyKey string) (*models.MCPIdempotencyRecord, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	record := &models.MCPIdempotencyRecord{}
	has, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND idempotency_key=?", uid, idempotencyKey).Get(record)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return record, nil
}

// TryCreatePendingRecord saves a new pending (not completed) idempotency record.
// Returns false without error if a record with the same key already exists (e.g. created by a concurrent request)
func (s *MCPIdempotencyRecordService) TryCreatePendingRecord(c core.Context, record *models.MCPIdempotencyRecord) (bool, error) {
	if record.Uid <= 0 {
		return false, errs.ErrUserIdInvalid
	}

	record.RecordId = s.GenerateUuid(uuid.UUID_TYPE_DEFAULT)

	if record.RecordId < 1 {
		return false, errs.ErrSystemIsBusy
	}

	now := time.Now().Unix()
	record.Completed = false
	record.Response = ""
	record.CreatedUnixTime = now
	record.UpdatedUnixTime = now

	err := s.UserDataDB(record.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Insert(record)
		return err
	})

	if err != nil {
		existingRecord, getErr := s.GetRecordByKey(c, record.Uid, record.IdempotencyKey)

		if getErr == nil && existingRecord != nil {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// CompleteRecord saves the response of the pending idempotency record and marks it as completed
func (s *MCPIdempotencyRecordService) CompleteRecord(c core.Context, uid int64, recordId int64, response string) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	updateModel := &models.MCPIdempotencyRecord{
		Completed:       true,
		Response:        response,
		UpdatedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.ID(recordId).Cols("completed", "response", "updated_unix_time").Where("uid=?", uid).Update(updateModel)
		return err
	})
}

// DeleteRecord deletes the idempotency record (e.g. the pending record of a failed write, so that the request can be retried)
func (s *MCPIdempotencyRecordService) DeleteRecord(c core.Context, uid int64, recordId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.ID(recordId).Where("uid=?", uid).Delete(&models.MCPIdempotencyRecord{})
		return err
	})
}

// DeleteExpiredRecords deletes all idempotency records of the user created before the given unix time
func (s *MCPIdempotencyRecordService) DeleteExpiredRecords(c core.Context, uid int64, createdBeforeUnixTime int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Where("uid=? AND created_unix_time<?", uid, createdBeforeUnixTime).Delete(&models.MCPIdempotencyRecord{})
		return err
	})
}

// DeleteAllRecords deletes all MCP idempotency records of the user
func (s *MCPIdempotencyRecordService) DeleteAllRecords(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Where("uid=?", uid).Delete(&models.MCPIdempotencyRecord{})
		return err
	})
}
