package mcp

import (
	"errors"
	"sort"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpTransactionTagNoGroupId is the textual tag group id which means "no tag group"
const mcpTransactionTagNoGroupId = "0"

// MCPTransactionTagInfo defines the structure of transaction tag information
type MCPTransactionTagInfo struct {
	Id           string `json:"id" jsonschema_description:"Stable tag ID"`
	Name         string `json:"name" jsonschema_description:"Tag name (unique for the user)"`
	TagGroupId   string `json:"tag_group_id" jsonschema_description:"Tag group ID (\"0\" means the tag does not belong to any tag group)"`
	TagGroupName string `json:"tag_group_name,omitempty" jsonschema_description:"Tag group name (empty if the tag does not belong to any tag group)"`
	Hidden       bool   `json:"hidden" jsonschema_description:"Whether the tag is hidden"`
	DisplayOrder int32  `json:"display_order" jsonschema_description:"Display order of the tag inside its tag group"`
}

// MCPTransactionTagGroupInfo defines the structure of transaction tag group information
type MCPTransactionTagGroupInfo struct {
	Id           string `json:"id" jsonschema_description:"Stable tag group ID"`
	Name         string `json:"name" jsonschema_description:"Tag group name"`
	DisplayOrder int32  `json:"display_order" jsonschema_description:"Display order of the tag group"`
}

// newMCPTransactionTagInfo returns the MCP view-object of the transaction tag
func newMCPTransactionTagInfo(tag *models.TransactionTag, tagGroupsMap map[int64]*models.TransactionTagGroup) *MCPTransactionTagInfo {
	info := &MCPTransactionTagInfo{
		Id:           utils.Int64ToString(tag.TagId),
		Name:         tag.Name,
		TagGroupId:   utils.Int64ToString(tag.TagGroupId),
		Hidden:       tag.Hidden,
		DisplayOrder: tag.DisplayOrder,
	}

	if tag.TagGroupId > 0 && tagGroupsMap != nil {
		if tagGroup, exists := tagGroupsMap[tag.TagGroupId]; exists && tagGroup != nil {
			info.TagGroupName = tagGroup.Name
		}
	}

	return info
}

// newMCPTransactionTagGroupInfo returns the MCP view-object of the transaction tag group
func newMCPTransactionTagGroupInfo(tagGroup *models.TransactionTagGroup) *MCPTransactionTagGroupInfo {
	return &MCPTransactionTagGroupInfo{
		Id:           utils.Int64ToString(tagGroup.TagGroupId),
		Name:         tagGroup.Name,
		DisplayOrder: tagGroup.DisplayOrder,
	}
}

// getMCPTransactionTagGroupsMap returns the map of tag groups by tag group id
func getMCPTransactionTagGroupsMap(tagGroups []*models.TransactionTagGroup) map[int64]*models.TransactionTagGroup {
	tagGroupsMap := make(map[int64]*models.TransactionTagGroup, len(tagGroups))

	for i := 0; i < len(tagGroups); i++ {
		tagGroupsMap[tagGroups[i].TagGroupId] = tagGroups[i]
	}

	return tagGroupsMap
}

// newMCPTransactionTagResolver returns the resolver of transaction tags by id or name
func newMCPTransactionTagResolver(tagGroupsMap map[int64]*models.TransactionTagGroup) *mcpEntityResolver[*models.TransactionTag] {
	return newMCPEntityResolver("tag",
		func(tag *models.TransactionTag) int64 {
			return tag.TagId
		},
		func(tag *models.TransactionTag) string {
			return tag.Name
		},
		func(tag *models.TransactionTag) string {
			description := ""

			if tagGroup, exists := tagGroupsMap[tag.TagGroupId]; exists && tagGroup != nil {
				description = "tag group: " + tagGroup.Name
			}

			if tag.Hidden {
				if description != "" {
					description += ", "
				}

				description += "hidden"
			}

			return description
		})
}

// newMCPTransactionTagGroupResolver returns the resolver of transaction tag groups by id or name
func newMCPTransactionTagGroupResolver() *mcpEntityResolver[*models.TransactionTagGroup] {
	return newMCPEntityResolver("tag group",
		func(tagGroup *models.TransactionTagGroup) int64 {
			return tagGroup.TagGroupId
		},
		func(tagGroup *models.TransactionTagGroup) string {
			return tagGroup.Name
		},
		nil)
}

// resolveMCPTransactionTagGroupId resolves the tag group id by textual tag group id ("0" means no tag group) or tag group name,
// returns (0, false, nil) if neither is specified
func resolveMCPTransactionTagGroupId(tagGroups []*models.TransactionTagGroup, idField string, id string, nameField string, name string) (int64, bool, error) {
	id = strings.TrimSpace(id)

	if id == mcpTransactionTagNoGroupId {
		if name != "" {
			return 0, false, NewMCPInvalidFieldError(nameField, name, "\""+nameField+"\" cannot be set when \""+idField+"\" is \"0\" (no tag group)")
		}

		return 0, true, nil
	}

	if id == "" && name == "" {
		return 0, false, nil
	}

	tagGroup, err := newMCPTransactionTagGroupResolver().Resolve(tagGroups, idField, id, nameField, name)

	if err != nil {
		return 0, false, err
	}

	return tagGroup.TagGroupId, true, nil
}

// findMCPTransactionTagsByExactName returns all tags whose name exactly equals to the given name (the same rule as the tag name uniqueness check of the application)
func findMCPTransactionTagsByExactName(tags []*models.TransactionTag, name string) []*models.TransactionTag {
	result := make([]*models.TransactionTag, 0)

	for i := 0; i < len(tags); i++ {
		if tags[i].Name == name {
			result = append(result, tags[i])
		}
	}

	return result
}

// sortMCPTransactionTags sorts the tags by the display order of their tag groups (tags without tag group first) and then by their own display order
func sortMCPTransactionTags(tags []*models.TransactionTag, tagGroupsMap map[int64]*models.TransactionTagGroup) {
	getGroupOrder := func(tag *models.TransactionTag) int64 {
		if tag.TagGroupId <= 0 {
			return -1
		}

		if tagGroup, exists := tagGroupsMap[tag.TagGroupId]; exists && tagGroup != nil {
			return int64(tagGroup.DisplayOrder)
		}

		return int64(1) << 40
	}

	sort.SliceStable(tags, func(i, j int) bool {
		groupOrderI := getGroupOrder(tags[i])
		groupOrderJ := getGroupOrder(tags[j])

		if groupOrderI != groupOrderJ {
			return groupOrderI < groupOrderJ
		}

		if tags[i].TagGroupId != tags[j].TagGroupId {
			return tags[i].TagGroupId < tags[j].TagGroupId
		}

		return tags[i].DisplayOrder < tags[j].DisplayOrder
	})
}

// getMCPTransactionTagReorderedDisplayOrders returns the new display orders for the given ids in requested order.
// The listed items are rearranged among their current display order slots, so items not listed keep their positions.
func getMCPTransactionTagReorderedDisplayOrders(field string, requestedIds []string, currentDisplayOrders map[int64]int32, entityName string) ([]int64, []int32, error) {
	if len(requestedIds) < 1 {
		return nil, nil, NewMCPRequiredFieldError(field)
	}

	if len(requestedIds) > mcpMaximumBatchItemsCount {
		return nil, nil, NewMCPTooManyItemsError(field, mcpMaximumBatchItemsCount, len(requestedIds))
	}

	ids := make([]int64, 0, len(requestedIds))
	slots := make([]int32, 0, len(requestedIds))
	visited := make(map[int64]bool, len(requestedIds))

	for i := 0; i < len(requestedIds); i++ {
		id, err := parseMCPRequiredId(field, requestedIds[i])

		if err != nil {
			return nil, nil, err
		}

		if visited[id] {
			return nil, nil, NewMCPInvalidFieldError(field, requestedIds[i], "\""+field+"\" contains duplicate ID \""+requestedIds[i]+"\"")
		}

		displayOrder, exists := currentDisplayOrders[id]

		if !exists {
			return nil, nil, NewMCPNotFoundError(field, entityName, requestedIds[i])
		}

		visited[id] = true
		ids = append(ids, id)
		slots = append(slots, displayOrder)
	}

	sort.Slice(slots, func(i, j int) bool {
		return slots[i] < slots[j]
	})

	// make sure the new display orders are strictly increasing even if some current display orders are duplicated
	for i := 1; i < len(slots); i++ {
		if slots[i] <= slots[i-1] {
			slots[i] = slots[i-1] + 1
		}
	}

	return ids, slots, nil
}

// convertMCPTransactionTagServiceError converts the user-correctable errors returned by tag services to structured MCP tool errors
func convertMCPTransactionTagServiceError(err error, field string, value string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, errs.ErrTransactionTagNameAlreadyExists) {
		toolErr := NewMCPToolError(MCPToolErrorCodeNotAllowed, field, "a tag with the same name already exists (tag names must be unique)")
		toolErr.Value = value
		return toolErr
	} else if errors.Is(err, errs.ErrTransactionTagInUseCannotBeDeleted) {
		toolErr := NewMCPToolError(MCPToolErrorCodeNotAllowed, field, "the tag is used by transactions or by transaction templates / active scheduled transactions and cannot be deleted, remove it from them first or hide it instead")
		toolErr.Value = value
		return toolErr
	} else if errors.Is(err, errs.ErrTransactionTagGroupInUseCannotBeDeleted) {
		toolErr := NewMCPToolError(MCPToolErrorCodeNotAllowed, field, "the tag group still contains tags and cannot be deleted, move or delete these tags first")
		toolErr.Value = value
		return toolErr
	} else if errors.Is(err, errs.ErrTransactionTagNotFound) {
		return NewMCPNotFoundError(field, "tag", value)
	} else if errors.Is(err, errs.ErrTransactionTagGroupNotFound) {
		return NewMCPNotFoundError(field, "tag group", value)
	}

	return err
}
