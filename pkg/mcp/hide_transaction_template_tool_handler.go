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

// MCPHideTransactionTemplateRequest represents all parameters of the hide transaction template request
type MCPHideTransactionTemplateRequest struct {
	TemplateId   string `json:"template_id,omitempty" jsonschema_description:"Transaction template ID (preferred)"`
	TemplateName string `json:"template_name,omitempty" jsonschema_description:"Transaction template name, used when template_id is not set"`
	TemplateType string `json:"template_type,omitempty" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type to search by template_name (optional, default: search all types)"`
	Hidden       *bool  `json:"hidden" jsonschema_description:"true to hide the template, false to unhide it"`
}

// MCPHideTransactionTemplateResponse represents the response structure for hiding transaction template
type MCPHideTransactionTemplateResponse struct {
	Success    bool   `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	TemplateId string `json:"template_id" jsonschema_description:"Transaction template ID"`
	Name       string `json:"name" jsonschema_description:"Transaction template name"`
	Hidden     bool   `json:"hidden" jsonschema_description:"Current hidden status of the template"`
}

type mcpHideTransactionTemplateToolHandler struct{}

var MCPHideTransactionTemplateToolHandler = &mcpHideTransactionTemplateToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpHideTransactionTemplateToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_HIDE_TRANSACTION_TEMPLATE
}

// Name returns the name of the MCP tool
func (h *mcpHideTransactionTemplateToolHandler) Name() string {
	return "hide_transaction_template"
}

// Description returns the description of the MCP tool
func (h *mcpHideTransactionTemplateToolHandler) Description() string {
	return "Hide or unhide a transaction template. Hiding does NOT stop a scheduled transaction from being created, use update_transaction_template with enabled=false to stop it."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpHideTransactionTemplateToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpHideTransactionTemplateToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpHideTransactionTemplateToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpHideTransactionTemplateToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPHideTransactionTemplateRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpHideTransactionTemplateToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPHideTransactionTemplateResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpHideTransactionTemplateToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPHideTransactionTemplateRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if request.Hidden == nil {
		return nil, nil, NewMCPRequiredFieldError("hidden")
	}

	templateTypes, err := mcpTransactionTemplateGetSearchTemplateTypes(request.TemplateType, currentConfig)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	template, err := mcpTransactionTemplateFindTemplate(c, uid, services, request.TemplateId, request.TemplateName, templateTypes)

	if err != nil {
		return nil, nil, err
	}

	if template.TemplateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE && !currentConfig.EnableScheduledTransaction {
		return nil, nil, errs.ErrScheduledTransactionNotEnabled
	}

	if template.Hidden != *request.Hidden {
		err = services.GetTransactionTemplateService().HideTemplate(c, uid, []int64{template.TemplateId}, *request.Hidden)

		if err != nil {
			log.Errorf(c, "[hide_transaction_template_tool_handler.Handle] failed to hide template \"id:%d\" for user \"uid:%d\", because %s", template.TemplateId, uid, err.Error())
			return nil, nil, err
		}

		log.Infof(c, "[hide_transaction_template_tool_handler.Handle] user \"uid:%d\" has set hidden status of template \"id:%d\" to %t", uid, template.TemplateId, *request.Hidden)
	}

	return newMCPJSONTextResult(&MCPHideTransactionTemplateResponse{
		Success:    true,
		TemplateId: utils.Int64ToString(template.TemplateId),
		Name:       template.Name,
		Hidden:     *request.Hidden,
	})
}
