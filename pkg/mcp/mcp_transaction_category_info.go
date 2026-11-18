package mcp

import (
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpTransactionCategoryDefaultIcon is the default system icon id of transaction category (same as DEFAULT_CATEGORY_ICON_ID in frontend)
const mcpTransactionCategoryDefaultIcon int64 = 1

// mcpTransactionCategoryDefaultColor is the default color of transaction category (same as DEFAULT_CATEGORY_COLOR in frontend)
const mcpTransactionCategoryDefaultColor = "000000"

// MCPTransactionCategoryInfo defines the structure of transaction category information
type MCPTransactionCategoryInfo struct {
	Id           string `json:"id,omitempty" jsonschema_description:"Stable transaction category ID (empty if the category has not been saved, e.g. in dry run)"`
	Name         string `json:"name" jsonschema_description:"Transaction category name"`
	Type         string `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type (income, expense, transfer)"`
	ParentId     string `json:"parent_id" jsonschema_description:"Parent (primary) category ID, \"0\" for primary categories"`
	ParentName   string `json:"parent_name,omitempty" jsonschema_description:"Parent (primary) category name (only for sub-categories)"`
	IsPrimary    bool   `json:"is_primary" jsonschema_description:"Whether this is a primary category (sub-categories are used by transactions)"`
	Icon         string `json:"icon" jsonschema_description:"Icon ID"`
	Color        string `json:"color" jsonschema_description:"Color in hex RGB format (e.g. 000000)"`
	Comment      string `json:"comment,omitempty" jsonschema_description:"Category description"`
	DisplayOrder int32  `json:"display_order" jsonschema_description:"Display order among sibling categories"`
	Hidden       bool   `json:"hidden" jsonschema_description:"Whether the category is hidden"`
}

// MCPTransactionCategoryTreeInfo defines the structure of transaction category information with its sub-categories
type MCPTransactionCategoryTreeInfo struct {
	Id            string                        `json:"id" jsonschema_description:"Stable transaction category ID"`
	Name          string                        `json:"name" jsonschema_description:"Transaction category name"`
	Type          string                        `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type (income, expense, transfer)"`
	ParentId      string                        `json:"parent_id" jsonschema_description:"Parent (primary) category ID, \"0\" for primary categories"`
	ParentName    string                        `json:"parent_name,omitempty" jsonschema_description:"Parent (primary) category name (only for sub-categories)"`
	IsPrimary     bool                          `json:"is_primary" jsonschema_description:"Whether this is a primary category (sub-categories are used by transactions)"`
	Icon          string                        `json:"icon" jsonschema_description:"Icon ID"`
	Color         string                        `json:"color" jsonschema_description:"Color in hex RGB format (e.g. 000000)"`
	Comment       string                        `json:"comment,omitempty" jsonschema_description:"Category description"`
	DisplayOrder  int32                         `json:"display_order" jsonschema_description:"Display order among sibling categories"`
	Hidden        bool                          `json:"hidden" jsonschema_description:"Whether the category is hidden"`
	SubCategories []*MCPTransactionCategoryInfo `json:"sub_categories,omitempty" jsonschema_description:"Sub-categories of this primary category"`
}

// mcpTransactionCategoryToInfo returns the MCP view-object of the transaction category
func mcpTransactionCategoryToInfo(category *models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory) *MCPTransactionCategoryInfo {
	info := &MCPTransactionCategoryInfo{
		Name:         category.Name,
		Type:         getMCPCategoryTypeName(category.Type),
		ParentId:     utils.Int64ToString(category.ParentCategoryId),
		IsPrimary:    category.ParentCategoryId == models.LevelOneTransactionCategoryParentId,
		Icon:         utils.Int64ToString(category.Icon),
		Color:        category.Color,
		Comment:      category.Comment,
		DisplayOrder: category.DisplayOrder,
		Hidden:       category.Hidden,
	}

	if category.CategoryId > 0 {
		info.Id = utils.Int64ToString(category.CategoryId)
	}

	if category.ParentCategoryId != models.LevelOneTransactionCategoryParentId && categoryMap != nil {
		if parentCategory, exists := categoryMap[category.ParentCategoryId]; exists && parentCategory != nil {
			info.ParentName = parentCategory.Name
		}
	}

	return info
}

// mcpTransactionCategoryToTreeInfo returns the MCP view-object of the transaction category with the given sub-categories
func mcpTransactionCategoryToTreeInfo(category *models.TransactionCategory, subCategories []*models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory) *MCPTransactionCategoryTreeInfo {
	info := mcpTransactionCategoryToInfo(category, categoryMap)
	treeInfo := &MCPTransactionCategoryTreeInfo{
		Id:           info.Id,
		Name:         info.Name,
		Type:         info.Type,
		ParentId:     info.ParentId,
		ParentName:   info.ParentName,
		IsPrimary:    info.IsPrimary,
		Icon:         info.Icon,
		Color:        info.Color,
		Comment:      info.Comment,
		DisplayOrder: info.DisplayOrder,
		Hidden:       info.Hidden,
	}

	if len(subCategories) > 0 {
		treeInfo.SubCategories = make([]*MCPTransactionCategoryInfo, 0, len(subCategories))

		for i := 0; i < len(subCategories); i++ {
			treeInfo.SubCategories = append(treeInfo.SubCategories, mcpTransactionCategoryToInfo(subCategories[i], categoryMap))
		}
	}

	return treeInfo
}

// mcpTransactionCategoryGetSubCategories returns all sub-categories (ordered by display order as stored) of the specified primary category
func mcpTransactionCategoryGetSubCategories(allCategories []*models.TransactionCategory, parentCategoryId int64, includeHidden bool) []*models.TransactionCategory {
	subCategories := make([]*models.TransactionCategory, 0)

	for i := 0; i < len(allCategories); i++ {
		category := allCategories[i]

		if category.ParentCategoryId != parentCategoryId || parentCategoryId == models.LevelOneTransactionCategoryParentId {
			continue
		}

		if !includeHidden && category.Hidden {
			continue
		}

		subCategories = append(subCategories, category)
	}

	return subCategories
}

// mcpTransactionCategoryDescribe returns the description of the transaction category used to tell candidates apart
func mcpTransactionCategoryDescribe(category *models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory) string {
	var builder strings.Builder
	builder.WriteString(getMCPCategoryTypeName(category.Type))

	if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
		builder.WriteString(" primary category")
	} else {
		builder.WriteString(" sub-category")

		if parentCategory, exists := categoryMap[category.ParentCategoryId]; exists && parentCategory != nil {
			builder.WriteString(" of \"")
			builder.WriteString(parentCategory.Name)
			builder.WriteString("\" (id: ")
			builder.WriteString(utils.Int64ToString(parentCategory.CategoryId))
			builder.WriteString(")")
		}
	}

	if category.Hidden {
		builder.WriteString(", hidden")
	}

	return builder.String()
}

