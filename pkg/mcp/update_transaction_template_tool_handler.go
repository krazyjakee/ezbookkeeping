package mcp

import (
	"reflect"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// MCPUpdateTransactionTemplateRequest represents all parameters of the update transaction template request
type MCPUpdateTransactionTemplateRequest struct {
	TemplateId             string    `json:"template_id,omitempty" jsonschema_description:"ID of the template to update (preferred)"`
	TemplateName           string    `json:"template_name,omitempty" jsonschema_description:"Name of the template to update, used when template_id is not set (an ambiguous name returns the candidates)"`
	TemplateType           string    `json:"template_type,omitempty" jsonschema:"enum=normal,enum=scheduled" jsonschema_description:"Template type to search by template_name (optional, default: search all types). The template type itself cannot be changed"`
	Name                   *string   `json:"name,omitempty" jsonschema_description:"New template name (optional)"`
	Type                   *string   `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"New transaction type (optional, the category must match the type)"`
	CategoryId             *string   `json:"category_id,omitempty" jsonschema_description:"New secondary category ID (optional)"`
	CategoryName           *string   `json:"category_name,omitempty" jsonschema_description:"New secondary category name (optional, used when category_id is not set)"`
	AccountId              *string   `json:"account_id,omitempty" jsonschema_description:"New source account ID (optional)"`
	AccountName            *string   `json:"account_name,omitempty" jsonschema_description:"New source account name (optional, used when account_id is not set)"`
	DestinationAccountId   *string   `json:"destination_account_id,omitempty" jsonschema_description:"New destination account ID for transfer templates (optional)"`
	DestinationAccountName *string   `json:"destination_account_name,omitempty" jsonschema_description:"New destination account name for transfer templates (optional)"`
	Amount                 *string   `json:"amount,omitempty" jsonschema_description:"New amount, e.g. 12.34 (optional)"`
	DestinationAmount      *string   `json:"destination_amount,omitempty" jsonschema_description:"New destination amount for transfer templates (optional)"`
	HideAmount             *bool     `json:"hide_amount,omitempty" jsonschema_description:"Whether to hide the amount in the web interface (optional)"`
	TagIds                 *[]string `json:"tag_ids,omitempty" jsonschema_description:"Replace all tags with these tag IDs (optional, combined with tag_names, empty array removes all tags)"`
	TagNames               *[]string `json:"tag_names,omitempty" jsonschema_description:"Replace all tags with these tag names (optional, combined with tag_ids)"`
	Comment                *string   `json:"comment,omitempty" jsonschema_description:"New transaction description (optional)"`
	Enabled                *bool     `json:"enabled,omitempty" jsonschema_description:"Scheduled templates only: false disables the schedule (frequency_type becomes disabled), true requires a non-disabled frequency_type (optional)"`
	FrequencyType          *string   `json:"frequency_type,omitempty" jsonschema:"enum=disabled,enum=daily,enum=weekly,enum=monthly,enum=yearly,enum=every_n_days" jsonschema_description:"New schedule frequency type (scheduled templates only, optional)"`
	Frequency              *string   `json:"frequency,omitempty" jsonschema_description:"New comma-separated schedule frequency values (scheduled templates only, optional)"`
	StartDate              *string   `json:"start_date,omitempty" jsonschema_description:"New schedule start date in YYYY-MM-DD format, empty string removes it (scheduled templates only, optional)"`
	EndDate                *string   `json:"end_date,omitempty" jsonschema_description:"New schedule end date in YYYY-MM-DD format, empty string removes it (scheduled templates only, optional)"`
	Timezone               *string   `json:"timezone,omitempty" jsonschema_description:"New IANA timezone name of the schedule (scheduled templates only, optional)"`
	DryRun                 bool      `json:"dry_run,omitempty" jsonschema_description:"If true, the update is fully validated with the same rules as the actual write but not saved (optional)"`
}

// MCPUpdateTransactionTemplateResponse represents the response structure for updating transaction template
type MCPUpdateTransactionTemplateResponse struct {
	Success   bool                        `json:"success" jsonschema_description:"Indicates whether this operation is successful"`
	DryRun    bool                        `json:"dry_run" jsonschema_description:"Indicates whether this operation is a dry run (nothing is saved)"`
	Committed bool                        `json:"committed" jsonschema_description:"Indicates whether the changes have been saved"`
	Status    string                      `json:"status" jsonschema:"enum=updated,enum=unchanged,enum=would_be_updated" jsonschema_description:"Result status"`
	Template  *MCPTransactionTemplateInfo `json:"template" jsonschema_description:"The template with persisted (or validated) values"`
}

type mcpUpdateTransactionTemplateToolHandler struct{}

var MCPUpdateTransactionTemplateToolHandler = &mcpUpdateTransactionTemplateToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpUpdateTransactionTemplateToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_UPDATE_TRANSACTION_TEMPLATE
}

