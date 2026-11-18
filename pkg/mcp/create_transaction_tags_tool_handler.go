package mcp

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPCreateTransactionTagsItemRequest represents one tag to create in the create transaction tags request
type MCPCreateTransactionTagsItemRequest struct {
	Name         string `json:"name" jsonschema_description:"Tag name (tag names are unique for the user)"`
	TagGroupId   string `json:"tag_group_id,omitempty" jsonschema_description:"Tag group ID (optional, \"0\" or empty means no tag group)"`
	TagGroupName string `json:"tag_group_name,omitempty" jsonschema_description:"Tag group name, used when tag_group_id is not set (optional, an ambiguous name returns an error with candidates)"`
}

// MCPCreateTransactionTagsRequest represents all parameters of the create transaction tags request
type MCPCreateTransactionTagsRequest struct {
	Tags         []*MCPCreateTransactionTagsItemRequest `json:"tags" jsonschema_description:"Tags to create (maximum 200 items)"`
	SkipExisting *bool                                  `json:"skip_existing,omitempty" jsonschema_description:"If true (default), a tag whose name already exists is reused (status existing) instead of failing, and a repeated name in the same request is skipped, so rerunning the same request creates no duplicates"`
	DryRun       bool                                   `json:"dry_run,omitempty" jsonschema_description:"If true, only validates and previews the result without saving anything (optional)"`
}

// MCPCreateTransactionTagsItemResult represents the result of one tag in the create transaction tags response
type MCPCreateTransactionTagsItemResult struct {
	Index   int                    `json:"index" jsonschema_description:"Index of the item in the request"`
	Name    string                 `json:"name" jsonschema_description:"Tag name in the request"`
	Status  string                 `json:"status" jsonschema:"enum=created,enum=existing,enum=skipped,enum=would_be_created,enum=failed" jsonschema_description:"Result status of the item"`
	Id      string                 `json:"id,omitempty" jsonschema_description:"ID of the created or reused tag (empty for would_be_created and failed)"`
	Tag     *MCPTransactionTagInfo `json:"tag,omitempty" jsonschema_description:"Persisted (or existing) tag"`
	Message string                 `json:"message,omitempty" jsonschema_description:"Additional information of the item"`
	Error   *MCPToolError          `json:"error,omitempty" jsonschema_description:"Validation error of the item"`
}

