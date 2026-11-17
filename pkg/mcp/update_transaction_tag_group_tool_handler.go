package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPUpdateTransactionTagGroupRequest represents all parameters of the update transaction tag group request
type MCPUpdateTransactionTagGroupRequest struct {
	TagGroupId   string `json:"tag_group_id,omitempty" jsonschema_description:"ID of the tag group to update (preferred)"`
	TagGroupName string `json:"tag_group_name,omitempty" jsonschema_description:"Current name of the tag group to update, used when tag_group_id is not set"`
	Name         string `json:"name" jsonschema_description:"New tag group name"`
	DryRun       bool   `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPUpdateTransactionTagGroupResponse represents the response structure for update transaction tag group
type MCPUpdateTransactionTagGroupResponse struct {
	Success  bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun   bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Status   string                      `json:"status" jsonschema:"enum=updated,enum=would_be_updated,enum=unchanged" jsonschema_description:"Result status"`
	TagGroup *MCPTransactionTagGroupInfo `json:"tag_group" jsonschema_description:"Tag group after the update"`
}

type mcpUpdateTransactionTagGroupToolHandler struct{}

var MCPUpdateTransactionTagGroupToolHandler = &mcpUpdateTransactionTagGroupToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateTransactionTagGroupToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_TRANSACTION_TAG_GROUP
}

// Name returns the name of the MCP tool
func (h *mcpUpdateTransactionTagGroupToolHandler) Name() string {
	return "update_transaction_tag_group"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateTransactionTagGroupToolHandler) Description() string {
	return "Rename an existing transaction tag group in ezBookkeeping."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateTransactionTagGroupToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateTransactionTagGroupToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateTransactionTagGroupToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateTransactionTagGroupToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTagGroupRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateTransactionTagGroupToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTagGroupResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateTransactionTagGroupToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUpdateTransactionTagGroupRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := checkMCPStringLength("name", request.Name, true, mcpTagGroupNameMaxLength); err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[update_transaction_tag_group_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagGroup, err := newMCPTransactionTagGroupResolver().Resolve(tagGroups, "tag_group_id", request.TagGroupId, "tag_group_name", request.TagGroupName)

	if err != nil {
		return nil, nil, err
	}

	newTagGroup := &models.TransactionTagGroup{
		TagGroupId:   tagGroup.TagGroupId,
		Uid:          uid,
		Name:         request.Name,
		DisplayOrder: tagGroup.DisplayOrder,
	}

	response := &MCPUpdateTransactionTagGroupResponse{
		Success:  true,
		DryRun:   request.DryRun,
		TagGroup: newMCPTransactionTagGroupInfo(newTagGroup),
	}

	if newTagGroup.Name == tagGroup.Name {
		response.Status = MCPWriteStatusUnchanged
		return newMCPJSONTextResult(response)
	}

	if request.DryRun {
		response.Status = MCPWriteStatusWouldBeUpdated
		return newMCPJSONTextResult(response)
	}

	err = services.GetTransactionTagGroupService().ModifyTagGroup(c, newTagGroup)

	if err != nil {
		log.Errorf(c, "[update_transaction_tag_group_tool_handler.Handle] failed to update tag group \"id:%d\" for user \"uid:%d\", because %s", tagGroup.TagGroupId, uid, err.Error())
		return nil, nil, convertMCPTransactionTagServiceError(err, "tag_group_id", request.TagGroupId)
	}

	log.Infof(c, "[update_transaction_tag_group_tool_handler.Handle] user \"uid:%d\" has updated tag group \"id:%d\" successfully", uid, tagGroup.TagGroupId)

	response.Status = MCPWriteStatusUpdated
	return newMCPJSONTextResult(response)
}
