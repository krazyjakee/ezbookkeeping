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

// MCPDetachTransactionAttachmentRequest represents all parameters of the detach transaction attachment request
type MCPDetachTransactionAttachmentRequest struct {
	AttachmentId  string `json:"attachment_id" jsonschema_description:"ID of the attachment to detach"`
	TransactionId string `json:"transaction_id,omitempty" jsonschema_description:"ID of the transaction which the attachment is expected to be attached to (optional, if set, the attachment must be attached to this transaction)"`
}

// MCPDetachTransactionAttachmentResponse represents the response structure for detach transaction attachment
type MCPDetachTransactionAttachmentResponse struct {
	Success               bool                          `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Status                string                        `json:"status" jsonschema:"enum=updated,enum=unchanged" jsonschema_description:"Result status (updated if the attachment has been detached, unchanged if it was already unattached)"`
	PreviousTransactionId string                        `json:"previous_transaction_id,omitempty" jsonschema_description:"ID of the transaction which the attachment was attached to"`
	Attachment            *MCPTransactionAttachmentInfo `json:"attachment" jsonschema_description:"The attachment after detached (kept as an unattached attachment)"`
}

type mcpDetachTransactionAttachmentToolHandler struct{}

var MCPDetachTransactionAttachmentToolHandler = &mcpDetachTransactionAttachmentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDetachTransactionAttachmentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DETACH_TRANSACTION_ATTACHMENT
}

// Name returns the name of the MCP tool
func (h *mcpDetachTransactionAttachmentToolHandler) Name() string {
	return "detach_transaction_attachment"
}

// Description returns the description of the MCP tool
func (h *mcpDetachTransactionAttachmentToolHandler) Description() string {
	return "Detach an attachment from its transaction without deleting it. The file is kept as an unattached attachment (see list_transaction_attachments without transaction_id), which can be attached to another transaction via update_transaction (add_attachment_ids) or removed with delete_transaction_attachment. The user's transaction edit scope applies."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDetachTransactionAttachmentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDetachTransactionAttachmentToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDetachTransactionAttachmentToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDetachTransactionAttachmentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDetachTransactionAttachmentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDetachTransactionAttachmentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDetachTransactionAttachmentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDetachTransactionAttachmentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDetachTransactionAttachmentRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	pictureInfo, err := mcpAttachmentGetPictureInfo(c, uid, "attachment_id", request.AttachmentId, services)

	if err != nil {
		return nil, nil, err
	}

	expectedTransactionId := int64(0)

	if request.TransactionId != "" {
		expectedTransaction, err := mcpAttachmentGetTransaction(c, uid, "transaction_id", request.TransactionId, services)

		if err != nil {
			return nil, nil, err
		}

		expectedTransactionId = expectedTransaction.TransactionId
	}

	response := &MCPDetachTransactionAttachmentResponse{
		Success: true,
	}

	if pictureInfo.TransactionId == models.TransactionPictureNewPictureTransactionId {
		if expectedTransactionId != 0 {
			return nil, nil, NewMCPInvalidFieldError("transaction_id", request.TransactionId, "attachment \""+request.AttachmentId+"\" is not attached to any transaction")
		}

		response.Status = MCPWriteStatusUnchanged
		response.Attachment, err = mcpAttachmentLoadInfo(c, uid, pictureInfo, services)

		if err != nil {
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		return newMCPJSONTextResult(response)
	}

	if expectedTransactionId != 0 && expectedTransactionId != pictureInfo.TransactionId {
		return nil, nil, NewMCPInvalidFieldError("transaction_id", request.TransactionId, "attachment \""+request.AttachmentId+"\" is attached to transaction \""+utils.Int64ToString(pictureInfo.TransactionId)+"\", not to transaction \""+request.TransactionId+"\"")
	}

	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, pictureInfo.TransactionId)

	if err != nil {
		log.Errorf(c, "[detach_transaction_attachment_tool_handler.Handle] failed to get transaction \"id:%d\" of attachment \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.TransactionId, pictureInfo.PictureId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if err := mcpAttachmentCheckTransactionEditable(c, uid, user, "attachment_id", transaction, services); err != nil {
		return nil, nil, err
	}

	err = services.GetTransactionPictureService().DetachPictureFromTransaction(c, uid, transaction.TransactionId, pictureInfo.PictureId)

	if err != nil {
		log.Errorf(c, "[detach_transaction_attachment_tool_handler.Handle] failed to detach attachment \"id:%d\" from transaction \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.PictureId, transaction.TransactionId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[detach_transaction_attachment_tool_handler.Handle] user \"uid:%d\" has detached attachment \"id:%d\" from transaction \"id:%d\"", uid, pictureInfo.PictureId, transaction.TransactionId)

	response.Status = MCPWriteStatusUpdated
	response.PreviousTransactionId = utils.Int64ToString(transaction.TransactionId)

	savedPictureInfo, err := services.GetTransactionPictureService().GetPictureInfoByPictureId(c, uid, pictureInfo.PictureId)

	if err != nil {
		log.Warnf(c, "[detach_transaction_attachment_tool_handler.Handle] failed to get attachment \"id:%d\" after detached for user \"uid:%d\", because %s", pictureInfo.PictureId, uid, err.Error())
		savedPictureInfo = pictureInfo
		savedPictureInfo.TransactionId = models.TransactionPictureNewPictureTransactionId
	}

	response.Attachment, err = mcpAttachmentLoadInfo(c, uid, savedPictureInfo, services)

	if err != nil {
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return newMCPJSONTextResult(response)
}
