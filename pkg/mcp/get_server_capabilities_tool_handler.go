package mcp

import (
	"reflect"
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/validators"
)

// mcpCapabilitiesAttachmentExtensionCandidates is the list of file extensions which are checked against the server supported attachment types
var mcpCapabilitiesAttachmentExtensionCandidates = []string{"jpg", "jpeg", "png", "gif", "webp", "pdf"}

// MCPGetServerCapabilitiesResponse represents the response structure for getting server capabilities
type MCPGetServerCapabilitiesResponse struct {
	Server                *MCPServerCapabilitiesServerInfo        `json:"server" jsonschema_description:"Server information"`
	Limits                *MCPServerCapabilitiesLimits            `json:"limits" jsonschema_description:"Limits applied to MCP tool inputs, dry runs enforce the same limits as actual writes"`
	Attachments           *MCPServerCapabilitiesAttachments       `json:"attachments" jsonschema_description:"Transaction attachment (receipt picture / document) settings"`
	ScheduledTransactions *MCPServerCapabilitiesScheduled         `json:"scheduled_transactions" jsonschema_description:"Scheduled transaction settings"`
	CurrentUser           *MCPServerCapabilitiesCurrentUser       `json:"current_user" jsonschema_description:"Settings of the current user which affect MCP tools"`
	Features              *MCPServerCapabilitiesFeatures          `json:"features" jsonschema_description:"Supported MCP tool features"`
	SupportedCurrencies   []string                                `json:"supported_currencies" jsonschema_description:"Supported currency codes (ISO 4217)"`
	Values                *MCPServerCapabilitiesEnumerationValues `json:"values" jsonschema_description:"Accepted textual enumeration values of MCP tool inputs"`
}

// MCPServerCapabilitiesServerInfo represents the server information
type MCPServerCapabilitiesServerInfo struct {
	Name               string `json:"name" jsonschema_description:"Application name"`
	Version            string `json:"version,omitempty" jsonschema_description:"Application version"`
	CommitHash         string `json:"commit_hash,omitempty" jsonschema_description:"Commit hash of the application build"`
	MCPProtocolVersion string `json:"mcp_protocol_version" jsonschema_description:"Latest supported Model Context Protocol version"`
}

// MCPServerCapabilitiesLimits represents the limits applied to MCP tool inputs
type MCPServerCapabilitiesLimits struct {
	TransactionDescriptionMaxLength int    `json:"transaction_description_max_length" jsonschema_description:"Maximum length (in characters) of transaction description (comment)"`
	TransactionTagsMaxCount         int    `json:"transaction_tags_max_count" jsonschema_description:"Maximum count of tags of one transaction"`
	TransactionAttachmentsMaxCount  int    `json:"transaction_attachments_max_count" jsonschema_description:"Maximum count of attachments of one transaction"`
	TransactionAmountMin            string `json:"transaction_amount_min" jsonschema_description:"Minimum transaction amount"`
	TransactionAmountMax            string `json:"transaction_amount_max" jsonschema_description:"Maximum transaction amount"`
	AccountNameMaxLength            int    `json:"account_name_max_length" jsonschema_description:"Maximum length (in characters) of account name"`
	AccountDescriptionMaxLength     int    `json:"account_description_max_length" jsonschema_description:"Maximum length (in characters) of account description (comment)"`
	CategoryNameMaxLength           int    `json:"category_name_max_length" jsonschema_description:"Maximum length (in characters) of transaction category name"`
	CategoryDescriptionMaxLength    int    `json:"category_description_max_length" jsonschema_description:"Maximum length (in characters) of transaction category description (comment)"`
	TagNameMaxLength                int    `json:"tag_name_max_length" jsonschema_description:"Maximum length (in characters) of transaction tag name"`
	TagGroupNameMaxLength           int    `json:"tag_group_name_max_length" jsonschema_description:"Maximum length (in characters) of transaction tag group name"`
	TemplateNameMaxLength           int    `json:"template_name_max_length" jsonschema_description:"Maximum length (in characters) of transaction template name"`
	TemplateDescriptionMaxLength    int    `json:"template_description_max_length" jsonschema_description:"Maximum length (in characters) of transaction template description (comment)"`
	BatchMaxItemsCount              int    `json:"batch_max_items_count" jsonschema_description:"Maximum count of items in one batch tool call (e.g. batch_create_transaction_categories, create_transaction_tags)"`
}

