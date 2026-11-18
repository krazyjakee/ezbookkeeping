package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPHideTransactionCategoryRequest represents all parameters of the hide transaction category request
type MCPHideTransactionCategoryRequest struct {
	CategoryId         string `json:"category_id,omitempty" jsonschema_description:"Transaction category ID (preferred, either category_id or category_name is required)"`
	CategoryName       string `json:"category_name,omitempty" jsonschema_description:"Transaction category name (used when category_id is not set)"`
	Type               string `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction category type to disambiguate category_name (optional)"`
	ParentCategoryName string `json:"parent_category_name,omitempty" jsonschema_description:"Parent category name to disambiguate category_name of a sub-category (optional)"`
	Hidden             *bool  `json:"hidden" jsonschema_description:"true to hide the category, false to unhide it"`
}

// MCPHideTransactionCategoryResponse represents the response structure for hiding transaction category
type MCPHideTransactionCategoryResponse struct {
	Success   bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Committed bool                        `json:"committed" jsonschema_description:"Indicates whether the change has been saved"`
	Status    string                      `json:"status" jsonschema:"enum=updated,enum=unchanged" jsonschema_description:"updated: saved; unchanged: the category already has the requested hidden status"`
	Category  *MCPTransactionCategoryInfo `json:"category" jsonschema_description:"The category after this operation"`
}

type mcpHideTransactionCategoryToolHandler struct{}

var MCPHideTransactionCategoryToolHandler = &mcpHideTransactionCategoryToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpHideTransactionCategoryToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_HIDE_TRANSACTION_CATEGORY
}

// Name returns the name of the MCP tool
func (h *mcpHideTransactionCategoryToolHandler) Name() string {
	return "hide_transaction_category"
}

// Description returns the description of the MCP tool
func (h *mcpHideTransactionCategoryToolHandler) Description() string {
	return "Hide (archive) or unhide a transaction category. Hidden categories are kept with their transactions but are not offered for new transactions."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpHideTransactionCategoryToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpHideTransactionCategoryToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpHideTransactionCategoryToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpHideTransactionCategoryToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPHideTransactionCategoryRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpHideTransactionCategoryToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPHideTransactionCategoryResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpHideTransactionCategoryToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPHideTransactionCategoryRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if request.Hidden == nil {
		return nil, nil, NewMCPRequiredFieldError("hidden")
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[hide_transaction_category_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := categoryService.GetCategoryMapByList(allCategories)
	category, err := mcpTransactionCategoryResolveTarget(allCategories, categoryMap, request.CategoryId, request.CategoryName, request.Type, "", request.ParentCategoryName)

	if err != nil {
		return nil, nil, err
	}

	if category.Hidden == *request.Hidden {
		return newMCPJSONTextResult(MCPHideTransactionCategoryResponse{
			Success:   true,
			Committed: false,
			Status:    MCPWriteStatusUnchanged,
			Category:  mcpTransactionCategoryToInfo(category, categoryMap),
		})
	}

	err = categoryService.HideCategory(c, uid, []int64{category.CategoryId}, *request.Hidden)

	if err != nil {
		log.Errorf(c, "[hide_transaction_category_tool_handler.Handle] failed to hide category \"id:%d\" for user \"uid:%d\", because %s", category.CategoryId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[hide_transaction_category_tool_handler.Handle] user \"uid:%d\" has set hidden status of category \"id:%d\" to %t", uid, category.CategoryId, *request.Hidden)

	category.Hidden = *request.Hidden

	return newMCPJSONTextResult(MCPHideTransactionCategoryResponse{
		Success:   true,
		Committed: true,
		Status:    MCPWriteStatusUpdated,
		Category:  mcpTransactionCategoryToInfo(category, categoryMap),
	})
}
