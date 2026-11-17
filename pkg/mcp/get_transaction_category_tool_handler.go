package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPGetTransactionCategoryRequest represents all parameters of the get transaction category request
type MCPGetTransactionCategoryRequest struct {
	CategoryId         string `json:"category_id,omitempty" jsonschema_description:"Transaction category ID (preferred, either category_id or category_name is required)"`
	CategoryName       string `json:"category_name,omitempty" jsonschema_description:"Transaction category name (used when category_id is not set)"`
	Type               string `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type to disambiguate the name (optional)"`
	ParentCategoryId   string `json:"parent_category_id,omitempty" jsonschema_description:"Parent (primary) category ID to disambiguate the name of a sub-category (optional)"`
	ParentCategoryName string `json:"parent_category_name,omitempty" jsonschema_description:"Parent (primary) category name to disambiguate the name of a sub-category (optional)"`
}

// MCPGetTransactionCategoryResponse represents the response structure for getting transaction category
type MCPGetTransactionCategoryResponse struct {
	Category *MCPTransactionCategoryTreeInfo `json:"category" jsonschema_description:"Transaction category (with all its sub-categories including hidden ones if it is a primary category)"`
}

type mcpGetTransactionCategoryToolHandler struct{}

var MCPGetTransactionCategoryToolHandler = &mcpGetTransactionCategoryToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetTransactionCategoryToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_TRANSACTION_CATEGORY
}

// Name returns the name of the MCP tool
func (h *mcpGetTransactionCategoryToolHandler) Name() string {
	return "get_transaction_category"
}

// Description returns the description of the MCP tool
func (h *mcpGetTransactionCategoryToolHandler) Description() string {
	return "Get a transaction category by ID or by name. If the name matches more than one category, an \"ambiguous_name\" error with all candidates is returned; specify the ID, type or parent category to disambiguate."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetTransactionCategoryToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetTransactionCategoryToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetTransactionCategoryToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetTransactionCategoryToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionCategoryRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetTransactionCategoryToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionCategoryResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetTransactionCategoryToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPGetTransactionCategoryRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	allCategories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[get_transaction_category_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := services.GetTransactionCategoryService().GetCategoryMapByList(allCategories)
	category, err := mcpTransactionCategoryResolveTarget(allCategories, categoryMap, request.CategoryId, request.CategoryName, request.Type, request.ParentCategoryId, request.ParentCategoryName)

	if err != nil {
		return nil, nil, err
	}

	subCategories := mcpTransactionCategoryGetSubCategories(allCategories, category.CategoryId, true)

	response := MCPGetTransactionCategoryResponse{
		Category: mcpTransactionCategoryToTreeInfo(category, subCategories, categoryMap),
	}

	return newMCPJSONTextResult(response)
}
