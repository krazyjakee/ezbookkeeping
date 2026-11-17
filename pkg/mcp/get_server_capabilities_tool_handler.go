package mcp

import (
	"reflect"
	"sort"
	"time"

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
	ImportMaxRows                   int    `json:"import_max_rows" jsonschema_description:"Maximum count of rows in one import_transactions call"`
	ExportMaxRows                   int    `json:"export_max_rows" jsonschema_description:"Maximum count of transactions returned by one export_transactions call"`
	TransactionReferencesMaxCount   int    `json:"transaction_references_max_count" jsonschema_description:"Maximum count of evidence / source references of one transaction"`
	TransactionReferenceMaxLength   int    `json:"transaction_reference_max_length" jsonschema_description:"Maximum length (in characters) of the value of an evidence / source reference"`
	IdempotencyKeyMaxLength         int    `json:"idempotency_key_max_length" jsonschema_description:"Maximum length (in characters) of idempotency_key"`
	IdempotencyKeyRetentionDays     int    `json:"idempotency_key_retention_days" jsonschema_description:"How many days an idempotency key and its stored result are kept"`
	OperationIdMaxLength            int    `json:"operation_id_max_length" jsonschema_description:"Maximum length (in characters) of operation_id"`
	AttachmentFileNameMaxLength     int    `json:"attachment_file_name_max_length" jsonschema_description:"Maximum length (in characters) of the stored original file name of an attachment"`
}

// MCPServerCapabilitiesAttachments represents the transaction attachment settings
type MCPServerCapabilitiesAttachments struct {
	Enabled                bool                                   `json:"enabled" jsonschema_description:"Whether transaction attachments are enabled on this server"`
	MaxFileSizeBytes       uint32                                 `json:"max_file_size_bytes" jsonschema_description:"Maximum size (in bytes) of one attachment file"`
	MaxCountPerTransaction int                                    `json:"max_count_per_transaction" jsonschema_description:"Maximum count of attachments of one transaction"`
	SupportedTypes         []*MCPServerCapabilitiesAttachmentType `json:"supported_types" jsonschema_description:"Supported attachment file types"`
	UploadViaMCP           bool                                   `json:"upload_via_mcp" jsonschema_description:"Whether attachments can be uploaded via the upload_transaction_attachment tool (true when attachments are enabled)"`
	UploadEncoding         string                                 `json:"upload_encoding" jsonschema_description:"Encoding of the file content in upload_transaction_attachment"`
	MaxUploadBase64Length  int64                                  `json:"max_upload_base64_length" jsonschema_description:"Maximum length of the base64 encoded content in upload_transaction_attachment (derived from max_file_size_bytes)"`
	ContentValidation      string                                 `json:"content_validation" jsonschema_description:"How uploaded content is validated"`
	Metadata               string                                 `json:"metadata" jsonschema_description:"Which attachment metadata is stored and returned"`
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
	Idempotency           string `json:"idempotency" jsonschema_description:"Semantics of the idempotency_key parameter of write tools"`
	AuditLog              string `json:"audit_log" jsonschema_description:"Which changes are recorded in the audit history (query_audit_log)"`
	References            string `json:"references" jsonschema_description:"Semantics of evidence / source references of transactions"`
	Import                string `json:"import" jsonschema_description:"Commit semantics of import_transactions"`
	Reconciliation        string `json:"reconciliation" jsonschema_description:"Supported reconciliation actions"`
}

