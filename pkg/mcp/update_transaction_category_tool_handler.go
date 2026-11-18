package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPUpdateTransactionCategoryRequest represents all parameters of the update transaction category request
type MCPUpdateTransactionCategoryRequest struct {
	CategoryId            string  `json:"category_id,omitempty" jsonschema_description:"ID of the transaction category to update (preferred, either category_id or category_name is required)"`
	CategoryName          string  `json:"category_name,omitempty" jsonschema_description:"Name of the transaction category to update (used when category_id is not set)"`
	Type                  string  `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type to disambiguate category_name (optional, the type of a category cannot be changed)"`
	ParentCategoryName    string  `json:"parent_category_name,omitempty" jsonschema_description:"Current parent category name to disambiguate category_name of a sub-category (optional)"`
	Name                  *string `json:"name,omitempty" jsonschema_description:"New category name (optional)"`
	Icon                  *string `json:"icon,omitempty" jsonschema_description:"New system icon ID (optional)"`
	Color                 *string `json:"color,omitempty" jsonschema_description:"New color in 6-digit hex RGB format (optional)"`
	Comment               *string `json:"comment,omitempty" jsonschema_description:"New category description (optional, empty string clears it)"`
	Hidden                *bool   `json:"hidden,omitempty" jsonschema_description:"Whether the category is hidden (optional)"`
	NewParentCategoryId   string  `json:"new_parent_category_id,omitempty" jsonschema_description:"Move this sub-category to another primary category of the same type by ID (optional, primary categories cannot be moved)"`
	NewParentCategoryName string  `json:"new_parent_category_name,omitempty" jsonschema_description:"Move this sub-category to another primary category of the same type by name (optional, used when new_parent_category_id is not set)"`
	DryRun                bool    `json:"dry_run,omitempty" jsonschema_description:"If true, only validate and preview the result without saving (optional)"`
}

// MCPUpdateTransactionCategoryResponse represents the response structure for updating transaction category
type MCPUpdateTransactionCategoryResponse struct {
	Success   bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun    bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing is saved)"`
	Committed bool                        `json:"committed" jsonschema_description:"Indicates whether the change has been saved"`
	Status    string                      `json:"status" jsonschema:"enum=updated,enum=unchanged,enum=would_be_updated" jsonschema_description:"updated: saved; unchanged: nothing would change; would_be_updated: dry run"`
	Category  *MCPTransactionCategoryInfo `json:"category" jsonschema_description:"The saved (or previewed) category"`
}

type mcpUpdateTransactionCategoryToolHandler struct{}

var MCPUpdateTransactionCategoryToolHandler = &mcpUpdateTransactionCategoryToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateTransactionCategoryToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_TRANSACTION_CATEGORY
}

