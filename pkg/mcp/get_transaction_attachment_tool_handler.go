package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPGetTransactionAttachmentRequest represents all parameters of the get transaction attachment request
type MCPGetTransactionAttachmentRequest struct {
	AttachmentId string `json:"attachment_id" jsonschema_description:"Attachment ID"`
}

// MCPGetTransactionAttachmentResponse represents the response structure for get transaction attachment
type MCPGetTransactionAttachmentResponse struct {
	Attachment *MCPTransactionAttachmentInfo `json:"attachment" jsonschema_description:"Attachment information with size, MIME type and SHA-256 checksum computed from the stored file"`
}

type mcpGetTransactionAttachmentToolHandler struct{}

var MCPGetTransactionAttachmentToolHandler = &mcpGetTransactionAttachmentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetTransactionAttachmentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_TRANSACTION_ATTACHMENT
}

// Name returns the name of the MCP tool
func (h *mcpGetTransactionAttachmentToolHandler) Name() string {
	return "get_transaction_attachment"
}

// Description returns the description of the MCP tool
func (h *mcpGetTransactionAttachmentToolHandler) Description() string {
	return "Inspect a transaction attachment by ID without downloading it: returns its transaction association, size, MIME type, SHA-256 checksum computed from the stored file and whether the stored file is available. Use it to verify that an uploaded file was persisted. " + mcpAttachmentLimitationNote
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetTransactionAttachmentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetTransactionAttachmentToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetTransactionAttachmentToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetTransactionAttachmentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionAttachmentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetTransactionAttachmentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionAttachmentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetTransactionAttachmentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPGetTransactionAttachmentRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	pictureInfo, err := mcpAttachmentGetPictureInfo(c, user.Uid, "attachment_id", request.AttachmentId, services)

	if err != nil {
		return nil, nil, err
	}

	info, err := mcpAttachmentLoadInfo(c, user.Uid, pictureInfo, services)

	if err != nil {
		return nil, nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return newMCPJSONTextResult(&MCPGetTransactionAttachmentResponse{
		Attachment: info,
	})
}
