package mcp

import (
	"reflect"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// MCPDeleteTransactionTemplateRequest represents all parameters of the delete transaction template request
type MCPDeleteTransactionTemplateRequest struct {
	TemplateId string `json:"template_id" jsonschema_description:"ID of the transaction template to delete (names are not accepted for destructive operations)"`
	Confirm    bool   `json:"confirm" jsonschema_description:"Must be true to confirm the permanent deletion"`
}

// MCPDeleteTransactionTemplateResponse represents the response structure for deleting transaction template
type MCPDeleteTransactionTemplateResponse struct {
	Success      bool   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	Committed    bool   `json:"committed" jsonschema_description:"Indicates whether the template has been deleted"`
	Status       string `json:"status" jsonschema:"enum=deleted" jsonschema_description:"Result status"`
	TemplateId   string `json:"template_id" jsonschema_description:"ID of the deleted template"`
	Name         string `json:"name" jsonschema_description:"Name of the deleted template"`
	TemplateType string `json:"template_type" jsonschema_description:"Template type of the deleted template"`
	Permanent    bool   `json:"permanent" jsonschema_description:"Always true, the deleted template cannot be restored through MCP or the web interface"`
}

type mcpDeleteTransactionTemplateToolHandler struct{}

var MCPDeleteTransactionTemplateToolHandler = &mcpDeleteTransactionTemplateToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpDeleteTransactionTemplateToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_DELETE_TRANSACTION_TEMPLATE
}

// Name returns the name of the MCP tool
func (h *mcpDeleteTransactionTemplateToolHandler) Name() string {
	return "delete_transaction_template"
}

// Description returns the description of the MCP tool
func (h *mcpDeleteTransactionTemplateToolHandler) Description() string {
	return "Permanently delete a transaction template or scheduled transaction by ID (requires confirm=true). Transactions already created by a scheduled template are not affected. This cannot be undone."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpDeleteTransactionTemplateToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpDeleteTransactionTemplateToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpDeleteTransactionTemplateToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpDeleteTransactionTemplateToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTemplateRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpDeleteTransactionTemplateToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPDeleteTransactionTemplateResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpDeleteTransactionTemplateToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPDeleteTransactionTemplateRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	templateId, err := parseMCPRequiredId("template_id", request.TemplateId)

	if err != nil {
		return nil, nil, err
	}

	if !request.Confirm {
		return nil, nil, NewMCPToolError(MCPToolErrorCodeNotAllowed, "confirm", "deleting a transaction template is permanent, set \"confirm\" to true to delete it")
	}

	uid := user.Uid
	template, err := services.GetTransactionTemplateService().GetTemplateByTemplateId(c, uid, templateId)

	if err != nil {
		if err == errs.ErrTransactionTemplateNotFound {
			return nil, nil, NewMCPNotFoundError("template_id", "transaction template", request.TemplateId)
		}

		log.Errorf(c, "[delete_transaction_template_tool_handler.Handle] failed to get template \"id:%d\" for user \"uid:%d\", because %s", templateId, uid, err.Error())
		return nil, nil, err
	}

	if template.TemplateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE && !currentConfig.EnableScheduledTransaction {
		return nil, nil, errs.ErrScheduledTransactionNotEnabled
	}

	err = services.GetTransactionTemplateService().DeleteTemplate(c, uid, templateId)

	if err != nil {
		log.Errorf(c, "[delete_transaction_template_tool_handler.Handle] failed to delete template \"id:%d\" for user \"uid:%d\", because %s", templateId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[delete_transaction_template_tool_handler.Handle] user \"uid:%d\" has deleted template \"id:%d\"", uid, templateId)

	return newMCPJSONTextResult(&MCPDeleteTransactionTemplateResponse{
		Success:      true,
		Committed:    true,
		Status:       MCPWriteStatusDeleted,
		TemplateId:   utils.Int64ToString(template.TemplateId),
		Name:         template.Name,
		TemplateType: mcpTransactionTemplateGetTemplateTypeName(template.TemplateType),
		Permanent:    true,
	})
}
