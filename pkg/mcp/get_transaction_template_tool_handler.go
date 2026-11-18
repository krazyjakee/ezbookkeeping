package mcp

import (
	"reflect"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPGetTransactionTemplateRequest represents all parameters of the get transaction template request
type MCPGetTransactionTemplateRequest struct {
	TemplateId   string `json:"template_id,omitempty" jsonschema_description:"Transaction template ID (preferred)"`
	TemplateName string `json:"template_name,omitempty" jsonschema_description:"Transaction template name, used when template_id is not set (an ambiguous name returns the candidates)"`
	TemplateType string `json:"template_type,omitempty" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type to search by name (optional, default: search all types)"`
}

// MCPGetTransactionTemplateResponse represents the response structure for getting transaction template
type MCPGetTransactionTemplateResponse struct {
	Template *MCPTransactionTemplateInfo `json:"template" jsonschema_description:"Transaction template"`
}

type mcpGetTransactionTemplateToolHandler struct{}

var MCPGetTransactionTemplateToolHandler = &mcpGetTransactionTemplateToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetTransactionTemplateToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_TRANSACTION_TEMPLATE
}

// Name returns the name of the MCP tool
func (h *mcpGetTransactionTemplateToolHandler) Name() string {
	return "get_transaction_template"
}

// Description returns the description of the MCP tool
func (h *mcpGetTransactionTemplateToolHandler) Description() string {
	return "Get a transaction template or scheduled transaction by ID or name, including schedule status, next occurrence and remaining occurrences for scheduled templates."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetTransactionTemplateToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetTransactionTemplateToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetTransactionTemplateToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetTransactionTemplateToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionTemplateRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetTransactionTemplateToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetTransactionTemplateResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetTransactionTemplateToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPGetTransactionTemplateRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
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

	refs, err := mcpTransactionTemplateLoadReferenceData(c, uid, services)

	if err != nil {
		return nil, nil, err
	}

	return newMCPJSONTextResult(&MCPGetTransactionTemplateResponse{
		Template: mcpTransactionTemplateBuildInfo(template, refs, time.Now().Unix()),
	})
}

// mcpTransactionTemplateGetSearchTemplateTypes returns the template types to search by name according to the textual template type and the server config
func mcpTransactionTemplateGetSearchTemplateTypes(textualTemplateType string, currentConfig *settings.Config) ([]models.TransactionTemplateType, error) {
	if textualTemplateType == "" {
		if currentConfig.EnableScheduledTransaction {
			return []models.TransactionTemplateType{models.TRANSACTION_TEMPLATE_TYPE_NORMAL, models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE}, nil
		}

		return []models.TransactionTemplateType{models.TRANSACTION_TEMPLATE_TYPE_NORMAL}, nil
	}

	templateType, err := mcpTransactionTemplateParseTemplateType("template_type", textualTemplateType)

	if err != nil {
		return nil, err
	}

	if templateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE && !currentConfig.EnableScheduledTransaction {
		return nil, errs.ErrScheduledTransactionNotEnabled
	}

	return []models.TransactionTemplateType{templateType}, nil
}
