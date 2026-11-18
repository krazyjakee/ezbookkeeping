package mcp

import (
	"fmt"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPReorderTransactionCategoriesRequest represents all parameters of the reorder transaction categories request
type MCPReorderTransactionCategoriesRequest struct {
	CategoryIds []string `json:"category_ids" jsonschema_description:"Category IDs in the desired order. All of them must be siblings (same type and same parent category). Sibling categories which are not listed keep their relative order and are placed after the listed ones"`
}

// MCPReorderTransactionCategoriesResponse represents the response structure for reordering transaction categories
type MCPReorderTransactionCategoriesResponse struct {
	Success    bool                          `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Committed  bool                          `json:"committed" jsonschema_description:"Indicates whether the new order has been saved"`
	Status     string                        `json:"status" jsonschema:"enum=updated,enum=unchanged" jsonschema_description:"updated: saved; unchanged: the categories are already in the requested order"`
	Categories []*MCPTransactionCategoryInfo `json:"categories" jsonschema_description:"All sibling categories in their new order"`
}

type mcpReorderTransactionCategoriesToolHandler struct{}

var MCPReorderTransactionCategoriesToolHandler = &mcpReorderTransactionCategoriesToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpReorderTransactionCategoriesToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_REORDER_TRANSACTION_CATEGORIES
}

// Name returns the name of the MCP tool
func (h *mcpReorderTransactionCategoriesToolHandler) Name() string {
	return "reorder_transaction_categories"
}

// Description returns the description of the MCP tool
func (h *mcpReorderTransactionCategoriesToolHandler) Description() string {
	return "Change the display order of sibling transaction categories (primary categories of one type, or sub-categories of one primary category)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpReorderTransactionCategoriesToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpReorderTransactionCategoriesToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpReorderTransactionCategoriesToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpReorderTransactionCategoriesToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionCategoriesRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpReorderTransactionCategoriesToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPReorderTransactionCategoriesResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpReorderTransactionCategoriesToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPReorderTransactionCategoriesRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if len(request.CategoryIds) < 1 {
		return nil, nil, NewMCPRequiredFieldError("category_ids")
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[reorder_transaction_categories_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := categoryService.GetCategoryMapByList(allCategories)
	orderedCategories, err := mcpTransactionCategoryGetReorderedSiblings(allCategories, categoryMap, request.CategoryIds)

	if err != nil {
		return nil, nil, err
	}

	toUpdateCategories := make([]*models.TransactionCategory, 0, len(orderedCategories))

	for i := 0; i < len(orderedCategories); i++ {
		newDisplayOrder := int32(i + 1)

		if orderedCategories[i].DisplayOrder != newDisplayOrder {
			toUpdateCategories = append(toUpdateCategories, &models.TransactionCategory{
				Uid:          uid,
				CategoryId:   orderedCategories[i].CategoryId,
				DisplayOrder: newDisplayOrder,
			})
		}
	}

	response := MCPReorderTransactionCategoriesResponse{
		Success:    true,
		Committed:  false,
		Status:     MCPWriteStatusUnchanged,
		Categories: make([]*MCPTransactionCategoryInfo, 0, len(orderedCategories)),
	}

	if len(toUpdateCategories) > 0 {
		err = categoryService.ModifyCategoryDisplayOrders(c, uid, toUpdateCategories)

		if err != nil {
			log.Errorf(c, "[reorder_transaction_categories_tool_handler.Handle] failed to move categories for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		log.Infof(c, "[reorder_transaction_categories_tool_handler.Handle] user \"uid:%d\" has moved categories", uid)

		response.Committed = true
		response.Status = MCPWriteStatusUpdated
	}

	for i := 0; i < len(orderedCategories); i++ {
		orderedCategories[i].DisplayOrder = int32(i + 1)
		response.Categories = append(response.Categories, mcpTransactionCategoryToInfo(orderedCategories[i], categoryMap))
	}

	return newMCPJSONTextResult(response)
}

// mcpTransactionCategoryGetReorderedSiblings returns all sibling categories in the new order (listed categories first, then the other siblings in their current order)
func mcpTransactionCategoryGetReorderedSiblings(allCategories []*models.TransactionCategory, categoryMap map[int64]*models.TransactionCategory, textualIds []string) ([]*models.TransactionCategory, error) {
	listedCategories := make([]*models.TransactionCategory, 0, len(textualIds))
	listedCategoryIds := make(map[int64]bool, len(textualIds))

	for i := 0; i < len(textualIds); i++ {
		field := fmt.Sprintf("category_ids[%d]", i)
		categoryId, err := parseMCPRequiredId(field, textualIds[i])

		if err != nil {
			return nil, err
		}

		category, exists := categoryMap[categoryId]

		if !exists || category == nil {
			return nil, NewMCPNotFoundError(field, "transaction category", textualIds[i])
		}

		if listedCategoryIds[categoryId] {
			return nil, NewMCPInvalidFieldError(field, textualIds[i], "category ID is listed more than once")
		}

		if len(listedCategories) > 0 && (category.Type != listedCategories[0].Type || category.ParentCategoryId != listedCategories[0].ParentCategoryId) {
			return nil, NewMCPInvalidFieldError(field, textualIds[i], "all categories must be siblings (same type and same parent category) of category_ids[0]")
		}

		listedCategoryIds[categoryId] = true
		listedCategories = append(listedCategories, category)
	}

	orderedCategories := listedCategories
	siblings := mcpTransactionCategoryFilter(allCategories, listedCategories[0].Type, listedCategories[0].ParentCategoryId)

	for i := 0; i < len(siblings); i++ {
		if !listedCategoryIds[siblings[i].CategoryId] {
			orderedCategories = append(orderedCategories, siblings[i])
		}
	}

	return orderedCategories, nil
}
