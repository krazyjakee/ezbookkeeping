package mcp

import (
	"reflect"
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPListTransactionTagsRequest represents all parameters of the list transaction tags request
type MCPListTransactionTagsRequest struct {
	IncludeHidden bool `json:"include_hidden,omitempty" jsonschema_description:"If true, hidden tags are also returned (optional, default false)"`
}

// MCPListTransactionTagsResponse represents the response structure for listing transaction tags
type MCPListTransactionTagsResponse struct {
	TagGroups []*MCPTransactionTagGroupInfo `json:"tag_groups" jsonschema_description:"List of all tag groups ordered by display order"`
	Tags      []*MCPTransactionTagInfo      `json:"tags" jsonschema_description:"List of tags ordered by tag group and display order (tags without tag group first)"`
}

type mcpListTransactionTagsToolHandler struct{}

var MCPListTransactionTagsToolHandler = &mcpListTransactionTagsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpListTransactionTagsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_LIST_TRANSACTION_TAGS
}

// Name returns the name of the MCP tool
func (h *mcpListTransactionTagsToolHandler) Name() string {
	return "list_transaction_tags"
}

// Description returns the description of the MCP tool
func (h *mcpListTransactionTagsToolHandler) Description() string {
	return "List transaction tags and tag groups with their stable IDs in ezBookkeeping."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpListTransactionTagsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpListTransactionTagsToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpListTransactionTagsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpListTransactionTagsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionTagsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpListTransactionTagsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionTagsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpListTransactionTagsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPListTransactionTagsRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[list_transaction_tags_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[list_transaction_tags_tool_handler.Handle] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	return newMCPJSONTextResult(h.createResponse(tagGroups, tags, request.IncludeHidden))
}

func (h *mcpListTransactionTagsToolHandler) createResponse(tagGroups []*models.TransactionTagGroup, tags []*models.TransactionTag, includeHidden bool) *MCPListTransactionTagsResponse {
	tagGroupsMap := getMCPTransactionTagGroupsMap(tagGroups)

	sortedTagGroups := make([]*models.TransactionTagGroup, len(tagGroups))
	copy(sortedTagGroups, tagGroups)
	sort.SliceStable(sortedTagGroups, func(i, j int) bool {
		return sortedTagGroups[i].DisplayOrder < sortedTagGroups[j].DisplayOrder
	})

	filteredTags := make([]*models.TransactionTag, 0, len(tags))

	for i := 0; i < len(tags); i++ {
		if tags[i].Hidden && !includeHidden {
			continue
		}

		filteredTags = append(filteredTags, tags[i])
	}

	sortMCPTransactionTags(filteredTags, tagGroupsMap)

	response := &MCPListTransactionTagsResponse{
		TagGroups: make([]*MCPTransactionTagGroupInfo, 0, len(sortedTagGroups)),
		Tags:      make([]*MCPTransactionTagInfo, 0, len(filteredTags)),
	}

	for i := 0; i < len(sortedTagGroups); i++ {
		response.TagGroups = append(response.TagGroups, newMCPTransactionTagGroupInfo(sortedTagGroups[i]))
	}

	for i := 0; i < len(filteredTags); i++ {
		response.Tags = append(response.Tags, newMCPTransactionTagInfo(filteredTags[i], tagGroupsMap))
	}

	return response
}
