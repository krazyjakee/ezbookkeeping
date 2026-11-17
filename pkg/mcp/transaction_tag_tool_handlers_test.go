package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func getTransactionTagTestTagGroups() []*models.TransactionTagGroup {
	return []*models.TransactionTagGroup{
		{TagGroupId: 1001, Name: "Projects", DisplayOrder: 2},
		{TagGroupId: 1002, Name: "People", DisplayOrder: 1},
		{TagGroupId: 1003, Name: "Duplicated", DisplayOrder: 3},
		{TagGroupId: 1004, Name: "Duplicated", DisplayOrder: 4},
	}
}

func getTransactionTagTestTags() []*models.TransactionTag {
	return []*models.TransactionTag{
		{TagId: 2001, Name: "Travel", TagGroupId: 0, DisplayOrder: 1},
		{TagId: 2002, Name: "Client A", TagGroupId: 1001, DisplayOrder: 1},
		{TagId: 2003, Name: "Alice", TagGroupId: 1002, DisplayOrder: 1},
		{TagId: 2004, Name: "Archived", TagGroupId: 0, DisplayOrder: 2, Hidden: true},
	}
}

func getTransactionTagTestToolError(t *testing.T, err error) *MCPToolError {
	var toolErr *MCPToolError
	assert.True(t, errors.As(err, &toolErr))
	return toolErr
}

func TestTransactionTagPlanCreation_SkipExisting(t *testing.T) {
	items := []*MCPCreateTransactionTagsItemRequest{
		{Name: "Travel"},
		{Name: "Client B", TagGroupName: "Projects"},
		{Name: "Client B", TagGroupId: "1001"},
		{Name: "Bob", TagGroupId: "1002"},
	}

	plans, failedCount := planMCPTransactionTagsCreation(items, getTransactionTagTestTags(), getTransactionTagTestTagGroups(), true)

	assert.Equal(t, 0, failedCount)
	assert.Equal(t, 4, len(plans))

	assert.NotNil(t, plans[0].existingTag)
	assert.Equal(t, int64(2001), plans[0].existingTag.TagId)

	assert.Nil(t, plans[1].existingTag)
	assert.Equal(t, -1, plans[1].duplicateOf)
	assert.Equal(t, int64(1001), plans[1].tagGroupId)

	assert.Equal(t, 1, plans[2].duplicateOf)

	assert.Nil(t, plans[3].existingTag)
	assert.Equal(t, int64(1002), plans[3].tagGroupId)
}

func TestTransactionTagPlanCreation_RerunCreatesNothing(t *testing.T) {
	items := []*MCPCreateTransactionTagsItemRequest{
		{Name: "Travel"},
		{Name: "Client A", TagGroupId: "1001"},
	}

	plans, failedCount := planMCPTransactionTagsCreation(items, getTransactionTagTestTags(), getTransactionTagTestTagGroups(), true)
	handler := &mcpCreateTransactionTagsToolHandler{}
	response := handler.createResponse(plans, nil, getMCPTransactionTagGroupsMap(getTransactionTagTestTagGroups()), true, false)

	assert.Equal(t, 0, failedCount)
	assert.True(t, response.Success)
	assert.Equal(t, 0, response.CreatedCount)
	assert.Equal(t, 2, response.ExistingCount)
	assert.Equal(t, MCPWriteStatusExisting, response.Results[0].Status)
	assert.Equal(t, "2001", response.Results[0].Id)
	assert.Equal(t, "2002", response.Results[1].Id)
}

func TestTransactionTagPlanCreation_NoSkipExisting(t *testing.T) {
	items := []*MCPCreateTransactionTagsItemRequest{
		{Name: "Travel"},
		{Name: "New"},
		{Name: "New"},
	}

	plans, failedCount := planMCPTransactionTagsCreation(items, getTransactionTagTestTags(), getTransactionTagTestTagGroups(), false)

	assert.Equal(t, 2, failedCount)

	toolErr := getTransactionTagTestToolError(t, plans[0].err)
	assert.Equal(t, MCPToolErrorCodeNotAllowed, toolErr.Code)
	assert.Equal(t, "tags[0].name", toolErr.Field)
	assert.Equal(t, 1, len(toolErr.Candidates))
	assert.Equal(t, "2001", toolErr.Candidates[0].Id)

	assert.Nil(t, plans[1].err)

	toolErr = getTransactionTagTestToolError(t, plans[2].err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
}

func TestTransactionTagPlanCreation_ValidationErrors(t *testing.T) {
	longName := ""

	for i := 0; i < mcpTagNameMaxLength+1; i++ {
		longName += "a"
	}

	items := []*MCPCreateTransactionTagsItemRequest{
		{Name: " "},
		{Name: longName},
		{Name: "X", TagGroupName: "Duplicated"},
		{Name: "Y", TagGroupId: "9999"},
		{Name: "Z", TagGroupName: "Projects"},
		{Name: "Z", TagGroupName: "People"},
		nil,
	}

	plans, failedCount := planMCPTransactionTagsCreation(items, getTransactionTagTestTags(), getTransactionTagTestTagGroups(), true)

	assert.Equal(t, 6, failedCount)
	assert.Equal(t, MCPToolErrorCodeRequired, getTransactionTagTestToolError(t, plans[0].err).Code)

	toolErr := getTransactionTagTestToolError(t, plans[1].err)
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, int64(mcpTagNameMaxLength), *toolErr.Limit)
	assert.Equal(t, int64(mcpTagNameMaxLength+1), *toolErr.Actual)

	toolErr = getTransactionTagTestToolError(t, plans[2].err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))

	assert.Equal(t, MCPToolErrorCodeNotFound, getTransactionTagTestToolError(t, plans[3].err).Code)
	assert.Nil(t, plans[4].err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getTransactionTagTestToolError(t, plans[5].err).Code)
	assert.Equal(t, MCPToolErrorCodeRequired, getTransactionTagTestToolError(t, plans[6].err).Code)

	handler := &mcpCreateTransactionTagsToolHandler{}
	response := handler.createResponse(plans, nil, getMCPTransactionTagGroupsMap(getTransactionTagTestTagGroups()), false, false)
	assert.False(t, response.Success)
	assert.False(t, response.Committed)
	assert.Equal(t, 6, response.FailedCount)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, response.Results[4].Status)
}

