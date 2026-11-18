package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func newTransactionCategoryTestData() ([]*models.TransactionCategory, map[int64]*models.TransactionCategory) {
	categories := []*models.TransactionCategory{
		{CategoryId: 1, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 0, Name: "Business", DisplayOrder: 1},
		{CategoryId: 2, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 0, Name: "Food", DisplayOrder: 2},
		{CategoryId: 3, Uid: 1, Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 0, Name: "Business", DisplayOrder: 1},
		{CategoryId: 11, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1, Name: "Software", DisplayOrder: 1},
		{CategoryId: 12, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 1, Name: "Travel", DisplayOrder: 2},
		{CategoryId: 21, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 2, Name: "Travel", DisplayOrder: 1, Hidden: true},
		{CategoryId: 31, Uid: 1, Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 3, Name: "Consulting", DisplayOrder: 1},
	}

	categoryMap := make(map[int64]*models.TransactionCategory)

	for i := 0; i < len(categories); i++ {
		categoryMap[categories[i].CategoryId] = categories[i]
	}

	return categories, categoryMap
}

func TestTransactionCategoryResolveTarget_ById(t *testing.T) {
	categories, categoryMap := newTransactionCategoryTestData()

	category, err := mcpTransactionCategoryResolveTarget(categories, categoryMap, "12", "", "", "", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(12), category.CategoryId)

	_, err = mcpTransactionCategoryResolveTarget(categories, categoryMap, "999", "", "", "", "")
	assert.Equal(t, MCPToolErrorCodeNotFound, err.(*MCPToolError).Code)

	_, err = mcpTransactionCategoryResolveTarget(categories, categoryMap, "abc", "", "", "", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, err.(*MCPToolError).Code)
}

func TestTransactionCategoryResolveTarget_AmbiguousName(t *testing.T) {
	categories, categoryMap := newTransactionCategoryTestData()

	_, err := mcpTransactionCategoryResolveTarget(categories, categoryMap, "", "Travel", "", "", "")
	toolErr := err.(*MCPToolError)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, toolErr.Code)
	assert.Equal(t, 2, len(toolErr.Candidates))
	assert.Equal(t, "12", toolErr.Candidates[0].Id)
	assert.True(t, strings.Contains(toolErr.Candidates[0].Description, "\"Business\""))
	assert.True(t, strings.Contains(toolErr.Candidates[1].Description, "hidden"))

	category, err := mcpTransactionCategoryResolveTarget(categories, categoryMap, "", "Travel", "", "", "Food")
	assert.Nil(t, err)
	assert.Equal(t, int64(21), category.CategoryId)

	_, err = mcpTransactionCategoryResolveTarget(categories, categoryMap, "", "Business", "", "", "")
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, err.(*MCPToolError).Code)

	category, err = mcpTransactionCategoryResolveTarget(categories, categoryMap, "", "business", "income", "", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(3), category.CategoryId)
}

func TestTransactionCategoryResolveParent_MustBePrimary(t *testing.T) {
	categories, categoryMap := newTransactionCategoryTestData()

	_, err := mcpTransactionCategoryResolveParent(categories, categoryMap, 0, "parent_category_id", "11", "parent_category_name", "")
	assert.Equal(t, MCPToolErrorCodeNotAllowed, err.(*MCPToolError).Code)

	_, err = mcpTransactionCategoryResolveParent(categories, categoryMap, models.CATEGORY_TYPE_INCOME, "parent_category_id", "1", "parent_category_name", "")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, err.(*MCPToolError).Code)

	parent, err := mcpTransactionCategoryResolveParent(categories, categoryMap, models.CATEGORY_TYPE_EXPENSE, "parent_category_id", "", "parent_category_name", "Business")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), parent.CategoryId)

	parent, err = mcpTransactionCategoryResolveParent(categories, categoryMap, 0, "parent_category_id", "", "parent_category_name", "")
	assert.Nil(t, err)
	assert.Nil(t, parent)
}

