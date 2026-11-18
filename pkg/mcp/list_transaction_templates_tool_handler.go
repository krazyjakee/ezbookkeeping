package mcp

import (
	"reflect"
	"sort"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPListTransactionTemplatesRequest represents all parameters of the list transaction templates request
type MCPListTransactionTemplatesRequest struct {
	TemplateType  string `json:"template_type,omitempty" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type to list (normal or scheduled, default: normal). Scheduled templates are only available when scheduled transactions are enabled on the server"`
	IncludeHidden bool   `json:"include_hidden,omitempty" jsonschema_description:"Whether to include hidden templates (optional, default: false)"`
}

// MCPListTransactionTemplatesResponse represents the response structure for listing transaction templates
type MCPListTransactionTemplatesResponse struct {
	TemplateType string                        `json:"template_type" jsonschema_description:"Template type of the listed templates"`
	Count        int                           `json:"count" jsonschema_description:"Number of the returned templates"`
	Templates    []*MCPTransactionTemplateInfo `json:"templates" jsonschema_description:"Transaction templates ordered by display order"`
}

type mcpListTransactionTemplatesToolHandler struct{}

var MCPListTransactionTemplatesToolHandler = &mcpListTransactionTemplatesToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpListTransactionTemplatesToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_LIST_TRANSACTION_TEMPLATES
}

// Name returns the name of the MCP tool
func (h *mcpListTransactionTemplatesToolHandler) Name() string {
	return "list_transaction_templates"
}

// Description returns the description of the MCP tool
func (h *mcpListTransactionTemplatesToolHandler) Description() string {
	return "List transaction templates or scheduled transactions with stable IDs. Scheduled templates include schedule status, next occurrence and remaining occurrences (when computable)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpListTransactionTemplatesToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpListTransactionTemplatesToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpListTransactionTemplatesToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpListTransactionTemplatesToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionTemplatesRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpListTransactionTemplatesToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPListTransactionTemplatesResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpListTransactionTemplatesToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPListTransactionTemplatesRequest](callToolReq, true)

	if err != nil {
		return nil, nil, err
	}

	templateType, err := mcpTransactionTemplateParseTemplateType("template_type", request.TemplateType)

	if err != nil {
		return nil, nil, err
	}

	if templateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE && !currentConfig.EnableScheduledTransaction {
		return nil, nil, errs.ErrScheduledTransactionNotEnabled
	}

	uid := user.Uid
	templates, err := services.GetTransactionTemplateService().GetAllTemplatesByUid(c, uid, templateType)

	if err != nil {
		log.Errorf(c, "[list_transaction_templates_tool_handler.Handle] failed to get templates for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	refs, err := mcpTransactionTemplateLoadReferenceData(c, uid, services)

	if err != nil {
		return nil, nil, err
	}

	sort.SliceStable(templates, func(i, j int) bool {
		return templates[i].DisplayOrder < templates[j].DisplayOrder
	})

	nowUnixTime := time.Now().Unix()
	response := &MCPListTransactionTemplatesResponse{
		TemplateType: mcpTransactionTemplateGetTemplateTypeName(templateType),
		Templates:    make([]*MCPTransactionTemplateInfo, 0, len(templates)),
	}

	for i := 0; i < len(templates); i++ {
		if templates[i].Hidden && !request.IncludeHidden {
			continue
		}

		response.Templates = append(response.Templates, mcpTransactionTemplateBuildInfo(templates[i], refs, nowUnixTime))
	}

	response.Count = len(response.Templates)

	return newMCPJSONTextResult(response)
}