// Name returns the name of the MCP tool
func (h *mcpUpdateTransactionCategoryToolHandler) Name() string {
	return "update_transaction_category"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateTransactionCategoryToolHandler) Description() string {
	return "Update a transaction category (name, icon, color, description, hidden), or move a sub-category to another primary category of the same type. Only the specified fields are changed. The type of a category cannot be changed, and primary categories cannot become sub-categories (or vice versa)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateTransactionCategoryToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateTransactionCategoryToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateTransactionCategoryToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateTransactionCategoryToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionCategoryRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateTransactionCategoryToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionCategoryResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateTransactionCategoryToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUpdateTransactionCategoryRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[update_transaction_category_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := categoryService.GetCategoryMapByList(allCategories)
	category, err := mcpTransactionCategoryResolveTarget(allCategories, categoryMap, request.CategoryId, request.CategoryName, request.Type, "", request.ParentCategoryName)

	if err != nil {
		return nil, nil, err
	}

	newCategory := &models.TransactionCategory{
		CategoryId:       category.CategoryId,
		Uid:              uid,
		Type:             category.Type,
		ParentCategoryId: category.ParentCategoryId,
		Name:             category.Name,
		DisplayOrder:     category.DisplayOrder,
		Icon:             category.Icon,
		IconType:         category.IconType,
		Color:            category.Color,
		Comment:          category.Comment,
		Hidden:           category.Hidden,
	}

	if request.Name != nil {
		if err := checkMCPStringLength("name", *request.Name, true, mcpCategoryNameMaxLength); err != nil {
			return nil, nil, err
		}

		newCategory.Name = *request.Name
	}

	if request.Comment != nil {
		if err := checkMCPStringLength("comment", *request.Comment, false, mcpCategoryCommentMaxLength); err != nil {
			return nil, nil, err
		}

		newCategory.Comment = *request.Comment
	}

	if request.Icon != nil {
		icon, err := parseMCPRequiredId("icon", *request.Icon)

		if err != nil {
			return nil, nil, err
		}

		if icon != category.Icon || category.IconType != core.ICON_TYPE_SYSTEM {
			newCategory.Icon = icon
			newCategory.IconType = core.ICON_TYPE_SYSTEM
		}
	}

	if request.Color != nil {
		if *request.Color == "" {
			return nil, nil, NewMCPRequiredFieldError("color")
		}

		color, err := mcpTransactionCategoryParseColor("color", *request.Color)

		if err != nil {
			return nil, nil, err
		}

		newCategory.Color = color
	}

	if request.Hidden != nil {
		newCategory.Hidden = *request.Hidden
	}

	newParentCategory, err := mcpTransactionCategoryResolveParent(allCategories, categoryMap, category.Type, "new_parent_category_id", request.NewParentCategoryId, "new_parent_category_name", request.NewParentCategoryName)

	if err != nil {
		return nil, nil, err
	}

	if newParentCategory != nil && newParentCategory.CategoryId != category.ParentCategoryId {
		if category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "new_parent_category_id", "a primary category cannot be changed to a sub-category")
		}

		maxOrderId, err := categoryService.GetMaxSubCategoryDisplayOrder(c, uid, category.Type, newParentCategory.CategoryId)

		if err != nil {
			log.Errorf(c, "[update_transaction_category_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		newCategory.ParentCategoryId = newParentCategory.CategoryId
		newCategory.DisplayOrder = maxOrderId + 1
	}

	if newCategory.ParentCategoryId == category.ParentCategoryId &&
		newCategory.Name == category.Name &&
		newCategory.Icon == category.Icon &&
		newCategory.IconType == category.IconType &&
		newCategory.Color == category.Color &&
		newCategory.Comment == category.Comment &&
		newCategory.Hidden == category.Hidden {
		return newMCPJSONTextResult(MCPUpdateTransactionCategoryResponse{
			Success:   true,
			DryRun:    request.DryRun,
			Committed: false,
			Status:    MCPWriteStatusUnchanged,
			Category:  mcpTransactionCategoryToInfo(category, categoryMap),
		})
	}

	if request.DryRun {
		return newMCPJSONTextResult(MCPUpdateTransactionCategoryResponse{
			Success:   true,
			DryRun:    true,
			Committed: false,
			Status:    MCPWriteStatusWouldBeUpdated,
			Category:  mcpTransactionCategoryToInfo(newCategory, categoryMap),
		})
	}

	err = categoryService.ModifyCategory(c, newCategory)

	if err != nil {
		log.Errorf(c, "[update_transaction_category_tool_handler.Handle] failed to update category \"id:%d\" for user \"uid:%d\", because %s", category.CategoryId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[update_transaction_category_tool_handler.Handle] user \"uid:%d\" has updated category \"id:%d\" successfully", uid, category.CategoryId)

	savedCategory, err := categoryService.GetCategoryByCategoryId(c, uid, category.CategoryId)

	if err != nil {
		log.Warnf(c, "[update_transaction_category_tool_handler.Handle] failed to get the updated category \"id:%d\" for user \"uid:%d\", because %s", category.CategoryId, uid, err.Error())
		savedCategory = newCategory
	}

	return newMCPJSONTextResult(MCPUpdateTransactionCategoryResponse{
		Success:   true,
		DryRun:    false,
		Committed: true,
		Status:    MCPWriteStatusUpdated,
		Category:  mcpTransactionCategoryToInfo(savedCategory, categoryMap),
	})
}
