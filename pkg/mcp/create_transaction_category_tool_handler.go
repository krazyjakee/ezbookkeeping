package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPCreateTransactionCategoryRequest represents all parameters of the create transaction category request
type MCPCreateTransactionCategoryRequest struct {
	Name               string `json:"name" jsonschema_description:"Transaction category name"`
	Type               string `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type (income, expense, transfer), required for primary categories, optional for sub-categories (inherited from the parent category and must match if set)"`
	ParentCategoryId   string `json:"parent_category_id,omitempty" jsonschema_description:"Parent (primary) category ID, set this (or parent_category_name) to create a sub-category (optional)"`
	ParentCategoryName string `json:"parent_category_name,omitempty" jsonschema_description:"Parent (primary) category name, used when parent_category_id is not set (optional)"`
	Icon               string `json:"icon,omitempty" jsonschema_description:"System icon ID (optional, default 1)"`
	Color              string `json:"color,omitempty" jsonschema_description:"Color in 6-digit hex RGB format (optional, default 000000)"`
	Comment            string `json:"comment,omitempty" jsonschema_description:"Category description (optional)"`
	SkipExisting       bool   `json:"skip_existing,omitempty" jsonschema_description:"If true and a category with the same name (exact, or case-insensitive if no exact match) already exists under the same parent with the same type, it is reused instead of creating a duplicate (optional, default false)"`
	DryRun             bool   `json:"dry_run,omitempty" jsonschema_description:"If true, only validate and preview the result without saving (optional)"`
}

// MCPCreateTransactionCategoryResponse represents the response structure for creating transaction category
type MCPCreateTransactionCategoryResponse struct {
	Success   bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun    bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing is saved)"`
	Committed bool                        `json:"committed" jsonschema_description:"Indicates whether a new category has been saved"`
	Status    string                      `json:"status" jsonschema:"enum=created,enum=existing,enum=would_be_created" jsonschema_description:"created: a new category has been saved; existing: an existing category is reused (skip_existing); would_be_created: dry run"`
	Category  *MCPTransactionCategoryInfo `json:"category" jsonschema_description:"The saved, reused or previewed category"`
}

type mcpCreateTransactionCategoryToolHandler struct{}

var MCPCreateTransactionCategoryToolHandler = &mcpCreateTransactionCategoryToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpCreateTransactionCategoryToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_CREATE_TRANSACTION_CATEGORY
}

// Name returns the name of the MCP tool
func (h *mcpCreateTransactionCategoryToolHandler) Name() string {
	return "create_transaction_category"
}

// Description returns the description of the MCP tool
func (h *mcpCreateTransactionCategoryToolHandler) Description() string {
	return "Create a primary transaction category, or a sub-category under a primary category (only two levels are supported, transactions use sub-categories). Use skip_existing to reuse an existing category with the same name instead of creating a duplicate, and dry_run to validate without saving."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpCreateTransactionCategoryToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpCreateTransactionCategoryToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpCreateTransactionCategoryToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpCreateTransactionCategoryToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionCategoryRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpCreateTransactionCategoryToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionCategoryResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpCreateTransactionCategoryToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPCreateTransactionCategoryRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpTransactionCategoryValidateFields("", request.Name, request.Comment); err != nil {
		return nil, nil, err
	}

	icon, err := mcpTransactionCategoryParseIcon("icon", request.Icon)

	if err != nil {
		return nil, nil, err
	}

	color, err := mcpTransactionCategoryParseColor("color", request.Color)

	if err != nil {
		return nil, nil, err
	}

	categoryType := models.TransactionCategoryType(0)

	if request.Type != "" {
		categoryType, err = parseMCPCategoryType("type", request.Type)

		if err != nil {
			return nil, nil, err
		}
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[create_transaction_category_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := categoryService.GetCategoryMapByList(allCategories)
	parentCategory, err := mcpTransactionCategoryResolveParent(allCategories, categoryMap, categoryType, "parent_category_id", request.ParentCategoryId, "parent_category_name", request.ParentCategoryName)

	if err != nil {
		return nil, nil, err
	}

	parentCategoryId := int64(models.LevelOneTransactionCategoryParentId)

	if parentCategory != nil {
		parentCategoryId = parentCategory.CategoryId
		categoryType = parentCategory.Type
	} else if categoryType == 0 {
		return nil, nil, NewMCPRequiredFieldError("type")
	}

	if request.SkipExisting {
		siblings := mcpTransactionCategoryFilter(allCategories, categoryType, parentCategoryId)
		resolver := mcpTransactionCategoryNewResolver(categoryMap)
		matches := resolver.FindAllByName(siblings, request.Name)

		if len(matches) > 1 {
			return nil, nil, NewMCPAmbiguousNameError("name", "transaction category", request.Name, resolver.GetCandidates(matches))
		} else if len(matches) == 1 {
			return newMCPJSONTextResult(MCPCreateTransactionCategoryResponse{
				Success:   true,
				DryRun:    request.DryRun,
				Committed: false,
				Status:    MCPWriteStatusExisting,
				Category:  mcpTransactionCategoryToInfo(matches[0], categoryMap),
			})
		}
	}

	var maxOrderId int32

	if parentCategoryId == models.LevelOneTransactionCategoryParentId {
		maxOrderId, err = categoryService.GetMaxDisplayOrder(c, uid, categoryType)
	} else {
		maxOrderId, err = categoryService.GetMaxSubCategoryDisplayOrder(c, uid, categoryType, parentCategoryId)
	}

	if err != nil {
		log.Errorf(c, "[create_transaction_category_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	category := mcpTransactionCategoryNewModel(uid, request.Name, categoryType, parentCategoryId, icon, color, request.Comment, maxOrderId+1)

	if request.DryRun {
		return newMCPJSONTextResult(MCPCreateTransactionCategoryResponse{
			Success:   true,
			DryRun:    true,
			Committed: false,
			Status:    MCPWriteStatusWouldBeCreated,
			Category:  mcpTransactionCategoryToInfo(category, categoryMap),
		})
	}

	err = categoryService.CreateCategory(c, category)

	if err != nil {
		log.Errorf(c, "[create_transaction_category_tool_handler.Handle] failed to create category for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[create_transaction_category_tool_handler.Handle] user \"uid:%d\" has created a new category \"id:%d\" successfully", uid, category.CategoryId)

	savedCategory, err := categoryService.GetCategoryByCategoryId(c, uid, category.CategoryId)

	if err != nil {
		log.Warnf(c, "[create_transaction_category_tool_handler.Handle] failed to get the created category \"id:%d\" for user \"uid:%d\", because %s", category.CategoryId, uid, err.Error())
		savedCategory = category
	}

	return newMCPJSONTextResult(MCPCreateTransactionCategoryResponse{
		Success:   true,
		DryRun:    false,
		Committed: true,
		Status:    MCPWriteStatusCreated,
		Category:  mcpTransactionCategoryToInfo(savedCategory, categoryMap),
	})
}
