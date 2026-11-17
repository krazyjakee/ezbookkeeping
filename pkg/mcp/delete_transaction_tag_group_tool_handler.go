package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPDeleteTransactionTagGroupRequest represents all parameters of the delete transaction tag group request
type MCPDeleteTransactionTagGroupRequest struct {
	TagGroupId string `json:"tag_group_id" jsonschema_description:"ID of the tag group to delete (names are not accepted for deletion)"`
	Confirm    bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion"`
}

// MCPDeleteTransactionTagGroupResponse represents the response structure for delete transaction tag group
type MCPDeleteTransactionTagGroupResponse struct {
	Success  bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Status   string                      `json:"status" jsonschema:"enum=deleted" jsonschema_description:"Result status"`
	TagGroup *MCPTransactionTagGroupInfo `json:"tag_group" jsonschema_description:"The deleted tag group"`
}

type mcpDeleteTransactionTagGroupToolHandler struct{}

var MCPDeleteTransactionTagGroupToolHandler = &mcpDeleteTransactionTagGroupToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionTagGroupToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION_TAG_GROUP
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionTagGroupToolHandler) Name() string {
	return "delete_transaction_tag_group"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionTagGroupToolHandler) Description() string {
	return "Permanently delete a transaction tag group in ezBookkeeping (cannot be restored through MCP). As in the application, a tag group which still contains tags (including hidden tags) cannot be deleted; move or delete these tags first. Requires confirm=true."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionTagGroupToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionTagGroupToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionTagGroupToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionTagGroupToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTagGroupRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionTagGroupToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTagGroupResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionTagGroupToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDeleteTransactionTagGroupRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	tagGroupId, err := parseMCPRequiredId("tag_group_id", request.TagGroupId)

	if err != nil {
		return nil, nil, err
	}

	if !request.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting a tag group is permanent, set \"confirm\" to true to delete it")
	}

	uid := user.Uid
	tagGroup, err := services.GetTransactionTagGroupService().GetTagGroupByTagGroupId(c, uid, tagGroupId)

	if err != nil {
		log.Warnf(c, "[delete_transaction_tag_group_tool_handler.Handle] failed to get tag group \"id:%d\" for user \"uid:%d\", because %s", tagGroupId, uid, err.Error())
		return nil, nil, convertMCPTransactionTagServiceError(err, "tag_group_id", request.TagGroupId)
	}

	err = services.GetTransactionTagGroupService().DeleteTagGroup(c, uid, tagGroupId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_tag_group_tool_handler.Handle] failed to delete tag group \"id:%d\" for user \"uid:%d\", because %s", tagGroupId, uid, err.Error())
		return nil, nil, convertMCPTransactionTagServiceError(err, "tag_group_id", request.TagGroupId)
	}

	log.Infof(c, "[delete_transaction_tag_group_tool_handler.Handle] user \"uid:%d\" has deleted tag group \"id:%d\"", uid, tagGroupId)

	return newMCPJSONTextResult(&MCPDeleteTransactionTagGroupResponse{
		Success:  true,
		Status:   MCPWriteStatusDeleted,
		TagGroup: newMCPTransactionTagGroupInfo(tagGroup),
	})
}
