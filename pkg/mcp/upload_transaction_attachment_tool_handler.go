package mcp

import (
	"bytes"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPUploadTransactionAttachmentRequest represents all parameters of the upload transaction attachment request
type MCPUploadTransactionAttachmentRequest struct {
	ContentBase64 string `json:"content_base64" jsonschema_description:"The file content encoded in base64 (standard alphabet, a \"data:<mime>;base64,\" prefix is allowed). The decoded size must not exceed max_file_size_bytes reported by get_server_capabilities. A file name or local path is not a file content and is rejected"`
	FileExtension string `json:"file_extension,omitempty" jsonschema_description:"File type of the content as file extension (e.g. jpg, png, gif, webp, pdf), required if content_type is not set"`
	ContentType   string `json:"content_type,omitempty" jsonschema_description:"File type of the content as MIME type (e.g. image/jpeg, application/pdf), required if file_extension is not set"`
	TransactionId string `json:"transaction_id,omitempty" jsonschema_description:"ID of an existing transaction to attach the uploaded file to (optional). If not set, the attachment is kept unattached and can be attached later via attachment_ids of add_transaction or add_attachment_ids of update_transaction"`
	DryRun        bool   `json:"dry_run,omitempty" jsonschema_description:"If true, the content is only validated (size, type, content and transaction checks) and nothing is saved (optional)"`
}

// MCPUploadTransactionAttachmentResponse represents the response structure for upload transaction attachment
type MCPUploadTransactionAttachmentResponse struct {
	Success           bool                          `json:"success" jsonschema_description:"Indicates whether the file has been validated (dry run) or saved"`
	DryRun            bool                          `json:"dry_run,omitzero" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Committed         bool                          `json:"committed" jsonschema_description:"Indicates whether the file has been saved"`
	Status            string                        `json:"status" jsonschema:"enum=created,enum=would_be_created" jsonschema_description:"Result status"`
	UploadedSizeBytes int64                         `json:"uploaded_size_bytes" jsonschema_description:"Size in bytes of the decoded uploaded content"`
	UploadedSHA256    string                        `json:"uploaded_sha256" jsonschema_description:"Hex encoded SHA-256 checksum of the decoded uploaded content"`
	ContentType       string                        `json:"content_type" jsonschema_description:"MIME type of the uploaded content"`
	Verified          bool                          `json:"verified" jsonschema_description:"Whether the stored file has been read back and its size and SHA-256 checksum match the uploaded content (only when committed)"`
	Attached          bool                          `json:"attached" jsonschema_description:"Whether the attachment is attached to the transaction specified by transaction_id"`
	AttachError       string                        `json:"attach_error,omitempty" jsonschema_description:"Why the saved attachment could not be attached to the transaction (the attachment is kept unattached, see next_step)"`
	NextStep          string                        `json:"next_step,omitempty" jsonschema_description:"Suggested next step"`
	Attachment        *MCPTransactionAttachmentInfo `json:"attachment,omitempty" jsonschema_description:"The saved attachment read back from the server (only when committed)"`
}

// mcpAttachmentContentFile wraps the decoded attachment content as a file which can be saved by the transaction picture service
type mcpAttachmentContentFile struct {
	*bytes.Reader
}

// Close implements the io.Closer interface
func (f *mcpAttachmentContentFile) Close() error {
	return nil
}

type mcpUploadTransactionAttachmentToolHandler struct{}

var MCPUploadTransactionAttachmentToolHandler = &mcpUploadTransactionAttachmentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUploadTransactionAttachmentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPLOAD_TRANSACTION_ATTACHMENT
}

// Name returns the name of the MCP tool
func (h *mcpUploadTransactionAttachmentToolHandler) Name() string {
	return "upload_transaction_attachment"
}

// Description returns the description of the MCP tool
func (h *mcpUploadTransactionAttachmentToolHandler) Description() string {
	return "Upload a receipt picture or PDF document as a transaction attachment, passing the file content as base64 in content_base64 (decoded size limited by max_file_size_bytes of get_server_capabilities). The content is validated against the file type (extension and magic bytes). Optionally attach it to an existing transaction (transaction_id, the transaction edit scope and attachment count limit apply); otherwise it stays unattached and its ID can be passed to add_transaction (attachment_ids) or update_transaction (add_attachment_ids). Returns the attachment ID, transaction association, size, MIME type and the SHA-256 checksum read back from storage. " + mcpAttachmentLimitationNote
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUploadTransactionAttachmentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUploadTransactionAttachmentToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUploadTransactionAttachmentToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUploadTransactionAttachmentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUploadTransactionAttachmentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUploadTransactionAttachmentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUploadTransactionAttachmentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUploadTransactionAttachmentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUploadTransactionAttachmentRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	extension, err := mcpAttachmentResolveExtension(request.FileExtension, request.ContentType)

	if err != nil {
		return nil, nil, err
	}

	data, err := mcpAttachmentDecodeContent("content_base64", request.ContentBase64, mcpAttachmentGetMaxFileSize(currentConfig))

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentValidateContent("content_base64", extension, data); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	var transaction *models.Transaction

	if request.TransactionId != "" {
		transaction, err = mcpAttachmentGetTransaction(c, uid, "transaction_id", request.TransactionId, services)

		if err != nil {
			return nil, nil, err
		}

		if err := mcpAttachmentCheckTransactionEditable(c, uid, user, "transaction_id", transaction, services); err != nil {
			return nil, nil, err
		}

		existingPictureInfos, err := services.GetTransactionPictureService().GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[upload_transaction_attachment_tool_handler.Handle] failed to get attachments of transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		if count := len(existingPictureInfos) + 1; count > models.MaximumPicturesCountOfTransaction {
			return nil, nil, NewMCPTooManyItemsError("transaction_id", models.MaximumPicturesCountOfTransaction, count)
		}
	}

	response := &MCPUploadTransactionAttachmentResponse{
		Success:           true,
		UploadedSizeBytes: int64(len(data)),
		UploadedSHA256:    mcpAttachmentComputeChecksum(data),
		ContentType:       utils.GetTransactionAttachmentContentType(extension),
	}

	if request.DryRun {
		response.DryRun = true
		response.Status = MCPWriteStatusWouldBeCreated
		return newMCPJSONTextResult(response)
	}

	pictureInfo := &models.TransactionPictureInfo{
		Uid:              uid,
		TransactionId:    models.TransactionPictureNewPictureTransactionId,
		PictureExtension: extension,
		CreatedIp:        c.ClientIP(),
	}

	err = services.GetTransactionPictureService().UploadPicture(c, pictureInfo, &mcpAttachmentContentFile{Reader: bytes.NewReader(data)})

	if err != nil {
		log.Errorf(c, "[upload_transaction_attachment_tool_handler.Handle] failed to upload transaction attachment for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[upload_transaction_attachment_tool_handler.Handle] user \"uid:%d\" has uploaded a new transaction attachment \"id:%d\" successfully", uid, pictureInfo.PictureId)

	response.Committed = true
	response.Status = MCPWriteStatusCreated
	attachmentId := utils.Int64ToString(pictureInfo.PictureId)

	if transaction != nil {
		err = services.GetTransactionService().ModifyTransaction(c, mcpAttachmentCopyTransactionForModify(transaction), false, 0, nil, nil, []int64{pictureInfo.PictureId}, nil)

		if err != nil {
			log.Errorf(c, "[upload_transaction_attachment_tool_handler.Handle] failed to attach transaction attachment \"id:%d\" to transaction \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.PictureId, transaction.TransactionId, uid, err.Error())
			response.AttachError = err.Error()
			response.NextStep = "The attachment \"" + attachmentId + "\" is saved but unattached. Retry with update_transaction (add_attachment_ids), or remove it with delete_transaction_attachment."
		} else {
			log.Infof(c, "[upload_transaction_attachment_tool_handler.Handle] user \"uid:%d\" has attached transaction attachment \"id:%d\" to transaction \"id:%d\" successfully", uid, pictureInfo.PictureId, transaction.TransactionId)
			response.Attached = true
		}
	} else {
		response.NextStep = "Pass \"" + attachmentId + "\" in attachment_ids of add_transaction or add_attachment_ids of update_transaction to attach it to a transaction."
	}

	savedPictureInfo, err := services.GetTransactionPictureService().GetPictureInfoByPictureId(c, uid, pictureInfo.PictureId)

	if err != nil {
		log.Warnf(c, "[upload_transaction_attachment_tool_handler.Handle] failed to get transaction attachment \"id:%d\" after uploaded for user \"uid:%d\", because %s", pictureInfo.PictureId, uid, err.Error())
		savedPictureInfo = pictureInfo
	}

	storedData, contentAvailable, err := mcpAttachmentReadContent(c, uid, savedPictureInfo, services)

	if err != nil {
		contentAvailable = false
		storedData = nil
	}

	response.Attachment = mcpAttachmentBuildInfo(savedPictureInfo, storedData, contentAvailable)
	response.Verified = contentAvailable && response.Attachment.SizeBytes == response.UploadedSizeBytes && response.Attachment.SHA256 == response.UploadedSHA256

	return newMCPJSONTextResult(response)
}
