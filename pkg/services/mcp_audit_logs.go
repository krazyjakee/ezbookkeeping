package services

import (
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// MCPAuditLogService represents the audit log service of changes made through MCP tools
type MCPAuditLogService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a mcp audit log service singleton instance
var (
	MCPAuditLogs = &MCPAuditLogService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// CreateAuditLog saves a new audit log to database
func (s *MCPAuditLogService) CreateAuditLog(c core.Context, auditLog *models.MCPAuditLog) error {
	if auditLog.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	auditLog.LogId = s.GenerateUuid(uuid.UUID_TYPE_DEFAULT)

	if auditLog.LogId < 1 {
		return errs.ErrSystemIsBusy
	}

	if auditLog.CreatedUnixTime <= 0 {
		auditLog.CreatedUnixTime = time.Now().Unix()
	}

	return s.UserDataDB(auditLog.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Insert(auditLog)
		return err
	})
}

// GetAuditLogs returns the audit logs of the user which match the filter (newest first) and the total count of matched audit logs
func (s *MCPAuditLogService) GetAuditLogs(c core.Context, uid int64, filter *models.MCPAuditLogFilter, page int32, count int32) ([]*models.MCPAuditLog, int64, error) {
	if uid <= 0 {
		return nil, 0, errs.ErrUserIdInvalid
	}

	if page < 1 {
		return nil, 0, errs.ErrPageIndexInvalid
	}

	if count < 1 {
		return nil, 0, errs.ErrPageCountInvalid
	}

	condition, conditionParams := s.buildQueryCondition(uid, filter)
	totalCount, err := s.UserDataDB(uid).NewSession(c).Where(condition, conditionParams...).Count(&models.MCPAuditLog{})

	if err != nil {
		return nil, 0, err
	}

	var auditLogs []*models.MCPAuditLog
	err = s.UserDataDB(uid).NewSession(c).Where(condition, conditionParams...).OrderBy("created_unix_time desc, log_id desc").Limit(int(count), int(count*(page-1))).Find(&auditLogs)

	if err != nil {
		return nil, 0, err
	}

	return auditLogs, totalCount, nil
}

func (s *MCPAuditLogService) buildQueryCondition(uid int64, filter *models.MCPAuditLogFilter) (string, []any) {
	condition := "uid=?"
	conditionParams := []any{uid}

	if filter == nil {
		return condition, conditionParams
	}

	if filter.MinCreatedUnixTime > 0 {
		condition = condition + " AND created_unix_time>=?"
		conditionParams = append(conditionParams, filter.MinCreatedUnixTime)
	}

	if filter.MaxCreatedUnixTime > 0 {
		condition = condition + " AND created_unix_time<=?"
		conditionParams = append(conditionParams, filter.MaxCreatedUnixTime)
	}

	if filter.ToolName != "" {
		condition = condition + " AND tool_name=?"
		conditionParams = append(conditionParams, filter.ToolName)
	}

	if filter.Action != "" {
		condition = condition + " AND action=?"
		conditionParams = append(conditionParams, filter.Action)
	}

	if filter.EntityType != "" {
		condition = condition + " AND entity_type=?"
		conditionParams = append(conditionParams, filter.EntityType)
	}

	if filter.EntityId > 0 {
		// entity ids are stored as ",id1,id2,...,", so that one id can be matched exactly
		condition = condition + " AND entity_ids LIKE ?"
		conditionParams = append(conditionParams, "%,"+utils.Int64ToString(filter.EntityId)+",%")
	}

	if filter.OperationId != "" {
		condition = condition + " AND operation_id=?"
		conditionParams = append(conditionParams, filter.OperationId)
	}

	return condition, conditionParams
}

// DeleteAllAuditLogs deletes all MCP audit logs of the user
func (s *MCPAuditLogService) DeleteAllAuditLogs(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		_, err := sess.Where("uid=?", uid).Delete(&models.MCPAuditLog{})
		return err
	})
}