func TestTransactionCategoryPlanBatch_SkipExisting(t *testing.T) {
	categories, _ := newTransactionCategoryTestData()
	items := []*MCPBatchCreateTransactionCategoriesItem{
		{
			Name: "Business",
			Type: "expense",
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: "software"},
				{Name: "Hosting"},
			},
		},
		{
			Name: "Office",
			Type: "expense",
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: "Rent"},
			},
		},
	}

	groups, results, valid := mcpTransactionCategoryPlanBatch(1, categories, items, true)
	assert.True(t, valid)
	assert.Equal(t, 2, len(groups))
	assert.Equal(t, 5, len(results))

	assert.Equal(t, MCPWriteStatusExisting, results[0].Status)
	assert.Equal(t, "1", results[0].Id)
	assert.Equal(t, MCPWriteStatusExisting, results[1].Status)
	assert.Equal(t, "11", results[1].Id)
	assert.Equal(t, 0, *results[1].SubIndex)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[2].Status)
	assert.Equal(t, "1", results[2].ParentId)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[3].Status)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[4].Status)

	assert.NotNil(t, groups[0].primaryExisting)
	assert.Nil(t, groups[0].primaryNew)
	assert.Nil(t, groups[0].subNews[0])
	assert.Equal(t, int64(1), groups[0].subNews[1].ParentCategoryId)
	assert.Equal(t, models.CATEGORY_TYPE_EXPENSE, groups[0].subNews[1].Type)
	assert.NotNil(t, groups[1].primaryNew)
	assert.Equal(t, "Office", groups[1].primaryNew.Name)
	assert.Equal(t, mcpTransactionCategoryDefaultIcon, groups[1].primaryNew.Icon)
	assert.Equal(t, mcpTransactionCategoryDefaultColor, groups[1].primaryNew.Color)
}

func TestTransactionCategoryPlanBatch_NotSkipExisting(t *testing.T) {
	categories, _ := newTransactionCategoryTestData()
	items := []*MCPBatchCreateTransactionCategoriesItem{
		{
			Name: "Business",
			Type: "expense",
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: "Software"},
			},
		},
	}

	groups, results, valid := mcpTransactionCategoryPlanBatch(1, categories, items, false)
	assert.True(t, valid)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[0].Status)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[1].Status)
	assert.NotNil(t, groups[0].primaryNew)
	assert.Equal(t, int64(0), groups[0].subNews[0].ParentCategoryId)
}

func TestTransactionCategoryPlanBatch_DuplicatesInRequest(t *testing.T) {
	items := []*MCPBatchCreateTransactionCategoriesItem{
		{
			Name: "Office",
			Type: "expense",
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: "Rent"},
				{Name: " rent "},
			},
		},
		{Name: "office", Type: "expense"},
		{Name: "Office", Type: "income"},
	}

	_, results, valid := mcpTransactionCategoryPlanBatch(1, nil, items, true)
	assert.False(t, valid)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[0].Status)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[1].Status)
	assert.Equal(t, MCPWriteStatusFailed, results[2].Status)
	assert.Equal(t, "categories[0].sub_categories[1].name", results[2].Error.Field)
	assert.Equal(t, MCPWriteStatusFailed, results[3].Status)
	assert.Equal(t, "categories[1].name", results[3].Error.Field)
	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[4].Status)
}

func TestTransactionCategoryPlanBatch_AmbiguousExisting(t *testing.T) {
	categories, _ := newTransactionCategoryTestData()
	categories = append(categories, &models.TransactionCategory{CategoryId: 4, Uid: 1, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 0, Name: "Business", DisplayOrder: 3})

	items := []*MCPBatchCreateTransactionCategoriesItem{
		{
			Name: "Business",
			Type: "expense",
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: "Software"},
			},
		},
	}

	_, results, valid := mcpTransactionCategoryPlanBatch(1, categories, items, true)
	assert.False(t, valid)
	assert.Equal(t, MCPWriteStatusFailed, results[0].Status)
	assert.Equal(t, MCPToolErrorCodeAmbiguousName, results[0].Error.Code)
	assert.Equal(t, 2, len(results[0].Error.Candidates))
	assert.Equal(t, MCPWriteStatusSkipped, results[1].Status)
}