// mcpTransactionCategoryNewResolver returns a new resolver of transaction categories
func mcpTransactionCategoryNewResolver(categoryMap map[int64]*models.TransactionCategory) *mcpEntityResolver[*models.TransactionCategory] {
	return mcpTransactionCategoryNewNamedResolver("transaction category", categoryMap)
}

// mcpTransactionCategoryNewNamedResolver returns a new resolver of transaction categories which uses the given entity name in error messages
func mcpTransactionCategoryNewNamedResolver(entityName string, categoryMap map[int64]*models.TransactionCategory) *mcpEntityResolver[*models.TransactionCategory] {
	return newMCPEntityResolver[*models.TransactionCategory](entityName,
		func(category *models.TransactionCategory) int64 {
			return category.CategoryId
		},
		func(category *models.TransactionCategory) string {
			return category.Name
		},
		func(category *models.TransactionCategory) string {
			return mcpTransactionCategoryDescribe(category, categoryMap)
		})
}

// mcpTransactionCategoryFilter returns the categories which match the category type (if not zero) and parent category id (if not negative)
func mcpTransactionCategoryFilter(allCategories []*models.TransactionCategory, categoryType models.TransactionCategoryType, parentCategoryId int64) []*models.TransactionCategory {
	categories := make([]*models.TransactionCategory, 0, len(allCategories))

	for i := 0; i < len(allCategories); i++ {
		category := allCategories[i]

		if categoryType > 0 && category.Type != categoryType {
			continue
		}

		if parentCategoryId >= 0 && category.ParentCategoryId != parentCategoryId {
			continue
		}

		categories = append(categories, category)
	}

	return categories
}

