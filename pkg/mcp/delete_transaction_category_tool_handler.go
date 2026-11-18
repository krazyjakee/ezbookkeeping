package mcp

import (
	"errors"
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPDeleteTransactionCategoryRequest represents all parameters of the delete transaction category request
type MCPDeleteTransactionCategoryRequest struct {
	CategoryId string `json:"category_id" jsonschema_description:"ID of the transaction category to delete (names are not accepted for deletion)"`
	Confirm    bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion"`
}

// MCPDeleteTransactionCategoryResponse represents the response structure for deleting transaction category
type MCPDeleteTransactionCategoryResponse struct {
	Success               bool     `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Committed             bool     `json:"committed" jsonschema_description:"Indicates whether the deletion has been saved"`
	Status                string   `json:"status" jsonschema:"enum=deleted" jsonschema_description:"deleted: the category has been deleted"`
	DeletedCategoryId     string   `json:"deleted_category_id" jsonschema_description:"ID of the deleted category"`
	DeletedCategoryName   string   `json:"deleted_category_name" jsonschema_description:"Name of the deleted category"`
	DeletedSubCategoryIds []string `json:"deleted_sub_category_ids,omitempty" jsonschema_description:"IDs of the sub-categories deleted together with the primary category"`
	Restorable            bool     `json:"restorable" jsonschema_description:"Whether the deletion can be undone (always false)"`
}

type mcpDeleteTransactionCategoryToolHandler struct{}

var MCPDeleteTransactionCategoryToolHandler = &mcpDeleteTransactionCategoryToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionCategoryToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION_CATEGORY
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionCategoryToolHandler) Name() string {
	return "delete_transaction_category"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionCategoryToolHandler) Description() string {
	return "Permanently delete a transaction category by ID (deleting a primary category also deletes all its sub-categories). This cannot be undone. " +
		"The deletion is rejected if the category (or any of its sub-categories) is used by any transaction, transaction template or active scheduled transaction; consider hide_transaction_category instead. " +
		"\"confirm\" must be true."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionCategoryToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionCategoryToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionCategoryToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionCategoryToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionCategoryRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionCategoryToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionCategoryResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionCategoryToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDeleteTransactionCategoryRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	categoryId, err := parseMCPRequiredId("category_id", request.CategoryId)

	if err != nil {
		return nil, nil, err
	}

	if !request.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "\"confirm\" must be true to permanently delete the category")
	}

	uid := user.Uid
	categoryService := services.GetTransactionCategoryService()
	allCategories, err := categoryService.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[delete_transaction_category_tool_handler.Handle] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	categoryMap := categoryService.GetCategoryMapByList(allCategories)
	category, exists := categoryMap[categoryId]

	if !exists || category == nil {
		return nil, nil, NewMCPNotFoundError("category_id", "transaction category", request.CategoryId)
	}

	err = categoryService.DeleteCategory(c, uid, categoryId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_category_tool_handler.Handle] failed to delete category \"id:%d\" for user \"uid:%d\", because %s", categoryId, uid, err.Error())

		var customErr *errs.Error

		if errors.As(err, &customErr) && customErr.Code() == errs.ErrTransactionCategoryInUseCannotBeDeleted.Code() {
			return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "category_id", "the category (or one of its sub-categories) is used by transactions, transaction templates or active scheduled transactions and cannot be deleted, hide it instead")
		}

		return nil, nil, err
	}

	log.Infof(c, "[delete_transaction_category_tool_handler.Handle] user \"uid:%d\" has deleted category \"id:%d\"", uid, categoryId)

	response := MCPDeleteTransactionCategoryResponse{
		Success:             true,
		Committed:           true,
		Status:              MCPWriteStatusDeleted,
		DeletedCategoryId:   utils.Int64ToString(categoryId),
		DeletedCategoryName: category.Name,
		Restorable:          false,
	}

	subCategories := mcpTransactionCategoryGetSubCategories(allCategories, categoryId, true)

	for i := 0; i < len(subCategories); i++ {
		response.DeletedSubCategoryIds = append(response.DeletedSubCategoryIds, utils.Int64ToString(subCategories[i].CategoryId))
	}

	return newMCPJSONTextResult(response)
}
