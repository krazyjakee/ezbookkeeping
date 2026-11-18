package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPReorderTransactionTagsRequest represents all parameters of the reorder transaction tags request
type MCPReorderTransactionTagsRequest struct {
	TagIds []string `json:"tag_ids" jsonschema_description:"Tag IDs in the desired display order, all tags must belong to the same tag group. Listed tags are rearranged among their current positions and tags not listed keep their positions, pass all tag IDs of the tag group to define the full order"`
	DryRun bool     `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPReorderTransactionTagsResponse represents the response structure for reorder transaction tags
type MCPReorderTransactionTagsResponse struct {
	Success bool                     `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun  bool                     `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Tags    []*MCPTransactionTagInfo `json:"tags" jsonschema_description:"Reordered tags with their new display orders"`
}

type mcpReorderTransactionTagsToolHandler struct{}

var MCPReorderTransactionTagsToolHandler = &mcpReorderTransactionTagsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpReorderTransactionTagsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_REORDER_TRANSACTION_TAGS
}

// Name returns the name of the MCP tool
func (h *mcpReorderTransactionTagsToolHandler) Name() string {
	return "reorder_transaction_tags"
}

// Description returns the description of the MCP tool
func (h *mcpReorderTransactionTagsToolHandler) Description() string {
	return "Change the display order of transaction tags inside one tag group in ezBookkeeping."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpReorderTransactionTagsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpReorderTransactionTagsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpReorderTransactionTagsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpReorderTransactionTagsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionTagsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpReorderTransactionTagsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionTagsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpReorderTransactionTagsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPReorderTransactionTagsRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[reorder_transaction_tags_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	allTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[reorder_transaction_tags_tool_handler.Handle] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagsMap := make(map[int64]*models.TransactionTag, len(allTags))
	currentDisplayOrders := make(map[int64]int32, len(allTags))

	for i := 0; i < len(allTags); i++ {
		tagsMap[allTags[i].TagId] = allTags[i]
		currentDisplayOrders[allTags[i].TagId] = allTags[i].DisplayOrder
	}

	tagIds, newDisplayOrders, err := getMCPTransactionTagReorderedDisplayOrders("tag_ids", request.TagIds, currentDisplayOrders, "tag")

	if err != nil {
		return nil, nil, err
	}

	tagGroupId := tagsMap[tagIds[0]].TagGroupId
	toUpdateTags := make([]*models.TransactionTag, len(tagIds))

	for i := 0; i < len(tagIds); i++ {
		tag := tagsMap[tagIds[i]]

		if tag.TagGroupId != tagGroupId {
			return nil, nil, NewMCPInvalidFieldError("tag_ids", utils.Int64ToString(tag.TagId), "all tags must belong to the same tag group, but tag \""+utils.Int64ToString(tag.TagId)+"\" belongs to a different tag group")
		}

		toUpdateTags[i] = &models.TransactionTag{
			Uid:          uid,
			TagId:        tag.TagId,
			DisplayOrder: newDisplayOrders[i],
		}
	}

	if !request.DryRun {
		err = services.GetTransactionTagService().ModifyTagDisplayOrders(c, uid, toUpdateTags)

		if err != nil {
			log.Errorf(c, "[reorder_transaction_tags_tool_handler.Handle] failed to move tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, convertMCPTransactionTagServiceError(err, "tag_ids", "")
		}

		log.Infof(c, "[reorder_transaction_tags_tool_handler.Handle] user \"uid:%d\" has moved tags", uid)
	}

	tagGroupsMap := getMCPTransactionTagGroupsMap(tagGroups)
	response := &MCPReorderTransactionTagsResponse{
		Success: true,
		DryRun:  request.DryRun,
		Tags:    make([]*MCPTransactionTagInfo, len(tagIds)),
	}

	for i := 0; i < len(tagIds); i++ {
		tag := *tagsMap[tagIds[i]]
		tag.DisplayOrder = newDisplayOrders[i]
		response.Tags[i] = newMCPTransactionTagInfo(&tag, tagGroupsMap)
	}

	return newMCPJSONTextResult(response)
}
