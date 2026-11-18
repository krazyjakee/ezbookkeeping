package mcp

import (
	"reflect"
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const mcpListUnattachedAttachmentsDefaultLimit = 20
const mcpListUnattachedAttachmentsMaxLimit = 100

// MCPListTransactionAttachmentsRequest represents all parameters of the list transaction attachments request
type MCPListTransactionAttachmentsRequest struct {
	TransactionId string `json:"transaction_id,omitempty" jsonschema_description:"ID of the transaction whose attachments are listed (optional). If not set, the uploaded but unattached attachments are listed (newest first), e.g. to recover from a failed staged upload"`
	Limit         int    `json:"limit,omitempty" jsonschema_description:"Maximum count of unattached attachments to return (optional, default 20, maximum 100, ignored when transaction_id is set)"`
}

// MCPListTransactionAttachmentsResponse represents the response structure for list transaction attachments
type MCPListTransactionAttachmentsResponse struct {
	TransactionId string                          `json:"transaction_id,omitempty" jsonschema_description:"ID of the transaction which holds the attachments (for the incoming side of a transfer, the ID of the transfer)"`
	Unattached    bool                            `json:"unattached,omitzero" jsonschema_description:"Whether the unattached attachments are listed"`
	TotalCount    int                             `json:"total_count" jsonschema_description:"Total count of matched attachments"`
	Attachments   []*MCPTransactionAttachmentInfo `json:"attachments" jsonschema_description:"Attachments with size, MIME type and SHA-256 checksum computed from the stored files"`
}

type mcpListTransactionAttachmentsToolHandler struct{}

var MCPListTransactionAttachmentsToolHandler = &mcpListTransactionAttachmentsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpListTransactionAttachmentsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_LIST_TRANSACTION_ATTACHMENTS
}

// Name returns the name of the MCP tool
func (h *mcpListTransactionAttachmentsToolHandler) Name() string {
	return "list_transaction_attachments"
}

// Description returns the description of the MCP tool
func (h *mcpListTransactionAttachmentsToolHandler) Description() string {
	return "List the attachments (receipt pictures / PDF documents) of a transaction, or the uploaded but unattached attachments when transaction_id is not set. Each attachment includes its ID, transaction association, size, MIME type and SHA-256 checksum computed from the stored file. " + mcpAttachmentLimitationNote
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpListTransactionAttachmentsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpListTransactionAttachmentsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpListTransactionAttachmentsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpListTransactionAttachmentsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionAttachmentsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpListTransactionAttachmentsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionAttachmentsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpListTransactionAttachmentsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPListTransactionAttachmentsRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	response := &MCPListTransactionAttachmentsResponse{}
	var pictureInfos []*models.TransactionPictureInfo

	if request.TransactionId != "" {
		transaction, err := mcpAttachmentGetTransaction(c, uid, "transaction_id", request.TransactionId, services)

		if err != nil {
			return nil, nil, err
		}

		pictureInfos, err = services.GetTransactionPictureService().GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[list_transaction_attachments_tool_handler.Handle] failed to get attachments of transaction \"id:%d\" for user \"uid:%d\", because %s", transaction.TransactionId, uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		response.TransactionId = utils.Int64ToString(transaction.TransactionId)
		response.TotalCount = len(pictureInfos)
	} else {
		limit := request.Limit

		if limit <= 0 {
			limit = mcpListUnattachedAttachmentsDefaultLimit
		} else if limit > mcpListUnattachedAttachmentsMaxLimit {
			return nil, nil, NewMCPToolError(MCPToolErrorCodeOutOfRange, "limit", "\"limit\" must not be greater than 100")
		}

		allPictureInfos, err := services.GetTransactionPictureService().GetAllPictureInfosOfAllTransactions(c, uid)

		if err != nil {
			log.Errorf(c, "[list_transaction_attachments_tool_handler.Handle] failed to get attachments for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		pictureInfos = allPictureInfos[models.TransactionPictureNewPictureTransactionId]
		response.Unattached = true
		response.TotalCount = len(pictureInfos)

		sort.Slice(pictureInfos, func(i, j int) bool {
			return pictureInfos[i].CreatedUnixTime > pictureInfos[j].CreatedUnixTime || (pictureInfos[i].CreatedUnixTime == pictureInfos[j].CreatedUnixTime && pictureInfos[i].PictureId > pictureInfos[j].PictureId)
		})

		if len(pictureInfos) > limit {
			pictureInfos = pictureInfos[:limit]
		}
	}

	response.Attachments = make([]*MCPTransactionAttachmentInfo, 0, len(pictureInfos))

	for i := 0; i < len(pictureInfos); i++ {
		info, err := mcpAttachmentLoadInfo(c, uid, pictureInfos[i], services)

		if err != nil {
			return nil, nil, errs.Or(err, errs.ErrOperationFailed)
		}

		response.Attachments = append(response.Attachments, info)
	}

	return newMCPJSONTextResult(response)
}