func TestTransactionCategoryPlanBatch_FieldValidation(t *testing.T) {
	items := []*MCPBatchCreateTransactionCategoriesItem{
		{Name: strings.Repeat("a", mcpCategoryNameMaxLength+1), Type: "expense"},
		{Name: "Valid", Type: "unknown"},
		{Name: "Valid", Type: "expense", Color: "xyz"},
		{
			Name:    "Valid2",
			Type:    "income",
			Comment: strings.Repeat("é", mcpCategoryCommentMaxLength),
			SubCategories: []*MCPBatchCreateTransactionCategoriesSubItem{
				{Name: " "},
				{Name: "Sub", Comment: strings.Repeat("b", mcpCategoryCommentMaxLength+1)},
				{Name: "Sub2", Icon: "-1"},
			},
		},
		nil,
	}

	_, results, valid := mcpTransactionCategoryPlanBatch(1, nil, items, true)
	assert.False(t, valid)
	assert.Equal(t, 8, len(results))

	assert.Equal(t, MCPToolErrorCodeTooLong, results[0].Error.Code)
	assert.Equal(t, "categories[0].name", results[0].Error.Field)
	assert.Equal(t, int64(mcpCategoryNameMaxLength), *results[0].Error.Limit)
	assert.Equal(t, int64(mcpCategoryNameMaxLength+1), *results[0].Error.Actual)

	assert.Equal(t, "categories[1].type", results[1].Error.Field)
	assert.Equal(t, "categories[2].color", results[2].Error.Field)

	assert.Equal(t, MCPWriteStatusWouldBeCreated, results[3].Status)
	assert.Equal(t, MCPToolErrorCodeRequired, results[4].Error.Code)
	assert.Equal(t, "categories[3].sub_categories[0].name", results[4].Error.Field)
	assert.Equal(t, MCPToolErrorCodeTooLong, results[5].Error.Code)
	assert.Equal(t, "categories[3].sub_categories[1].comment", results[5].Error.Field)
	assert.Equal(t, "categories[3].sub_categories[2].icon", results[6].Error.Field)

	assert.Equal(t, MCPWriteStatusFailed, results[len(results)-1].Status)
}

func TestTransactionCategoryGetReorderedSiblings(t *testing.T) {
	categories, categoryMap := newTransactionCategoryTestData()

	orderedCategories, err := mcpTransactionCategoryGetReorderedSiblings(categories, categoryMap, []string{"12"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(orderedCategories))
	assert.Equal(t, int64(12), orderedCategories[0].CategoryId)
	assert.Equal(t, int64(11), orderedCategories[1].CategoryId)

	_, err = mcpTransactionCategoryGetReorderedSiblings(categories, categoryMap, []string{"12", "21"})
	assert.Equal(t, MCPToolErrorCodeInvalidValue, err.(*MCPToolError).Code)

	_, err = mcpTransactionCategoryGetReorderedSiblings(categories, categoryMap, []string{"12", "12"})
	assert.Equal(t, MCPToolErrorCodeInvalidValue, err.(*MCPToolError).Code)

	_, err = mcpTransactionCategoryGetReorderedSiblings(categories, categoryMap, []string{"999"})
	assert.Equal(t, MCPToolErrorCodeNotFound, err.(*MCPToolError).Code)
}

func TestTransactionCategoryParseColorAndIcon(t *testing.T) {
	color, err := mcpTransactionCategoryParseColor("color", "")
	assert.Nil(t, err)
	assert.Equal(t, mcpTransactionCategoryDefaultColor, color)

	color, err = mcpTransactionCategoryParseColor("color", "ff00AA")
	assert.Nil(t, err)
	assert.Equal(t, "ff00AA", color)

	_, err = mcpTransactionCategoryParseColor("color", "fff")
	assert.NotNil(t, err)

	icon, err := mcpTransactionCategoryParseIcon("icon", "")
	assert.Nil(t, err)
	assert.Equal(t, mcpTransactionCategoryDefaultIcon, icon)

	icon, err = mcpTransactionCategoryParseIcon("icon", "25")
	assert.Nil(t, err)
	assert.Equal(t, int64(25), icon)

	_, err = mcpTransactionCategoryParseIcon("icon", "0")
	assert.NotNil(t, err)
}
