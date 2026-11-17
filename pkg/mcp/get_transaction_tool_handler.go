package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPGetTransactionRequest represents all parameters of the get transaction request
type MCPGetTransactionRequest struct {
	TransactionId string `json:"transaction_id" jsonschema_description:"Transaction ID"`
}

// MCPGetTransactionResponse represents the response structure for get transaction
type MCPGetTransactionResponse struct {
	Transaction *MCPTransactionDetail `json:"transaction" jsonschema_description:"Full information of the transaction"`
}

type mcpGetTransactionToolHandler struct{}

var MCPGetTransactionToolHandler = &mcpGetTransactionToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetTransactionToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_TRANSACTION
}

// Name returns the name of the MCP tool
func (h *mcpGetTransactionToolHandler) Name() string {
	return "get_transaction"
}

// Description returns the description of the MCP tool
func (h *mcpGetTransactionToolHandler) Description() string {
	return "Get the full information of a transaction by its ID in ezBookkeeping, including accounts, category, tags, description, location, attachments and whether it can still be edited."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetTransactionToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetTransactionToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetTransactionToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetTransactionToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetTransactionToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetTransactionToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	getTransactionRequest, err := parseMCPToolArguments[MCPGetTransactionRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	transactionId, err := parseMCPRequiredId("transaction_id", getTransactionRequest.TransactionId)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	transaction, err := mcpTxnGetTransaction(c, uid, transactionId, services)

	if err != nil {
		return nil, nil, err
	}

	detailData, err := mcpTxnLoadDetailData(c, uid, []*models.Transaction{transaction}, currentConfig, services)

	if err != nil {
		return nil, nil, err
	}

	response := MCPGetTransactionResponse{
		Transaction: mcpTxnBuildDetail(transaction, detailData, user, true),
	}

	return newMCPJSONTextResult(response)
}

// mcpTxnGetTransaction returns the transaction of the current user, the transfer-in side of a transfer is resolved to the transfer-out side as the web interface does
func mcpTxnGetTransaction(c *core.WebContext, uid int64, transactionId int64, services MCPAvailableServices) (*models.Transaction, error) {
	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, transactionId)

	if err != nil {
		log.Warnf(c, "[get_transaction_tool_handler.mcpTxnGetTransaction] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionId, uid, err.Error())

		if err == errs.ErrTransactionNotFound {
			return nil, NewMCPNotFoundError("transaction_id", "transaction", utils.Int64ToString(transactionId))
		}

		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		transaction = services.GetTransactionService().GetRelatedTransferTransaction(transaction)
	}

	return transaction, nil
}
