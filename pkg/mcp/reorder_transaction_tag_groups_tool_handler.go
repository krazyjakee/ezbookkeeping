package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPReorderTransactionTagGroupsRequest represents all parameters of the reorder transaction tag groups request
type MCPReorderTransactionTagGroupsRequest struct {
	TagGroupIds []string `json:"tag_group_ids" jsonschema_description:"Tag group IDs in the desired display order. Listed tag groups are rearranged among their current positions and tag groups not listed keep their positions, pass all tag group IDs to define the full order"`
	DryRun      bool     `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPReorderTransactionTagGroupsResponse represents the response structure for reorder transaction tag groups
type MCPReorderTransactionTagGroupsResponse struct {
	Success   bool                          `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun    bool                          `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	TagGroups []*MCPTransactionTagGroupInfo `json:"tag_groups" jsonschema_description:"Reordered tag groups with their new display orders"`
}

type mcpReorderTransactionTagGroupsToolHandler struct{}

var MCPReorderTransactionTagGroupsToolHandler = &mcpReorderTransactionTagGroupsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpReorderTransactionTagGroupsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_REORDER_TRANSACTION_TAG_GROUPS
}

// Name returns the name of the MCP tool
func (h *mcpReorderTransactionTagGroupsToolHandler) Name() string {
	return "reorder_transaction_tag_groups"
}

// Description returns the description of the MCP tool
func (h *mcpReorderTransactionTagGroupsToolHandler) Description() string {
	return "Change the display order of transaction tag groups in ezBookkeeping."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpReorderTransactionTagGroupsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpReorderTransactionTagGroupsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpReorderTransactionTagGroupsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpReorderTransactionTagGroupsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionTagGroupsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpReorderTransactionTagGroupsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionTagGroupsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpReorderTransactionTagGroupsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPReorderTransactionTagGroupsRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[reorder_transaction_tag_groups_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagGroupsMap := getMCPTransactionTagGroupsMap(tagGroups)
	currentDisplayOrders := make(map[int64]int32, len(tagGroups))

	for i := 0; i < len(tagGroups); i++ {
		currentDisplayOrders[tagGroups[i].TagGroupId] = tagGroups[i].DisplayOrder
	}

	tagGroupIds, newDisplayOrders, err := getMCPTransactionTagReorderedDisplayOrders("tag_group_ids", request.TagGroupIds, currentDisplayOrders, "tag group")

	if err != nil {
		return nil, nil, err
	}

	toUpdateTagGroups := make([]*models.TransactionTagGroup, len(tagGroupIds))
	response := &MCPReorderTransactionTagGroupsResponse{
		Success:   true,
		DryRun:    request.DryRun,
		TagGroups: make([]*MCPTransactionTagGroupInfo, len(tagGroupIds)),
	}

	for i := 0; i < len(tagGroupIds); i++ {
		toUpdateTagGroups[i] = &models.TransactionTagGroup{
			Uid:          uid,
			TagGroupId:   tagGroupIds[i],
			DisplayOrder: newDisplayOrders[i],
		}

		tagGroup := *tagGroupsMap[tagGroupIds[i]]
		tagGroup.DisplayOrder = newDisplayOrders[i]
		response.TagGroups[i] = newMCPTransactionTagGroupInfo(&tagGroup)
	}

	if !request.DryRun {
		err = services.GetTransactionTagGroupService().ModifyTagGroupDisplayOrders(c, uid, toUpdateTagGroups)

		if err != nil {
			log.Errorf(c, "[reorder_transaction_tag_groups_tool_handler.Handle] failed to move tag groups for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, convertMCPTransactionTagServiceError(err, "tag_group_ids", "")
		}

		log.Infof(c, "[reorder_transaction_tag_groups_tool_handler.Handle] user \"uid:%d\" has moved tag groups", uid)
	}

	return newMCPJSONTextResult(response)
}
