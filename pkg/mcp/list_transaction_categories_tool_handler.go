package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPListTransactionCategoriesRequest represents all parameters of the list transaction categories request
type MCPListTransactionCategoriesRequest struct {
	Type          string `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type to filter by (income, expense, transfer) (optional)"`
	IncludeHidden bool   `json:"include_hidden,omitempty" jsonschema_description:"Set to true to include hidden categories (optional, default false)"`
}

// MCPListTransactionCategoriesResponse represents the response structure for listing transaction categories
type MCPListTransactionCategoriesResponse struct {
	Categories []*MCPTransactionCategoryTreeInfo `json:"categories" jsonschema_description:"Primary categories (ordered by type and display order), each with its sub-categories"`
}

type mcpListTransactionCategoriesToolHandler struct{}

var MCPListTransactionCategoriesToolHandler = &mcpListTransactionCategoriesToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpListTransactionCategoriesToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_LIST_TRANSACTION_CATEGORIES
}

// Name returns the name of the MCP tool
func (h *mcpListTransactionCategoriesToolHandler) Name() string {
	return "list_transaction_categories"
}

// Description returns the description of the MCP tool
func (h *mcpListTransactionCategoriesToolHandler) Description() string {
	return "List transaction categories with stable IDs as a two-level hierarchy (primary categories with their sub-categories). Transactions must use sub-categories."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpListTransactionCategoriesToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpListTransactionCategoriesToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpListTransactionCategoriesToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpListTransactionCategoriesToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionCategoriesRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpListTransactionCategoriesToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionCategoriesResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpListTransactionCategoriesToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPListTransactionCategoriesRequest](callToolReq, true)

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
	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, categoryType, -1)

	if err != nil {
		log.Errorf(c, "[list_transaction_categories_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := services.GetTransactionCategoryService().GetCategoryMapByList(allCategories)
	response := MCPListTransactionCategoriesResponse{
		Categories: make([]*MCPTransactionCategoryTreeInfo, 0),
	}

	for i := 0; i < len(allCategories); i++ {
		category := allCategories[i]

		if category.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
			continue
		}

		if !request.IncludeHidden && category.Hidden {
			continue
		}

		subCategories := mcpTransactionCategoryGetSubCategories(allCategories, category.CategoryId, request.IncludeHidden)
		response.Categories = append(response.Categories, mcpTransactionCategoryToTreeInfo(category, subCategories, categoryMap))
	}

	return newMCPJSONTextResult(response)
}
