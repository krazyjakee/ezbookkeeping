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

// MCPDeleteTransactionAttachmentRequest represents all parameters of the delete transaction attachment request
type MCPDeleteTransactionAttachmentRequest struct {
	AttachmentId string `json:"attachment_id" jsonschema_description:"ID of the attachment to delete"`
	Confirm      bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion of the attachment. There is no restore operation, use detach_transaction_attachment to only remove the reference from the transaction"`
}

// MCPDeleteTransactionAttachmentResponse represents the response structure for delete transaction attachment
type MCPDeleteTransactionAttachmentResponse struct {
	Success               bool   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DeletedAttachmentId   string `json:"deleted_attachment_id" jsonschema_description:"ID of the deleted attachment"`
	PreviousTransactionId string `json:"previous_transaction_id,omitempty" jsonschema_description:"ID of the transaction which the attachment was attached to (empty if it was unattached)"`
	Permanent             bool   `json:"permanent" jsonschema_description:"Always true, deleted attachments cannot be restored through ezBookkeeping"`
}

type mcpDeleteTransactionAttachmentToolHandler struct{}

var MCPDeleteTransactionAttachmentToolHandler = &mcpDeleteTransactionAttachmentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionAttachmentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION_ATTACHMENT
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionAttachmentToolHandler) Name() string {
	return "delete_transaction_attachment"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionAttachmentToolHandler) Description() string {
	return "Permanently delete a transaction attachment (attached or unattached), the same as removing an attachment in the web interface. This is distinct from detach_transaction_attachment, which keeps the file. There is no restore operation, so \"confirm\" must be set to true. For attached attachments, the user's transaction edit scope applies."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionAttachmentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionAttachmentToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionAttachmentToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionAttachmentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionAttachmentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionAttachmentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionAttachmentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionAttachmentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDeleteTransactionAttachmentRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	if _, err := parseMCPRequiredId("attachment_id", request.AttachmentId); err != nil {
		return nil, nil, err
	}

	if !request.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting an attachment is permanent and cannot be undone, set \"confirm\" to true to delete attachment \""+request.AttachmentId+"\" (or use detach_transaction_attachment to keep the file)")
	}

	uid := user.Uid
	pictureInfo, err := mcpAttachmentGetPictureInfo(c, uid, "attachment_id", request.AttachmentId, services)

	if err != nil {
		return nil, nil, err
	}

	response := &MCPDeleteTransactionAttachmentResponse{
		Success:             true,
		DeletedAttachmentId: utils.Int64ToString(pictureInfo.PictureId),
		Permanent:           true,
	}

	if pictureInfo.TransactionId == models.TransactionPictureNewPictureTransactionId {
		err = services.GetTransactionPictureService().RemoveUnusedTransactionPicture(c, uid, pictureInfo.PictureId)

		if err != nil {
			log.Errorf(c, "[delete_transaction_attachment_tool_handler.Handle] failed to delete unattached attachment \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.PictureId, uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		log.Infof(c, "[delete_transaction_attachment_tool_handler.Handle] user \"uid:%d\" has deleted unattached attachment \"id:%d\"", uid, pictureInfo.PictureId)
		return newMCPJSONTextResult(response)
	}

	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, pictureInfo.TransactionId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_attachment_tool_handler.Handle] failed to get transaction \"id:%d\" of attachment \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.TransactionId, pictureInfo.PictureId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if err := mcpAttachmentCheckTransactionEditable(c, uid, user, "attachment_id", transaction, services); err != nil {
		return nil, nil, err
	}

	err = services.GetTransactionService().ModifyTransaction(c, mcpAttachmentCopyTransactionForModify(transaction), false, 0, nil, nil, nil, []int64{pictureInfo.PictureId})

	if err != nil {
		log.Errorf(c, "[delete_transaction_attachment_tool_handler.Handle] failed to delete attachment \"id:%d\" of transaction \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.PictureId, transaction.TransactionId, uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[delete_transaction_attachment_tool_handler.Handle] user \"uid:%d\" has deleted attachment \"id:%d\" of transaction \"id:%d\"", uid, pictureInfo.PictureId, transaction.TransactionId)

	response.PreviousTransactionId = utils.Int64ToString(transaction.TransactionId)
	return newMCPJSONTextResult(response)
}