// MCPServerCapabilitiesAttachments represents the transaction attachment settings
type MCPServerCapabilitiesAttachments struct {
	Enabled                bool                                   `json:"enabled" jsonschema_description:"Whether transaction attachments are enabled on this server"`
	MaxFileSizeBytes       uint32                                 `json:"max_file_size_bytes" jsonschema_description:"Maximum size (in bytes) of one attachment file"`
	MaxCountPerTransaction int                                    `json:"max_count_per_transaction" jsonschema_description:"Maximum count of attachments of one transaction"`
	SupportedTypes         []*MCPServerCapabilitiesAttachmentType `json:"supported_types" jsonschema_description:"Supported attachment file types"`
	UploadViaMCP           bool                                   `json:"upload_via_mcp" jsonschema_description:"Whether attachments can be uploaded via MCP tools (not supported yet, attachments must be uploaded via the web interface or HTTP API)"`
}

// MCPServerCapabilitiesAttachmentType represents a supported attachment file type
type MCPServerCapabilitiesAttachmentType struct {
	Extension   string `json:"extension" jsonschema_description:"File extension"`
	ContentType string `json:"content_type" jsonschema_description:"Content type (MIME type)"`
}

// MCPServerCapabilitiesScheduled represents the scheduled transaction settings
type MCPServerCapabilitiesScheduled struct {
	Enabled bool `json:"enabled" jsonschema_description:"Whether scheduled transactions are enabled on this server, scheduled transaction templates can only be created or modified when enabled"`
}

// MCPServerCapabilitiesCurrentUser represents the settings of current user
type MCPServerCapabilitiesCurrentUser struct {
	DefaultCurrency       string `json:"default_currency" jsonschema_description:"Default currency of the current user"`
	TransactionEditScope  string `json:"transaction_edit_scope" jsonschema:"enum=none,enum=all,enum=today_or_later,enum=last_24h_or_later,enum=this_week_or_later,enum=this_month_or_later,enum=this_year_or_later,enum=last_reconciled_time_or_later" jsonschema_description:"Range of transaction time which can be created, modified or deleted by the current user"`
	UseLastReconciledTime bool   `json:"use_last_reconciled_time" jsonschema_description:"Whether the current user enables last reconciled time of accounts"`
}

// MCPServerCapabilitiesFeatures represents the supported MCP tool features
type MCPServerCapabilitiesFeatures struct {
	DryRun                string `json:"dry_run" jsonschema_description:"Semantics of the dry_run parameter"`
	SkipExisting          string `json:"skip_existing" jsonschema_description:"Semantics of the skip_existing parameter"`
	NameResolution        string `json:"name_resolution" jsonschema_description:"How names are resolved to entities"`
	StructuredErrors      string `json:"structured_errors" jsonschema_description:"How user-correctable errors are returned"`
	DestructiveOperations string `json:"destructive_operations" jsonschema_description:"Semantics of destructive operations"`
	BatchAtomicity        string `json:"batch_atomicity" jsonschema_description:"Commit semantics of batch tools"`
	LiabilityBalances     string `json:"liability_balances" jsonschema_description:"Semantics of balances of liability accounts"`
	TransactionQueryOrder string `json:"transaction_query_order" jsonschema_description:"Order of transactions returned by query_transactions"`
}

// MCPServerCapabilitiesEnumerationValues represents accepted textual enumeration values of MCP tool inputs
type MCPServerCapabilitiesEnumerationValues struct {
	AccountCategories []string `json:"account_categories" jsonschema_description:"Accepted account categories"`
	TransactionTypes  []string `json:"transaction_types" jsonschema_description:"Accepted transaction types"`
	CategoryTypes     []string `json:"category_types" jsonschema_description:"Accepted transaction category types"`
}

type mcpGetServerCapabilitiesToolHandler struct{}