// mcpTransactionCategoryResolveParent resolves the primary category by id or name (optionally filtered by category type, 0 means any type)
// returns nil if neither id nor name is set
func mcpTransactionCategoryResolveParent(allCategories []*models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory, categoryType models.TransactionCategoryType, idField string, id string, nameField string, name string) (*models.TransactionCategory, error) {
	if strings.TrimSpace(id) == "" && name == "" {
		return nil, nil
	}

	candidates := allCategories

	if strings.TrimSpace(id) == "" {
		candidates = mcpTransactionCategoryFilter(allCategories, categoryType, models.LevelOneTransactionCategoryParentId)
	}

	parentCategory, err := mcpTransactionCategoryNewResolver(categoryMap).Resolve(candidates, idField, id, nameField, name)

	if err != nil {
		return nil, err
	}

	if parentCategory.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
		return nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, idField, "the parent category must be a primary category, only two levels of categories are supported")
	}

	if categoryType > 0 && parentCategory.Type != categoryType {
		return nil, NewMCPInvalidFieldError(idField, utils.Int64ToString(parentCategory.CategoryId), "the type of the parent category is "+getMCPCategoryTypeName(parentCategory.Type)+", which does not match the specified type "+getMCPCategoryTypeName(categoryType))
	}

	return parentCategory, nil
}

// mcpTransactionCategoryResolveTarget resolves the transaction category by id, or by name with optional type and parent category to disambiguate
func mcpTransactionCategoryResolveTarget(allCategories []*models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory, id string, name string, textualType string, parentId string, parentName string) (*models.TransactionCategory, error) {
	candidates := allCategories

	if strings.TrimSpace(id) == "" {
		categoryType := models.TransactionCategoryType(0)

		if textualType != "" {
			parsedType, err := parseMCPCategoryType("type", textualType)

			if err != nil {
				return nil, err
			}

			categoryType = parsedType
		}

		parentCategoryId := int64(-1)
		parentCategory, err := mcpTransactionCategoryResolveParent(allCategories, categoryMap, categoryType, "parent_category_id", parentId, "parent_category_name", parentName)

		if err != nil {
			return nil, err
		}

		if parentCategory != nil {
			parentCategoryId = parentCategory.CategoryId
		}

		candidates = mcpTransactionCategoryFilter(allCategories, categoryType, parentCategoryId)
	}

	return mcpTransactionCategoryNewResolver(categoryMap).Resolve(candidates, "category_id", id, "category_name", name)
}

// mcpTransactionCategoryParseIcon parses the textual system icon id, returns the default icon if it is empty
func mcpTransactionCategoryParseIcon(field string, value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return mcpTransactionCategoryDefaultIcon, nil
	}

	return parseMCPOptionalId(field, value)
}

// mcpTransactionCategoryParseColor checks the color (6 hex digits as the HTTP API requires), returns the default color if it is empty
func mcpTransactionCategoryParseColor(field string, value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return mcpTransactionCategoryDefaultColor, nil
	}

	if len(value) != 6 || !utils.IsValidHexRGBColor(value) {
		return "", NewMCPInvalidFieldError(field, value, "\""+field+"\" must be a color in 6-digit hex RGB format (e.g. 000000)")
	}

	return value, nil
}

// mcpTransactionCategoryValidateFields validates the common fields of transaction category
func mcpTransactionCategoryValidateFields(fieldPrefix string, name string, comment string) error {
	if err := checkMCPStringLength(fieldPrefix+"name", name, true, mcpCategoryNameMaxLength); err != nil {
		return err
	}

	if err := checkMCPStringLength(fieldPrefix+"comment", comment, false, mcpCategoryCommentMaxLength); err != nil {
		return err
	}

	return nil
}

// mcpTransactionCategoryNewModel returns a new transaction category model
func mcpTransactionCategoryNewModel(uid int64, name string, categoryType models.TransactionCategoryType, parentCategoryId int64, icon int64, color string, comment string, displayOrder int32) *models.TransactionCategory {
	return &models.TransactionCategory{
		Uid:              uid,
		Name:             name,
		Type:             categoryType,
		ParentCategoryId: parentCategoryId,
		DisplayOrder:     displayOrder,
		Icon:             icon,
		IconType:         core.ICON_TYPE_SYSTEM,
		Color:            color,
		Comment:          comment,
	}
}