// Name returns the name of the MCP tool
func (h *mcpUpdateTransactionTemplateToolHandler) Name() string {
	return "update_transaction_template"
}

// Description returns the description of the MCP tool
func (h *mcpUpdateTransactionTemplateToolHandler) Description() string {
	return "Update a transaction template or scheduled transaction. Only the provided fields are changed. Use enabled=false to stop a scheduled transaction. Supports dry_run (same validation as the actual write)."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpUpdateTransactionTemplateToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpUpdateTransactionTemplateToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpUpdateTransactionTemplateToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpUpdateTransactionTemplateToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTemplateRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpUpdateTransactionTemplateToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPUpdateTransactionTemplateResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpUpdateTransactionTemplateToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPUpdateTransactionTemplateRequest](callToolReq, false)

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

	input, err := h.mergeInput(template, request)

	if err != nil {
		return nil, nil, err
	}

	refs, err := mcpTransactionTemplateLoadReferenceData(c, uid, services)

	if err != nil {
		return nil, nil, err
	}

	now := time.Now().Unix()
	newTemplate, err := mcpTransactionTemplateBuildModel(input, template.TemplateType, refs, uid, template.CreatedUnixTime, now)

	if err != nil {
		return nil, nil, err
	}

	newTemplate.TemplateId = template.TemplateId
	newTemplate.DisplayOrder = template.DisplayOrder
	newTemplate.Hidden = template.Hidden
	newTemplate.CreatedUnixTime = template.CreatedUnixTime

	if template.TemplateType == models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE && request.Timezone == nil && newTemplate.ScheduledTimezoneName == template.ScheduledTimezoneName {
		// keep the stored utc offset if the timezone is not changed, the same as the web interface does
		newTemplate.ScheduledTimezoneUtcOffset = template.ScheduledTimezoneUtcOffset
	}

	if mcpTransactionTemplateIsSame(template, newTemplate) {
		return newMCPJSONTextResult(&MCPUpdateTransactionTemplateResponse{
			Success:   true,
			DryRun:    request.DryRun,
			Committed: false,
			Status:    MCPWriteStatusUnchanged,
			Template:  mcpTransactionTemplateBuildInfo(template, refs, now),
		})
	}

	if request.DryRun {
		return newMCPJSONTextResult(&MCPUpdateTransactionTemplateResponse{
			Success:   true,
			DryRun:    true,
			Committed: false,
			Status:    MCPWriteStatusWouldBeUpdated,
			Template:  mcpTransactionTemplateBuildInfo(newTemplate, refs, now),
		})
	}

	err = services.GetTransactionTemplateService().ModifyTemplate(c, newTemplate)

	if err != nil {
		log.Errorf(c, "[update_transaction_template_tool_handler.Handle] failed to update template \"id:%d\" for user \"uid:%d\", because %s", template.TemplateId, uid, err.Error())
		return nil, nil, err
	}

	log.Infof(c, "[update_transaction_template_tool_handler.Handle] user \"uid:%d\" has updated template \"id:%d\" successfully", uid, template.TemplateId)

	return newMCPJSONTextResult(&MCPUpdateTransactionTemplateResponse{
		Success:   true,
		DryRun:    false,
		Committed: true,
		Status:    MCPWriteStatusUpdated,
		Template:  mcpTransactionTemplateBuildInfo(newTemplate, refs, time.Now().Unix()),
	})
}