var MCPGetServerCapabilitiesToolHandler = &mcpGetServerCapabilitiesToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpGetServerCapabilitiesToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_GET_SERVER_CAPABILITIES
}

// Name returns the name of the MCP tool
func (h *mcpGetServerCapabilitiesToolHandler) Name() string {
	return "get_server_capabilities"
}

// Description returns the description of the MCP tool
func (h *mcpGetServerCapabilitiesToolHandler) Description() string {
	return "Get the server version, supported capabilities and limits (description length, attachment types / sizes / counts, batch size, supported currencies, transaction edit scope) of ezBookkeeping. Call this before writing data to validate input in advance."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpGetServerCapabilitiesToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_READ_ONLY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpGetServerCapabilitiesToolHandler) IsIdempotent() bool {
	return true
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpGetServerCapabilitiesToolHandler) DependsOnExternalData() bool {
	return false
}

// InputType returns the input type for the MCP tool request
func (h *mcpGetServerCapabilitiesToolHandler) InputType() reflect.Type {
	return nil
}

// OutputType returns the output type for the MCP tool response
func (h *mcpGetServerCapabilitiesToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPGetServerCapabilitiesResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpGetServerCapabilitiesToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	return newMCPJSONTextResult(h.createNewMCPGetServerCapabilitiesResponse(user, currentConfig))
}

func (h *mcpGetServerCapabilitiesToolHandler) createNewMCPGetServerCapabilitiesResponse(user *models.User, currentConfig *settings.Config) *MCPGetServerCapabilitiesResponse {
	response := &MCPGetServerCapabilitiesResponse{
		Server: &MCPServerCapabilitiesServerInfo{
			Name:               core.ApplicationName,
			Version:            core.Version,
			CommitHash:         core.CommitHash,
			MCPProtocolVersion: string(LatestSupportedMCPVersion),
		},
		Limits: &MCPServerCapabilitiesLimits{
			TransactionDescriptionMaxLength: mcpTransactionCommentMaxLength,
			TransactionTagsMaxCount:         models.MaximumTagsCountOfTransaction,
			TransactionAttachmentsMaxCount:  models.MaximumPicturesCountOfTransaction,
			TransactionAmountMin:            utils.FormatAmount(models.MinimumTransactionAmount),
			TransactionAmountMax:            utils.FormatAmount(models.MaximumTransactionAmount),
			AccountNameMaxLength:            mcpAccountNameMaxLength,
			AccountDescriptionMaxLength:     mcpAccountCommentMaxLength,
			CategoryNameMaxLength:           mcpCategoryNameMaxLength,
			CategoryDescriptionMaxLength:    mcpCategoryCommentMaxLength,
			TagNameMaxLength:                mcpTagNameMaxLength,
			TagGroupNameMaxLength:           mcpTagGroupNameMaxLength,
			TemplateNameMaxLength:           mcpTemplateNameMaxLength,
			TemplateDescriptionMaxLength:    mcpTemplateCommentMaxLength,
			BatchMaxItemsCount:              mcpMaximumBatchItemsCount,
		},
		Attachments: &MCPServerCapabilitiesAttachments{
			MaxCountPerTransaction: models.MaximumPicturesCountOfTransaction,
			SupportedTypes:         make([]*MCPServerCapabilitiesAttachmentType, 0, len(mcpCapabilitiesAttachmentExtensionCandidates)),
			UploadViaMCP:           false,
		},
		ScheduledTransactions: &MCPServerCapabilitiesScheduled{},
		Features: &MCPServerCapabilitiesFeatures{
			DryRun:                "Write tools with dry_run=true perform the same validation as the actual write (including field lengths, edit scope and name resolution) but save nothing; results are marked with dry_run=true and statuses such as would_be_created / would_be_updated, and contain no new IDs.",
			SkipExisting:          "Setup tools with skip_existing=true reuse an existing item with the same name (under the same parent / type / group) and report status \"existing\" with its ID instead of creating a duplicate, so reruns are safe.",
			NameResolution:        "Tools accept stable IDs (preferred) or names. Exact name matches are preferred, then case-insensitive matches. If a name matches more than one item, an \"ambiguous_name\" error listing the candidates is returned instead of choosing one.",
			StructuredErrors:      "User-correctable errors are returned as tool results with isError=true and a JSON body {success:false, error:{code, message, field, limit, actual, value, candidates}}.",
			DestructiveOperations: "Delete tools only accept IDs and require confirm=true. Deletions are permanent, there is no restore via MCP.",
			BatchAtomicity:        "Batch tools validate all items first; if any item is invalid nothing is saved. Committed results report the actual saved state (IDs and statuses) of every item.",
			LiabilityBalances:     "For liability accounts (credit card, debt), balance is the signed accounting balance (negative means money owed) and outstanding_balance is the displayed balance in the web interface (positive means money owed).",
			TransactionQueryOrder: "query_transactions returns transactions ordered by transaction time descending.",
		},
		SupportedCurrencies: h.getSupportedCurrencies(),
		Values: &MCPServerCapabilitiesEnumerationValues{
			AccountCategories: []string{mcpAccountCategoryCash, mcpAccountCategoryCheckingAccount, mcpAccountCategorySavingsAccount, mcpAccountCategoryCreditCard, mcpAccountCategoryVirtual, mcpAccountCategoryDebt, mcpAccountCategoryReceivables, mcpAccountCategoryCertificateOfDeposit, mcpAccountCategoryInvestment},
			TransactionTypes:  []string{transactionTypeIncome, transactionTypeExpense, transactionTypeTransfer, transactionTypeModifyBalance},
			CategoryTypes:     []string{mcpCategoryTypeIncome, mcpCategoryTypeExpense, mcpCategoryTypeTransfer},
		},
	}

	for _, extension := range mcpCapabilitiesAttachmentExtensionCandidates {
		contentType := utils.GetTransactionAttachmentContentType(extension)

		if contentType != "" {
			response.Attachments.SupportedTypes = append(response.Attachments.SupportedTypes, &MCPServerCapabilitiesAttachmentType{
				Extension:   extension,
				ContentType: contentType,
			})
		}
	}

	if currentConfig != nil {
		response.Attachments.Enabled = currentConfig.EnableTransactionPictures
		response.Attachments.MaxFileSizeBytes = currentConfig.MaxTransactionPictureFileSize
		response.ScheduledTransactions.Enabled = currentConfig.EnableScheduledTransaction
	}

	if user != nil {
		response.CurrentUser = &MCPServerCapabilitiesCurrentUser{
			DefaultCurrency:       user.DefaultCurrency,
			TransactionEditScope:  getMCPTransactionEditScopeName(user.TransactionEditScope),
			UseLastReconciledTime: user.UseLastReconciledTime,
		}
	}

	return response
}