// MCPServerCapabilitiesEnumerationValues represents accepted textual enumeration values of MCP tool inputs
type MCPServerCapabilitiesEnumerationValues struct {
	AccountCategories []string `json:"account_categories" jsonschema_description:"Accepted account categories"`
	TransactionTypes  []string `json:"transaction_types" jsonschema_description:"Accepted transaction types"`
	CategoryTypes     []string `json:"category_types" jsonschema_description:"Accepted transaction category types"`
	ReferenceTypes    []string `json:"reference_types" jsonschema_description:"Accepted evidence / source reference types"`
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
	return "Get the server version, supported capabilities and limits (description length, attachment types / sizes / counts, batch and import size, export rows, reference and idempotency key limits, supported currencies, transaction edit scope) of ezBookkeeping. Call this before writing data to validate input in advance."
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
			ImportMaxRows:                   mcpImportMaxRows,
			ExportMaxRows:                   mcpExportMaxRows,
			TransactionReferencesMaxCount:   models.MaximumReferencesCountOfTransaction,
			TransactionReferenceMaxLength:   models.TransactionReferenceValueMaxLength,
			IdempotencyKeyMaxLength:         models.MCPIdempotencyKeyMaxLength,
			IdempotencyKeyRetentionDays:     int(mcpIdempotencyRecordRetention / (24 * time.Hour)),
			OperationIdMaxLength:            mcpOperationIdMaxLength,
			AttachmentFileNameMaxLength:     models.TransactionPictureOriginalFileNameMaxLength,
		},
		Attachments: &MCPServerCapabilitiesAttachments{
			MaxCountPerTransaction: models.MaximumPicturesCountOfTransaction,
			SupportedTypes:         make([]*MCPServerCapabilitiesAttachmentType, 0, len(mcpCapabilitiesAttachmentExtensionCandidates)),
			UploadViaMCP:           false,
			UploadEncoding:         "base64 (RFC 4648 standard alphabet) in content_base64, a \"data:<mime>;base64,\" prefix is allowed. File names or local paths are never treated as uploads.",
			ContentValidation:      "The file type is taken from file_extension or content_type, and the content must match it (magic bytes for PDF documents and detected type for images), otherwise the upload is rejected.",
			Metadata:               "Attachment ID, transaction association, stored file extension / MIME type, and the original file name (sanitized), byte size and SHA-256 checksum recorded at upload (pass file_name to upload_transaction_attachment). The SHA-256 checksum is also computed from the stored file when an attachment is read (integrity_verified compares both); attachments uploaded by older versions have no recorded metadata and use the computed checksum.",
		},
		ScheduledTransactions: &MCPServerCapabilitiesScheduled{},
		Features: &MCPServerCapabilitiesFeatures{
			DryRun:                "Write tools with dry_run=true perform the same validation as the actual write (including field lengths, edit scope and name resolution) but save nothing; results are marked with dry_run=true and statuses such as would_be_created / would_be_updated, and contain no new IDs.",
			SkipExisting:          "Setup tools with skip_existing=true reuse an existing item with the same name (under the same parent / type / group) and report status \"existing\" with its ID instead of creating a duplicate, so reruns are safe.",
			NameResolution:        "Tools accept stable IDs (preferred) or names. Exact name matches are preferred, then case-insensitive matches. If a name matches more than one item, an \"ambiguous_name\" error listing the candidates is returned instead of choosing one.",
			StructuredErrors:      "User-correctable errors are returned as tool results with isError=true and a JSON body {success:false, error:{code, message, field, limit, actual, value, candidates}}.",
			DestructiveOperations: "Delete tools only accept IDs and require confirm=true. Deletions are permanent, there is no restore via MCP. detach_transaction_attachment only removes the reference from the transaction and keeps the file, delete_transaction_attachment deletes it.",
			BatchAtomicity:        "Batch tools validate all items first; if any item is invalid nothing is saved. Committed results report the actual saved state (IDs and statuses) of every item.",
			LiabilityBalances:     "For liability accounts (credit card, debt), balance is the signed accounting balance (negative means money owed) and outstanding_balance is the displayed balance in the web interface (positive means money owed).",
			TransactionQueryOrder: "query_transactions returns transactions ordered by transaction time descending.",
			Idempotency:           "All write tools accept an optional idempotency_key (and operation_id). Retrying the same request (same tool and arguments) with the same key returns the stored original result marked with idempotent_replay=true without executing the write again; the same key with a different request fails with idempotency_conflict, and a key whose request is still running fails with idempotency_in_progress. Keys are kept per user for idempotency_key_retention_days and are ignored for dry runs. A failed write does not consume the key.",
			AuditLog:              "Every committed change made through MCP write tools (not dry runs, unchanged results or reads) is recorded with the tool, action, entity type, affected IDs, operation_id, idempotency_key, token ID, client IP and time, and can be queried with query_audit_log.",
			References:            "Transactions can have structured evidence / source references (type and value): supplier, invoice_number, receipt_number, external_source_id, source_document and source_row. They are set by add_transaction / update_transaction / import_transactions, returned by get_transaction / query_transactions / export_transactions, and query_transactions can filter by reference_value.",
			Import:                "import_transactions defaults to dry_run=true. " + mcpImportBatchSemantics,
			Reconciliation:        "get_account_reconciliation_statement returns the statement of an account, set_account_reconciled_time marks an account as reconciled up to a time (requires current_user.use_last_reconciled_time), and export_transactions exports transactions with IDs, attachment checksums and references as CSV or JSON.",
		},
		SupportedCurrencies: h.getSupportedCurrencies(),
		Values: &MCPServerCapabilitiesEnumerationValues{
			AccountCategories: []string{mcpAccountCategoryCash, mcpAccountCategoryCheckingAccount, mcpAccountCategorySavingsAccount, mcpAccountCategoryCreditCard, mcpAccountCategoryVirtual, mcpAccountCategoryDebt, mcpAccountCategoryReceivables, mcpAccountCategoryCertificateOfDeposit, mcpAccountCategoryInvestment},
			TransactionTypes:  []string{transactionTypeIncome, transactionTypeExpense, transactionTypeTransfer, transactionTypeModifyBalance},
			CategoryTypes:     []string{mcpCategoryTypeIncome, mcpCategoryTypeExpense, mcpCategoryTypeTransfer},
			ReferenceTypes:    models.AllTransactionReferenceTypes,
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
		response.Attachments.UploadViaMCP = currentConfig.EnableTransactionPictures
		response.Attachments.MaxUploadBase64Length = mcpAttachmentGetMaxBase64Length(int64(currentConfig.MaxTransactionPictureFileSize))
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
