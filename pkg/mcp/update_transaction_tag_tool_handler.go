package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPUpdateTransactionTagRequest represents all parameters of the update transaction tag request
type MCPUpdateTransactionTagRequest struct {
	TagId        string  `json:"tag_id,omitempty" jsonschema_description:"ID of the tag to update (preferred)"`
	TagName      string  `json:"tag_name,omitempty" jsonschema_description:"Current name of the tag to update, used when tag_id is not set"`
	Name         *string `json:"name,omitempty" jsonschema_description:"New tag name (optional, tag names must be unique)"`
	TagGroupId   *string `json:"tag_group_id,omitempty" jsonschema_description:"New tag group ID (optional, \"0\" moves the tag out of any tag group, omit to keep the current tag group)"`
	TagGroupName *string `json:"tag_group_name,omitempty" jsonschema_description:"New tag group name, used when tag_group_id is not set (optional, omit to keep the current tag group)"`
	Hidden       *bool   `json:"hidden,omitempty" jsonschema_description:"Whether the tag is hidden (optional)"`
	DryRun       bool    `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPUpdateTransactionTagResponse represents the response structure for update transaction tag
type MCPUpdateTransactionTagResponse struct {
	Success       bool                   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun        bool                   `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Status        string                 `json:"status" jsonschema:"enum=updated,enum=would_be_updated,enum=unchanged" jsonschema_description:"Result status"`
	ChangedFields []string               `json:"changed_fields,omitempty" jsonschema_description:"Fields which are (or would be) changed"`
	Tag           *MCPTransactionTagInfo `json:"tag" jsonschema_description:"Tag after the update (persisted values if not dry run)"`
}

type mcpUpdateTransactionTagToolHandler struct{}

var MCPUpdateTransactionTagToolHandler = &mcpUpdateTransactionTagToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateTransactionTagToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_TRANSACTION_TAG
}

// Name returns the name of the MCP tool
func (h *mcpUpdateTransactionTagToolHandler) Name() string {
	return "update_transaction_tag"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateTransactionTagToolHandler) Description() string {
	return "Update an existing transaction tag in ezBookkeeping: rename it, move it to another tag group, or hide/unhide it. Only the specified fields are changed."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateTransactionTagToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateTransactionTagToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateTransactionTagToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateTransactionTagToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTagRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateTransactionTagToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTagResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateTransactionTagToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUpdateTransactionTagRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[update_transaction_tag_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	allTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[update_transaction_tag_tool_handler.Handle] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagGroupsMap := getMCPTransactionTagGroupsMap(tagGroups)
	tag, err := newMCPTransactionTagResolver(tagGroupsMap).Resolve(allTags, "tag_id", request.TagId, "tag_name", request.TagName)

	if err != nil {
		return nil, nil, err
	}

	newTag := &models.TransactionTag{
		TagId:        tag.TagId,
		Uid:          uid,
		Name:         tag.Name,
		TagGroupId:   tag.TagGroupId,
		DisplayOrder: tag.DisplayOrder,
		Hidden:       tag.Hidden,
	}

	changedFields := make([]string, 0, 3)

	if request.Name != nil && *request.Name != tag.Name {
		if err := checkMCPStringLength("name", *request.Name, true, mcpTagNameMaxLength); err != nil {
			return nil, nil, err
		}

		sameNameTags := findMCPTransactionTagsByExactName(allTags, *request.Name)

		if len(sameNameTags) > 0 {
			toolErr := NewMCPToolError(MCPToolErrorCodeNotAllowed, "name", "a tag with the same name already exists (tag names must be unique)")
			toolErr.Value = *request.Name
			toolErr.Candidates = newMCPTransactionTagResolver(tagGroupsMap).GetCandidates(sameNameTags)
			return nil, nil, toolErr
		}

		newTag.Name = *request.Name
		changedFields = append(changedFields, "name")
	}

	tagGroupIdText := ""
	tagGroupNameText := ""

	if request.TagGroupId != nil {
		tagGroupIdText = *request.TagGroupId
	}

	if request.TagGroupName != nil {
		tagGroupNameText = *request.TagGroupName
	}

	newTagGroupId, tagGroupSet, err := resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", tagGroupIdText, "tag_group_name", tagGroupNameText)

	if err != nil {
		return nil, nil, err
	}

	if tagGroupSet && newTagGroupId != tag.TagGroupId {
		newTag.TagGroupId = newTagGroupId
		changedFields = append(changedFields, "tag_group_id")
	}

	hiddenChanged := request.Hidden != nil && *request.Hidden != tag.Hidden

	if hiddenChanged {
		newTag.Hidden = *request.Hidden
		changedFields = append(changedFields, "hidden")
	}

	response := &MCPUpdateTransactionTagResponse{
		Success:       true,
		DryRun:        request.DryRun,
		ChangedFields: changedFields,
	}

	if len(changedFields) < 1 {
		response.Status = MCPWriteStatusUnchanged
		response.Tag = newMCPTransactionTagInfo(tag, tagGroupsMap)
		return newMCPJSONTextResult(response)
	}

	if request.DryRun {
		response.Status = MCPWriteStatusWouldBeUpdated
		response.Tag = newMCPTransactionTagInfo(newTag, tagGroupsMap)
		return newMCPJSONTextResult(response)
	}

	tagNameChanged := newTag.Name != tag.Name

	if tagNameChanged || newTag.TagGroupId != tag.TagGroupId {
		if newTag.TagGroupId != tag.TagGroupId {
			maxOrderId, err := services.GetTransactionTagService().GetMaxDisplayOrder(c, uid, newTag.TagGroupId)

			if err != nil {
				log.Errorf(c, "[update_transaction_tag_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
				return nil, nil, err
			}

			newTag.DisplayOrder = maxOrderId + 1
		}

		err = services.GetTransactionTagService().ModifyTag(c, newTag, tagNameChanged)

		if err != nil {
			log.Errorf(c, "[update_transaction_tag_tool_handler.Handle] failed to update tag \"id:%d\" for user \"uid:%d\", because %s", tag.TagId, uid, err.Error())
			return nil, nil, convertMCPTransactionTagServiceError(err, "name", newTag.Name)
		}

		log.Infof(c, "[update_transaction_tag_tool_handler.Handle] user \"uid:%d\" has updated tag \"id:%d\" successfully", uid, tag.TagId)
	}

	if hiddenChanged {
		err = services.GetTransactionTagService().HideTag(c, uid, []int64{tag.TagId}, newTag.Hidden)

		if err != nil {
			log.Errorf(c, "[update_transaction_tag_tool_handler.Handle] failed to change hidden status of tag \"id:%d\" for user \"uid:%d\", because %s", tag.TagId, uid, err.Error())
			return nil, nil, convertMCPTransactionTagServiceError(err, "tag_id", request.TagId)
		}

		log.Infof(c, "[update_transaction_tag_tool_handler.Handle] user \"uid:%d\" has changed hidden status of tag \"id:%d\" to %t", uid, tag.TagId, newTag.Hidden)
	}

	updatedTag, err := services.GetTransactionTagService().GetTagByTagId(c, uid, tag.TagId)

	if err != nil {
		log.Warnf(c, "[update_transaction_tag_tool_handler.Handle] failed to get updated tag \"id:%d\" for user \"uid:%d\", because %s", tag.TagId, uid, err.Error())
		updatedTag = newTag
	}

	response.Status = MCPWriteStatusUpdated
	response.Tag = newMCPTransactionTagInfo(updatedTag, tagGroupsMap)

	return newMCPJSONTextResult(response)
}
