package mcp

import (
	"reflect"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPCreateTransactionTemplateRequest represents all parameters of the create transaction template request
type MCPCreateTransactionTemplateRequest struct {
	TemplateType           string   `json:"template_type,omitempty" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type (normal or scheduled, default: normal). Scheduled templates create transactions automatically and require scheduled transactions to be enabled on the server"`
	Name                   string   `json:"name" jsonschema_description:"Template name"`
	Type                   string   `json:"type" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction type (income, expense, transfer)"`
	CategoryId             string   `json:"category_id,omitempty" jsonschema_description:"Secondary category ID (preferred over category_name)"`
	CategoryName           string   `json:"category_name,omitempty" jsonschema_description:"Secondary category name, used when category_id is not set"`
	AccountId              string   `json:"account_id,omitempty" jsonschema_description:"Source account ID (preferred over account_name)"`
	AccountName            string   `json:"account_name,omitempty" jsonschema_description:"Source account name, used when account_id is not set"`
	DestinationAccountId   string   `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID for transfer templates"`
	DestinationAccountName string   `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name for transfer templates, used when destination_account_id is not set"`
	Amount                 string   `json:"amount,omitempty" jsonschema_description:"Amount (e.g. 12.34, optional, default: 0)"`
	DestinationAmount      string   `json:"destination_amount,omitempty" jsonschema_description:"Destination amount for transfer templates (optional, default: same as amount)"`
	HideAmount             bool     `json:"hide_amount,omitempty" jsonschema_description:"Whether to hide the amount in the web interface (optional)"`
	TagIds                 []string `json:"tag_ids,omitempty" jsonschema_description:"Tag IDs (optional, at most 10 tags in total with tag_names)"`
	TagNames               []string `json:"tag_names,omitempty" jsonschema_description:"Tag names (optional, at most 10 tags in total with tag_ids)"`
	Comment                string   `json:"comment,omitempty" jsonschema_description:"Transaction description (optional, the maximum length is reported by get_server_capabilities)"`
	FrequencyType          string   `json:"frequency_type,omitempty" jsonschema:"enum=disabled,enum=daily,enum=weekly,enum=monthly,enum=yearly,enum=every_n_days" jsonschema_description:"Schedule frequency type (required for scheduled templates)"`
	Frequency              string   `json:"frequency,omitempty" jsonschema_description:"Comma-separated schedule frequency values (weekly: weekday 0-6 with 0 as Sunday; monthly: day of month 1-31, or -1 to -31 counting from the end of month; yearly: MMDD; every_n_days: N; ignored for daily and disabled)"`
	StartDate              string   `json:"start_date,omitempty" jsonschema_description:"Schedule start date in YYYY-MM-DD format (optional, required for every_n_days)"`
	EndDate                string   `json:"end_date,omitempty" jsonschema_description:"Schedule end date (inclusive) in YYYY-MM-DD format (optional)"`
	Timezone               string   `json:"timezone,omitempty" jsonschema_description:"IANA timezone name of the schedule, e.g. Europe/London (required for scheduled templates)"`
	SkipExisting           bool     `json:"skip_existing,omitempty" jsonschema_description:"If true and a template with the same name and template type already exists, the existing template is returned with status 'existing' instead of creating a duplicate (optional)"`
	DryRun                 bool     `json:"dry_run,omitempty" jsonschema_description:"If true, the template is fully validated with the same rules as the actual write but not saved (optional)"`
}

// MCPCreateTransactionTemplateResponse represents the response structure for creating transaction template
type MCPCreateTransactionTemplateResponse struct {
	Success   bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun    bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing is saved)"`
	Committed bool                        `json:"committed" jsonschema_description:"Indicates whether a new template has been saved"`
	Status    string                      `json:"status" jsonschema:"enum=created,enum=existing,enum=would_be_created" jsonschema_description:"Result status"`
	Template  *MCPTransactionTemplateInfo `json:"template" jsonschema_description:"The created (or existing, or validated) template with persisted values"`
}

type mcpCreateTransactionTemplateToolHandler struct{}

var MCPCreateTransactionTemplateToolHandler = &mcpCreateTransactionTemplateToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpCreateTransactionTemplateToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_CREATE_TRANSACTION_TEMPLATE
}

