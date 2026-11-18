package mcp

import (
	"encoding/base64"
	"encoding/json/v2"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPDownloadTransactionAttachmentRequest represents all parameters of the download transaction attachment request
type MCPDownloadTransactionAttachmentRequest struct {
	AttachmentId string `json:"attachment_id" jsonschema_description:"Attachment ID"`
}

// MCPDownloadTransactionAttachmentResponse represents the structured response for download transaction attachment (the file content is returned as an embedded resource)
type MCPDownloadTransactionAttachmentResponse struct {
	ResourceUri string                        `json:"resource_uri" jsonschema_description:"URI of the embedded resource which contains the base64 encoded file content"`
	Attachment  *MCPTransactionAttachmentInfo `json:"attachment" jsonschema_description:"Attachment information with size, MIME type and SHA-256 checksum of the returned content"`
}

type mcpDownloadTransactionAttachmentToolHandler struct{}

var MCPDownloadTransactionAttachmentToolHandler = &mcpDownloadTransactionAttachmentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDownloadTransactionAttachmentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DOWNLOAD_TRANSACTION_ATTACHMENT
}

// Name returns the name of the MCP tool
func (h *mcpDownloadTransactionAttachmentToolHandler) Name() string {
	return "download_transaction_attachment"
}

// Description returns the description of the MCP tool
func (h *mcpDownloadTransactionAttachmentToolHandler) Description() string {
	return "Download the content of a transaction attachment (receipt picture or PDF document). The file is returned as an embedded resource (base64 blob with its MIME type), followed by an embedded JSON resource with its ID, transaction association, size and SHA-256 checksum. Files larger than max_file_size_bytes of get_server_capabilities are not returned. " + mcpAttachmentLimitationNote
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDownloadTransactionAttachmentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDownloadTransactionAttachmentToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDownloadTransactionAttachmentToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDownloadTransactionAttachmentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDownloadTransactionAttachmentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDownloadTransactionAttachmentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDownloadTransactionAttachmentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDownloadTransactionAttachmentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPEmbeddedResource, error) {
	request, err := parseMCPToolArguments[MCPDownloadTransactionAttachmentRequest](callToolReq, false)

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

	data, contentAvailable, err := mcpAttachmentReadContent(c, uid, pictureInfo, services)

	if err != nil {
		return nil, nil, err
	}

	if !contentAvailable {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotFound, "attachment_id", "the stored file of attachment \""+request.AttachmentId+"\" cannot be found on the server storage")
	}

	if maxFileSize := mcpAttachmentGetMaxFileSize(currentConfig); maxFileSize > 0 && int64(len(data)) > maxFileSize {
		return nil, nil, NewMCPTooLargeError("attachment_id", maxFileSize, int64(len(data)))
	}

	resourceUri := mcpAttachmentGetResourceUri(pictureInfo)
	info := mcpAttachmentBuildInfo(pictureInfo, data, true)
	response := &MCPDownloadTransactionAttachmentResponse{
		ResourceUri: resourceUri,
		Attachment:  info,
	}

	metadata, err := json.Marshal(response)

	if err != nil {
		return nil, nil, err
	}

	return response, []*MCPEmbeddedResource{
		NewMCPEmbeddedResource(&MCPBlobResourceContents{
			URI:      resourceUri,
			Blob:     base64.StdEncoding.EncodeToString(data),
			MimeType: info.ContentType,
		}),
		NewMCPEmbeddedResource(&MCPTextResourceContents{
			URI:      resourceUri + "/metadata",
			Text:     string(metadata),
			MimeType: "application/json",
		}),
	}, nil
}