func TestTransactionTagResolveTagGroupId(t *testing.T) {
	tagGroups := getTransactionTagTestTagGroups()

	tagGroupId, set, err := resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", "", "tag_group_name", "")
	assert.Nil(t, err)
	assert.False(t, set)
	assert.Equal(t, int64(0), tagGroupId)

	tagGroupId, set, err = resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", "0", "tag_group_name", "")
	assert.Nil(t, err)
	assert.True(t, set)
	assert.Equal(t, int64(0), tagGroupId)

	tagGroupId, set, err = resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", "", "tag_group_name", "projects")
	assert.Nil(t, err)
	assert.True(t, set)
	assert.Equal(t, int64(1001), tagGroupId)

	_, _, err = resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", "0", "tag_group_name", "Projects")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getTransactionTagTestToolError(t, err).Code)

	_, _, err = resolveMCPTransactionTagGroupId(tagGroups, "tag_group_id", "abc", "tag_group_name", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getTransactionTagTestToolError(t, err).Code)
}

func TestTransactionTagFindTagGroupForReuse(t *testing.T) {
	tagGroups := getTransactionTagTestTagGroups()

	tagGroup, err := findMCPTransactionTagGroupForReuse(tagGroups, "People")
	assert.Nil(t, err)
	assert.Equal(t, int64(1002), tagGroup.TagGroupId)

	tagGroup, err = findMCPTransactionTagGroupForReuse(tagGroups, "Nothing")
	assert.Nil(t, err)
	assert.Nil(t, tagGroup)

	_, err = findMCPTransactionTagGroupForReuse(tagGroups, "Duplicated")
	toolErr := getTransactionTagTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))
}

func TestTransactionTagReorderedDisplayOrders(t *testing.T) {
	currentDisplayOrders := map[int64]int32{
		1: 1,
		2: 2,
		3: 5,
		4: 7,
	}

	ids, displayOrders, err := getMCPTransactionTagReorderedDisplayOrders("tag_ids", []string{"4", "1", "3"}, currentDisplayOrders, "tag")
	assert.Nil(t, err)
	assert.Equal(t, []int64{4, 1, 3}, ids)
	assert.Equal(t, []int32{1, 5, 7}, displayOrders)

	_, _, err = getMCPTransactionTagReorderedDisplayOrders("tag_ids", []string{"1", "1"}, currentDisplayOrders, "tag")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getTransactionTagTestToolError(t, err).Code)

	_, _, err = getMCPTransactionTagReorderedDisplayOrders("tag_ids", []string{"9"}, currentDisplayOrders, "tag")
	assert.Equal(t, MCPToolErrorCodeNotFound, getTransactionTagTestToolError(t, err).Code)

	_, _, err = getMCPTransactionTagReorderedDisplayOrders("tag_ids", []string{}, currentDisplayOrders, "tag")
	assert.Equal(t, MCPToolErrorCodeRequired, getTransactionTagTestToolError(t, err).Code)
}

func TestTransactionTagReorderedDisplayOrders_DuplicatedCurrentOrders(t *testing.T) {
	currentDisplayOrders := map[int64]int32{
		1: 3,
		2: 3,
	}

	ids, displayOrders, err := getMCPTransactionTagReorderedDisplayOrders("tag_ids", []string{"2", "1"}, currentDisplayOrders, "tag")
	assert.Nil(t, err)
	assert.Equal(t, []int64{2, 1}, ids)
	assert.Equal(t, []int32{3, 4}, displayOrders)
}

func TestTransactionTagListCreateResponse(t *testing.T) {
	handler := &mcpListTransactionTagsToolHandler{}
	response := handler.createResponse(getTransactionTagTestTagGroups(), getTransactionTagTestTags(), false)

	assert.Equal(t, 4, len(response.TagGroups))
	assert.Equal(t, "1002", response.TagGroups[0].Id)
	assert.Equal(t, "1001", response.TagGroups[1].Id)

	assert.Equal(t, 3, len(response.Tags))
	assert.Equal(t, "2001", response.Tags[0].Id)
	assert.Equal(t, "0", response.Tags[0].TagGroupId)
	assert.Equal(t, "2003", response.Tags[1].Id)
	assert.Equal(t, "People", response.Tags[1].TagGroupName)
	assert.Equal(t, "2002", response.Tags[2].Id)

	response = handler.createResponse(getTransactionTagTestTagGroups(), getTransactionTagTestTags(), true)
	assert.Equal(t, 4, len(response.Tags))
	assert.Equal(t, "2004", response.Tags[1].Id)
	assert.True(t, response.Tags[1].Hidden)
}