// MCPCreateTransactionTagsResponse represents the response structure for create transaction tags
type MCPCreateTransactionTagsResponse struct {
	Success       bool                                  `json:"success" jsonschema_description:"Indicates whether all items are valid (and saved if not dry run)"`
	DryRun        bool                                  `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Committed     bool                                  `json:"committed" jsonschema_description:"Indicates whether new tags have been saved; the batch is atomic, if any item is invalid nothing is saved"`
	CreatedCount  int                                   `json:"created_count" jsonschema_description:"Count of tags created (or would be created in dry run)"`
	ExistingCount int                                   `json:"existing_count" jsonschema_description:"Count of items reusing an existing tag"`
	SkippedCount  int                                   `json:"skipped_count" jsonschema_description:"Count of items skipped because the same name appears earlier in the request"`
	FailedCount   int                                   `json:"failed_count" jsonschema_description:"Count of invalid items"`
	Results       []*MCPCreateTransactionTagsItemResult `json:"results" jsonschema_description:"Per-item results in request order"`
}

// mcpTransactionTagCreationPlanItem represents the planned operation of one tag in the create transaction tags request
type mcpTransactionTagCreationPlanItem struct {
	name        string
	tagGroupId  int64
	existingTag *models.TransactionTag
	duplicateOf int
	err         error
}

type mcpCreateTransactionTagsToolHandler struct{}

var MCPCreateTransactionTagsToolHandler = &mcpCreateTransactionTagsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpCreateTransactionTagsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_CREATE_TRANSACTION_TAGS
}

// Name returns the name of the MCP tool
func (h *mcpCreateTransactionTagsToolHandler) Name() string {
	return "create_transaction_tags"
}

// Description returns the description of the MCP tool
func (h *mcpCreateTransactionTagsToolHandler) Description() string {
	return "Create one or more transaction tags in ezBookkeeping. Existing tags with the same name are reused by default (skip_existing), so reruns do not create duplicates. The batch is atomic: all items are validated first and if any item is invalid nothing is saved. Supports dry_run preview with per-item results."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpCreateTransactionTagsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpCreateTransactionTagsToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpCreateTransactionTagsToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpCreateTransactionTagsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTagsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpCreateTransactionTagsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTagsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpCreateTransactionTagsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPCreateTransactionTagsRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if len(request.Tags) < 1 {
		return nil, nil, NewMCPRequiredFieldError("tags")
	}

	if len(request.Tags) > mcpMaximumBatchItemsCount {
		return nil, nil, NewMCPTooManyItemsError("tags", mcpMaximumBatchItemsCount, len(request.Tags))
	}

	skipExisting := true

	if request.SkipExisting != nil {
		skipExisting = *request.SkipExisting
	}

	uid := user.Uid
	tagGroups, err := services.GetTransactionTagGroupService().GetAllTagGroupsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[create_transaction_tags_tool_handler.Handle] failed to get tag groups for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	existingTags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[create_transaction_tags_tool_handler.Handle] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	tagGroupsMap := getMCPTransactionTagGroupsMap(tagGroups)
	plans, failedCount := planMCPTransactionTagsCreation(request.Tags, existingTags, tagGroups, skipExisting)

	if failedCount > 0 || request.DryRun {
		return newMCPJSONTextResult(h.createResponse(plans, nil, tagGroupsMap, request.DryRun, false))
	}

	newTags := make(map[int]*models.TransactionTag)
	newTagList := make([]*models.TransactionTag, 0, len(plans))
	maxDisplayOrders := make(map[int64]int32)

	for i := 0; i < len(plans); i++ {
		plan := plans[i]

		if plan.existingTag != nil || plan.duplicateOf >= 0 {
			continue
		}

		maxDisplayOrder, exists := maxDisplayOrders[plan.tagGroupId]

		if !exists {
			maxDisplayOrder, err = services.GetTransactionTagService().GetMaxDisplayOrder(c, uid, plan.tagGroupId)

			if err != nil {
				log.Errorf(c, "[create_transaction_tags_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
				return nil, nil, err
			}
		}

		maxDisplayOrder++
		maxDisplayOrders[plan.tagGroupId] = maxDisplayOrder

		tag := &models.TransactionTag{
			Uid:          uid,
			Name:         plan.name,
			TagGroupId:   plan.tagGroupId,
			DisplayOrder: maxDisplayOrder,
		}

		newTags[i] = tag
		newTagList = append(newTagList, tag)
	}

	if len(newTagList) > 0 {
		err = services.GetTransactionTagService().CreateTags(c, uid, newTagList, true)

		if err != nil {
			log.Errorf(c, "[create_transaction_tags_tool_handler.Handle] failed to create tags for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, convertMCPTransactionTagServiceError(err, "tags", "")
		}

		log.Infof(c, "[create_transaction_tags_tool_handler.Handle] user \"uid:%d\" has created %d tags successfully", uid, len(newTagList))
	}

	return newMCPJSONTextResult(h.createResponse(plans, newTags, tagGroupsMap, false, true))
}

func (h *mcpCreateTransactionTagsToolHandler) createResponse(plans []*mcpTransactionTagCreationPlanItem, newTags map[int]*models.TransactionTag, tagGroupsMap map[int64]*models.TransactionTagGroup, dryRun bool, committed bool) *MCPCreateTransactionTagsResponse {
	response := &MCPCreateTransactionTagsResponse{
		DryRun:    dryRun,
		Committed: committed,
		Results:   make([]*MCPCreateTransactionTagsItemResult, len(plans)),
	}

	for i := 0; i < len(plans); i++ {
		plan := plans[i]
		result := &MCPCreateTransactionTagsItemResult{
			Index: i,
			Name:  plan.name,
		}

		if plan.err != nil {
			result.Status = MCPWriteStatusFailed
			var toolErr *MCPToolError

			if errors.As(plan.err, &toolErr) {
				result.Error = toolErr
			} else {
				result.Message = plan.err.Error()
			}

			response.FailedCount++
		} else if plan.duplicateOf >= 0 {
			result.Status = MCPWriteStatusSkipped
			result.Message = fmt.Sprintf("same tag as item #%d in this request", plan.duplicateOf)
			response.SkippedCount++
		} else if plan.existingTag != nil {
			result.Status = MCPWriteStatusExisting
			result.Id = utils.Int64ToString(plan.existingTag.TagId)
			result.Tag = newMCPTransactionTagInfo(plan.existingTag, tagGroupsMap)

			if plan.existingTag.TagGroupId != plan.tagGroupId {
				result.Message = "an existing tag with the same name is reused, but it belongs to a different tag group"
			}

			response.ExistingCount++
		} else if tag, exists := newTags[i]; exists && tag != nil && committed {
			result.Status = MCPWriteStatusCreated
			result.Id = utils.Int64ToString(tag.TagId)
			result.Tag = newMCPTransactionTagInfo(tag, tagGroupsMap)
			response.CreatedCount++
		} else {
			result.Status = MCPWriteStatusWouldBeCreated
			response.CreatedCount++
		}

		response.Results[i] = result
	}

	for i := 0; i < len(plans); i++ {
		if plans[i].err == nil && plans[i].duplicateOf >= 0 && plans[i].duplicateOf < len(response.Results) {
			response.Results[i].Id = response.Results[plans[i].duplicateOf].Id
		}
	}

	response.Success = response.FailedCount == 0
	return response
}

// planMCPTransactionTagsCreation validates all items and decides whether each tag should be created, reused or skipped, returns the plans and the count of invalid items
func planMCPTransactionTagsCreation(items []*MCPCreateTransactionTagsItemRequest, existingTags []*models.TransactionTag, tagGroups []*models.TransactionTagGroup, skipExisting bool) ([]*mcpTransactionTagCreationPlanItem, int) {
	plans := make([]*mcpTransactionTagCreationPlanItem, len(items))
	firstIndexes := make(map[string]int, len(items))
	failedCount := 0

	for i := 0; i < len(items); i++ {
		item := items[i]
		plan := &mcpTransactionTagCreationPlanItem{
			duplicateOf: -1,
		}
		plans[i] = plan

		if item == nil {
			plan.err = NewMCPRequiredFieldError(fmt.Sprintf("tags[%d]", i))
			failedCount++
			continue
		}

		plan.name = item.Name
		nameField := fmt.Sprintf("tags[%d].name", i)

		if err := checkMCPStringLength(nameField, item.Name, true, mcpTagNameMaxLength); err != nil {
			plan.err = err
			failedCount++
			continue
		}

		tagGroupId, _, err := resolveMCPTransactionTagGroupId(tagGroups, fmt.Sprintf("tags[%d].tag_group_id", i), item.TagGroupId, fmt.Sprintf("tags[%d].tag_group_name", i), item.TagGroupName)

		if err != nil {
			plan.err = err
			failedCount++
			continue
		}

		plan.tagGroupId = tagGroupId

		if firstIndex, exists := firstIndexes[item.Name]; exists {
			if plans[firstIndex].tagGroupId != tagGroupId {
				plan.err = NewMCPInvalidFieldError(nameField, item.Name, fmt.Sprintf("tag name is the same as item #%d but the tag group is different (tag names must be unique)", firstIndex))
				failedCount++
				continue
			}

			if !skipExisting {
				plan.err = NewMCPInvalidFieldError(nameField, item.Name, fmt.Sprintf("tag name is the same as item #%d (tag names must be unique)", firstIndex))
				failedCount++
				continue
			}

			plan.duplicateOf = firstIndex
			continue
		}

		firstIndexes[item.Name] = i
		sameNameTags := findMCPTransactionTagsByExactName(existingTags, item.Name)

		if len(sameNameTags) > 0 {
			if !skipExisting {
				toolErr := NewMCPToolError(MCPToolErrorCodeNotAllowed, nameField, "a tag with the same name already exists (tag names must be unique), set skip_existing to true to reuse it")
				toolErr.Value = item.Name
				toolErr.Candidates = newMCPTransactionTagResolver(getMCPTransactionTagGroupsMap(tagGroups)).GetCandidates(sameNameTags)
				plan.err = toolErr
				failedCount++
				continue
			}

			plan.existingTag = sameNameTags[0]
		}
	}

	return plans, failedCount
}
