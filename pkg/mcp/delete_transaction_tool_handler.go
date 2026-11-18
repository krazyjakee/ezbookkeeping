package mcp

import (
	"encoding/json/v2"
	"reflect"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPDeleteTransactionRequest represents all parameters of the delete transaction request
type MCPDeleteTransactionRequest struct {
	TransactionId string `json:"transaction_id" jsonschema_description:"Transaction ID"`
	Confirm       bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion of the transaction (and its attachments). There is no restore operation"`
}

// MCPDeleteTransactionResponse represents the response structure for delete transaction
type MCPDeleteTransactionResponse struct {
	Success              bool   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DeletedTransactionId string `json:"deleted_transaction_id" jsonschema_description:"ID of the deleted transaction"`
	Permanent            bool   `json:"permanent" jsonschema_description:"Always true, deleted transactions cannot be restored through ezBookkeeping"`
}

type mcpDeleteTransactionToolHandler struct{}

var MCPDeleteTransactionToolHandler = &mcpDeleteTransactionToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionToolHandler) Name() string {
	return "delete_transaction"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionToolHandler) Description() string {
	return "Permanently delete an existing transaction in ezBookkeeping (for transfers, both sides are deleted). Account balances are reverted, and its tags and attachments are removed together with the transaction. ezBookkeeping has no restore (undo) operation, so \"confirm\" must be set to true. The user's transaction edit scope applies."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	var deleteTransactionRequest MCPDeleteTransactionRequest

	if callToolReq.Arguments == nil {
		return nil, nil, errs.ErrIncompleteOrIncorrectSubmission
	}

	if err := json.Unmarshal(callToolReq.Arguments, &deleteTransactionRequest); err != nil {
		return nil, nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	transactionId, err := utils.StringToInt64(deleteTransactionRequest.TransactionId)

	if err != nil || transactionId <= 0 {
		return nil, nil, errs.ErrTransactionIdInvalid
	}

	if !deleteTransactionRequest.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting a transaction is permanent and cannot be undone, set \"confirm\" to true to delete transaction \""+deleteTransactionRequest.TransactionId+"\"")
	}

	uid := user.Uid
	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, transactionId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_tool_handler.Handle] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		log.Warnf(c, "[delete_transaction_tool_handler.Handle] cannot delete transaction \"id:%d\" for user \"uid:%d\", because transaction type is transfer in", transactionId, uid)
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "transaction_id", "transaction \""+deleteTransactionRequest.TransactionId+"\" is the incoming side of a transfer, please delete the transfer by its ID \""+utils.Int64ToString(transaction.RelatedId)+"\" instead")
	}

	usedAccounts, err := h.getTransactionUsedAccounts(c, uid, transaction, services)

	if err != nil {
		return nil, nil, err
	}

	transactionTimezone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)
	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, transactionTimezone, usedAccounts[transaction.AccountId], usedAccounts[transaction.RelatedAccountId])

	if !transactionEditable {
		log.Warnf(c, "[delete_transaction_tool_handler.Handle] transaction \"id:%d\" is not editable for user \"uid:%d\"", transactionId, uid)
		return nil, nil, errs.ErrCannotDeleteTransactionWithThisTransactionTime
	}

	err = services.GetTransactionService().DeleteTransaction(c, uid, transactionId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_tool_handler.Handle] failed to delete transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[delete_transaction_tool_handler.Handle] user \"uid:%d\" has deleted transaction \"id:%d\"", uid, transactionId)

	return h.createMCPDeleteTransactionResponse(transactionId)
}

func (h *mcpDeleteTransactionToolHandler) getTransactionUsedAccounts(c *core.WebContext, uid int64, transaction *models.Transaction, services MCPAvailableServices) (map[int64]*models.Account, error) {
	accountIds := []int64{transaction.AccountId}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		accountIds = append(accountIds, transaction.RelatedAccountId)
	}

	accountMap, err := services.GetAccountService().GetAccountsByAccountIds(c, uid, accountIds)

	if err != nil {
		log.Errorf(c, "[delete_transaction_tool_handler.getTransactionUsedAccounts] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if _, exists := accountMap[transaction.AccountId]; !exists {
		log.Warnf(c, "[delete_transaction_tool_handler.getTransactionUsedAccounts] account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
		return nil, errs.ErrSourceAccountNotFound
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		if _, exists := accountMap[transaction.RelatedAccountId]; !exists {
			log.Warnf(c, "[delete_transaction_tool_handler.getTransactionUsedAccounts] related account of transaction \"id:%d\" does not exist for user \"uid:%d\"", transaction.TransactionId, uid)
			return nil, errs.ErrDestinationAccountNotFound
		}
	}

	return accountMap, nil
}

func (h *mcpDeleteTransactionToolHandler) createMCPDeleteTransactionResponse(transactionId int64) (any, []*MCPTextContent, error) {
	response := MCPDeleteTransactionResponse{
		Success:              true,
		DeletedTransactionId: utils.Int64ToString(transactionId),
		Permanent:            true,
	}

	content, err := json.Marshal(response)

	if err != nil {
		return nil, nil, err
	}

	return response, []*MCPTextContent{
		NewMCPTextContent(string(content)),
	}, nil
}