// mergeInput returns the textual input of the existed template overridden by the provided fields of the request
func (h *mcpUpdateTransactionTemplateToolHandler) mergeInput(template *models.TransactionTemplate, request *MCPUpdateTransactionTemplateRequest) (*mcpTransactionTemplateInput, error) {
	input := mcpTransactionTemplateBuildInputFromTemplate(template)

	if request.Name != nil {
		input.Name = *request.Name
	}

	if request.Type != nil {
		input.Type = *request.Type
	}

	if request.CategoryId != nil || request.CategoryName != nil {
		input.CategoryId = h.getStringValue(request.CategoryId)
		input.CategoryName = h.getStringValue(request.CategoryName)
	}

	if request.AccountId != nil || request.AccountName != nil {
		input.AccountId = h.getStringValue(request.AccountId)
		input.AccountName = h.getStringValue(request.AccountName)
	}

	if request.DestinationAccountId != nil || request.DestinationAccountName != nil {
		input.DestinationAccountId = h.getStringValue(request.DestinationAccountId)
		input.DestinationAccountName = h.getStringValue(request.DestinationAccountName)
	}

	if request.Amount != nil {
		input.Amount = *request.Amount
	}

	if request.DestinationAmount != nil {
		input.DestinationAmount = *request.DestinationAmount
	}

	if request.HideAmount != nil {
		input.HideAmount = *request.HideAmount
	}

	if request.TagIds != nil || request.TagNames != nil {
		input.TagIds = nil
		input.TagNames = nil

		if request.TagIds != nil {
			input.TagIds = *request.TagIds
		}

		if request.TagNames != nil {
			input.TagNames = *request.TagNames
		}
	}

	if request.Comment != nil {
		input.Comment = *request.Comment
	}

	if template.TemplateType != models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE {
		if request.Enabled != nil || request.FrequencyType != nil || request.Frequency != nil || request.StartDate != nil || request.EndDate != nil || request.Timezone != nil {
			return nil, NewMCPNotSupportedError("enabled", "schedule fields (enabled, frequency_type, frequency, start_date, end_date, timezone) are only supported for scheduled templates")
		}

		return input, nil
	}

	if request.FrequencyType != nil {
		input.FrequencyType = *request.FrequencyType

		if request.Frequency == nil {
			input.Frequency = ""
		}
	}

	if request.Frequency != nil {
		input.Frequency = *request.Frequency
	}

	if request.StartDate != nil {
		input.StartDate = *request.StartDate
	}

	if request.EndDate != nil {
		input.EndDate = *request.EndDate
	}

	if request.Timezone != nil {
		input.Timezone = *request.Timezone
	}

	if request.Enabled != nil {
		if !*request.Enabled {
			if request.FrequencyType != nil && *request.FrequencyType != mcpTransactionTemplateFrequencyDisabled {
				return nil, NewMCPInvalidFieldError("frequency_type", *request.FrequencyType, "\"frequency_type\" conflicts with enabled=false")
			}

			input.FrequencyType = mcpTransactionTemplateFrequencyDisabled
			input.Frequency = ""
		} else if input.FrequencyType == mcpTransactionTemplateFrequencyDisabled || input.FrequencyType == "" {
			if request.FrequencyType == nil {
				return nil, NewMCPRequiredFieldError("frequency_type")
			}

			return nil, NewMCPInvalidFieldError("frequency_type", *request.FrequencyType, "\"frequency_type\" must not be disabled when enabled=true")
		}
	}

	return input, nil
}

func (h *mcpUpdateTransactionTemplateToolHandler) getStringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