// Name returns the name of the MCP tool
func (h *mcpCreateTransactionTemplateToolHandler) Name() string {
	return "create_transaction_template"
}

// Description returns the description of the MCP tool
func (h *mcpCreateTransactionTemplateToolHandler) Description() string {
	return "Create a transaction template or a scheduled transaction. Accounts, category and tags can be specified by ID or name. Supports dry_run (same validation as the actual write) and skip_existing."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpCreateTransactionTemplateToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpCreateTransactionTemplateToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpCreateTransactionTemplateToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpCreateTransactionTemplateToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTemplateRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpCreateTransactionTemplateToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPCreateTransactionTemplateResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpCreateTransactionTemplateToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPCreateTransactionTemplateRequest](callToolReq, false)

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
	refs, err := mcpTransactionTemplateLoadReferenceData(c, uid, services)

	if err != nil {
		return nil, nil, err
	}

	now := time.Now().Unix()
	input := &mcpTransactionTemplateInput{
		Name:                   request.Name,
		Type:                   request.Type,
		CategoryId:             request.CategoryId,
		CategoryName:           request.CategoryName,
		AccountId:              request.AccountId,
		AccountName:            request.AccountName,
		DestinationAccountId:   request.DestinationAccountId,
		DestinationAccountName: request.DestinationAccountName,
		Amount:                 request.Amount,
		DestinationAmount:      request.DestinationAmount,
		HideAmount:             request.HideAmount,
		TagIds:                 request.TagIds,
		TagNames:               request.TagNames,
		Comment:                request.Comment,
		FrequencyType:          request.FrequencyType,
		Frequency:              request.Frequency,
		StartDate:              request.StartDate,
		EndDate:                request.EndDate,
		Timezone:               request.Timezone,
	}

	template, err := mcpTransactionTemplateBuildModel(input, templateType, refs, uid, now, now)

	if err != nil {
		return nil, nil, err
	}

	if request.SkipExisting {
		existingTemplates, err := services.GetTransactionTemplateService().GetAllTemplatesByUid(c, uid, templateType)

		if err != nil {
			log.Errorf(c, "[create_transaction_template_tool_handler.Handle] failed to get templates for user \"uid:%d\", because %s", uid, err.Error())
			return nil, nil, err
		}

		resolver := mcpTransactionTemplateNewResolver()
		matches := resolver.FindAllByName(existingTemplates, strings.TrimSpace(request.Name))

		if len(matches) > 1 {
			return nil, nil, NewMCPAmbiguousNameError("name", "transaction template", request.Name, resolver.GetCandidates(matches))
		} else if len(matches) == 1 {
			return newMCPJSONTextResult(&MCPCreateTransactionTemplateResponse{
				Success:   true,
				DryRun:    request.DryRun,
				Committed: false,
				Status:    MCPWriteStatusExisting,
				Template:  mcpTransactionTemplateBuildInfo(matches[0], refs, now),
			})
		}
	}

	if request.DryRun {
		template.CreatedUnixTime = now
		return newMCPJSONTextResult(&MCPCreateTransactionTemplateResponse{
			Success:   true,
			DryRun:    true,
			Committed: false,
			Status:    MCPWriteStatusWouldBeCreated,
			Template:  mcpTransactionTemplateBuildInfo(template, refs, now),
		})
	}

	maxOrderId, err := services.GetTransactionTemplateService().GetMaxDisplayOrder(c, uid, templateType)

	if err != nil {
		log.Errorf(c, "[create_transaction_template_tool_handler.Handle] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	template.DisplayOrder = maxOrderId + 1

	err = services.GetTransactionTemplateService().CreateTemplate(c, template)

	if err != nil {
		log.Errorf(c, "[create_transaction_template_tool_handler.Handle] failed to create template for user \"uid:%d\", because %s", uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[create_transaction_template_tool_handler.Handle] user \"uid:%d\" has created a new template \"id:%d\" successfully", uid, template.TemplateId)

	return newMCPJSONTextResult(&MCPCreateTransactionTemplateResponse{
		Success:   true,
		DryRun:    false,
		Committed: true,
		Status:    MCPWriteStatusCreated,
		Template:  mcpTransactionTemplateBuildInfo(template, refs, time.Now().Unix()),
	})
}
