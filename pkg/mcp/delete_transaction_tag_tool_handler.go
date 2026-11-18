package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPDeleteTransactionTagRequest represents all parameters of the delete transaction tag request
type MCPDeleteTransactionTagRequest struct {
	TagId   string `json:"tag_id" jsonschema_description:"ID of the tag to delete (names are not accepted for deletion)"`
	Confirm bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion"`
}

// MCPDeleteTransactionTagResponse represents the response structure for delete transaction tag
type MCPDeleteTransactionTagResponse struct {
	Success bool                   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Status  string                 `json:"status" jsonschema:"enum=deleted" jsonschema_description:"Result status"`
	Tag     *MCPTransactionTagInfo `json:"tag" jsonschema_description:"The deleted tag"`
}

type mcpDeleteTransactionTagToolHandler struct{}

var MCPDeleteTransactionTagToolHandler = &mcpDeleteTransactionTagToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionTagToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION_TAG
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionTagToolHandler) Name() string {
	return "delete_transaction_tag"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionTagToolHandler) Description() string {
	return "Permanently delete a transaction tag in ezBookkeeping (cannot be restored through MCP). As in the application, a tag used by any transaction, transaction template or active scheduled transaction cannot be deleted; hide it with update_transaction_tag instead. Requires confirm=true."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionTagToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionTagToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionTagToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionTagToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTagRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionTagToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTagResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionTagToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDeleteTransactionTagRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	tagId, err := parseMCPRequiredId("tag_id", request.TagId)

	if err != nil {
		return nil, nil, err
	}

	if !request.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting a tag is permanent, set \"confirm\" to true to delete it")
	}

	uid := user.Uid
	tag, err := services.GetTransactionTagService().GetTagByTagId(c, uid, tagId)

	if err != nil {
		log.Warnf(c, "[delete_transaction_tag_tool_handler.Handle] failed to get tag \"id:%d\" for user \"uid:%d\", because %s", tagId, uid, err.Error())
		return nil, nil, convertMCPTransactionTagServiceError(err, "tag_id", request.TagId)
	}

	var tagGroupsMap map[int64]*models.TransactionTagGroup

	if tag.TagGroupId > 0 {
		tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

		if err != nil {
			log.Warnf(c, "[delete_transaction_tag_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		} else {
			tagGroupsMap = getMCPTransactionTagGroupsMap(tagGroups)
		}
	}

	err = services.GetTransactionTagService().DeleteTag(c, uid, tagId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_tag_tool_handler.Handle] failed to delete tag \"id:%d\" for user \"uid:%d\", because %s", tagId, uid, err.Error())
		return nil, nil, convertMCPTransactionTagServiceError(err, "tag_id", request.TagId)
	}

	log.Infof(c, "[delete_transaction_tag_tool_handler.Handle] user \"uid:%d\" has deleted tag \"id:%d\"", uid, tagId)

	return newMCPJSONTextResult(&MCPDeleteTransactionTagResponse{
		Success: true,
		Status:  MCPWriteStatusDeleted,
		Tag:     newMCPTransactionTagInfo(tag, tagGroupsMap),
	})
}