func (h *mcpGetServerCapabilitiesToolHandler) getSupportedCurrencies() []string {
	currencies := make([]string, 0, len(validators.AllCurrencyNames))

	for currency, supported := range validators.AllCurrencyNames {
		if supported {
			currencies = append(currencies, currency)
		}
	}

	sort.Strings(currencies)
	return currencies
}

// getMCPTransactionEditScopeName returns the textual transaction edit scope
func getMCPTransactionEditScopeName(scope models.TransactionEditScope) string {
	switch scope {
	case models.TRANSACTION_EDIT_SCOPE_NONE:
		return "none"
	case models.TRANSACTION_EDIT_SCOPE_ALL:
		return "all"
	case models.TRANSACTION_EDIT_SCOPE_TODAY_OR_LATER:
		return "today_or_later"
	case models.TRANSACTION_EDIT_SCOPE_LAST_24H_OR_LATER:
		return "last_24h_or_later"
	case models.TRANSACTION_EDIT_SCOPE_THIS_WEEK_OR_LATER:
		return "this_week_or_later"
	case models.TRANSACTION_EDIT_SCOPE_THIS_MONTH_OR_LATER:
		return "this_month_or_later"
	case models.TRANSACTION_EDIT_SCOPE_THIS_YEAR_OR_LATER:
		return "this_year_or_later"
	case models.TRANSACTION_EDIT_SCOPE_LAST_RECONCILED_TIME_OR_LATER:
		return "last_reconciled_time_or_later"
	default:
		return ""
	}
}
