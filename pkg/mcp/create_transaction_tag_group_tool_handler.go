package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPCreateTransactionTagGroupRequest represents all parameters of the create transaction tag group request
type MCPCreateTransactionTagGroupRequest struct {
	Name         string `json:"name" jsonschema_description:"Tag group name"`
	SkipExisting *bool  `json:"skip_existing,omitempty" jsonschema_description:"If true (default), an existing tag group with exactly the same name is reused (status existing) instead of creating another one, so reruns do not create duplicates. If false, a new tag group is always created (the application allows duplicate tag group names)"`
	DryRun       bool   `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPCreateTransactionTagGroupResponse represents the response structure for create transaction tag group
type MCPCreateTransactionTagGroupResponse struct {
	Success  bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun   bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Status   string                      `json:"status" jsonschema:"enum=created,enum=existing,enum=would_be_created" jsonschema_description:"Result status"`
	Id       string                      `json:"id,omitempty" jsonschema_description:"ID of the created or reused tag group (empty for would_be_created)"`
	TagGroup *MCPTransactionTagGroupInfo `json:"tag_group" jsonschema_description:"Created, reused or previewed tag group"`
}

type mcpCreateTransactionTagGroupToolHandler struct{}

var MCPCreateTransactionTagGroupToolHandler = &mcpCreateTransactionTagGroupToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpCreateTransactionTagGroupToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_CREATE_TRANSACTION_TAG_GROUP
}

// Name returns the name of the MCP tool
func (h *mcpCreateTransactionTagGroupToolHandler) Name() string {
	return "create_transaction_tag_group"
}

// Description returns the description of the MCP tool
func (h *mcpCreateTransactionTagGroupToolHandler) Description() string {
	return "Create a transaction tag group in ezBookkeeping, reusing an existing tag group with the same name by default."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpCreateTransactionTagGroupToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpCreateTransactionTagGroupToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpCreateTransactionTagGroupToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpCreateTransactionTagGroupToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTagGroupRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpCreateTransactionTagGroupToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTagGroupResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpCreateTransactionTagGroupToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPCreateTransactionTagGroupRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := checkMCPStringLength("name", request.Name, true, mcpTagGroupNameMaxLength); err != nil {
		return nil, nil, err
	}

	skipExisting := true

	if request.SkipExisting != nil {
		skipExisting = *request.SkipExisting
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[create_transaction_tag_group_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	if skipExisting {
		existingTagGroup, err := findMCPTransactionTagGroupForReuse(tagGroups, request.Name)

		if err != nil {
			return nil, nil, err
		}

		if existingTagGroup != nil {
			return newMCPJSONTextResult(&MCPCreateTransactionTagGroupResponse{
				Success:  true,
				DryRun:   request.DryRun,
				Status:   MCPWriteStatusExisting,
				Id:       utils.Int64ToString(existingTagGroup.TagGroupId),
				TagGroup: newMCPTransactionTagGroupInfo(existingTagGroup),
			})
		}
	}

	maxOrderId, err := services.GetTransactionTagGroupService().GetMaxDisplayOrder(c, uid)

	if err != nil {
		log.Errorf(c, "[create_transaction_tag_group_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagGroup := &models.TransactionTagGroup{
		Uid:          uid,
		Name:         request.Name,
		DisplayOrder: maxOrderId + 1,
	}

	if request.DryRun {
		return newMCPJSONTextResult(&MCPCreateTransactionTagGroupResponse{
			Success:  true,
			DryRun:   true,
			Status:   MCPWriteStatusWouldBeCreated,
			TagGroup: newMCPTransactionTagGroupInfo(tagGroup),
		})
	}

	err = services.GetTransactionTagGroupService().CreateTagGroup(c, tagGroup)

	if err != nil {
		log.Errorf(c, "[create_transaction_tag_group_tool_handler.Handle] failed to create tag group for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[create_transaction_tag_group_tool_handler.Handle] user \"uid:%d\" has created a new tag group \"id:%d\" successfully", uid, tagGroup.TagGroupId)

	return newMCPJSONTextResult(&MCPCreateTransactionTagGroupResponse{
		Success:  true,
		Status:   MCPWriteStatusCreated,
		Id:       utils.Int64ToString(tagGroup.TagGroupId),
		TagGroup: newMCPTransactionTagGroupInfo(tagGroup),
	})
}

// findMCPTransactionTagGroupForReuse returns the only tag group whose name exactly equals to the given name, returns nil if not found, or an ambiguous name error if more than one
func findMCPTransactionTagGroupForReuse(tagGroups []*models.TransactionTagGroup, name string) (*models.TransactionTagGroup, error) {
	matches := make([]*models.TransactionTagGroup, 0)

	for i := 0; i < len(tagGroups); i++ {
		if tagGroups[i].Name == name {
			matches = append(matches, tagGroups[i])
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return nil, NewMCPAmbiguousNameError("name", "tag group", name, newMCPTransactionTagGroupResolver().GetCandidates(matches))
	}

	return nil, nil
}
