package mcp

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/llm"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/templates"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpDocumentRecognitionMaxTextLength is the maximum length (in characters) of the text extracted from a document which is sent to the large language model
const mcpDocumentRecognitionMaxTextLength = 50000

// mcpDocumentRecognitionNotesMaxLength is the maximum length (in characters) of the additional notes for the large language model
const mcpDocumentRecognitionNotesMaxLength = 2000

// Statuses of add transaction from document results
const (
	mcpDocumentTransactionStatusNeedsInput = "needs_input"
)

// Inputs which are sent to the large language model
const (
	mcpDocumentRecognitionInputImage         = "image"
	mcpDocumentRecognitionInputDocument      = "document"
	mcpDocumentRecognitionInputExtractedText = "extracted_text"
)

// MCPAddTransactionFromDocumentRequest represents all parameters of the add transaction from document request
type MCPAddTransactionFromDocumentRequest struct {
	ContentBase64 string                           `json:"content_base64,omitempty" jsonschema_description:"The document content encoded in base64 (standard alphabet, a \"data:<mime>;base64,\" prefix is allowed), required if attachment_id is not set. The decoded size must not exceed max_file_size_bytes reported by get_server_capabilities. A file name or local path is not a file content and is rejected"`
	FileExtension string                           `json:"file_extension,omitempty" jsonschema_description:"File type of content_base64 as file extension (e.g. jpg, png, pdf, docx, xlsx, csv, the supported types are reported by get_server_capabilities), required if content_type is not set (if neither is set, the extension of file_name is used)"`
	ContentType   string                           `json:"content_type,omitempty" jsonschema_description:"File type of content_base64 as MIME type (e.g. image/jpeg, application/pdf, text/csv), required if file_extension is not set"`
	FileName      string                           `json:"file_name,omitempty" jsonschema_description:"Original file name of the document (optional, e.g. invoice-2026-10.pdf). It is stored as attachment metadata and recorded as source_document reference"`
	AttachmentId  string                           `json:"attachment_id,omitempty" jsonschema_description:"ID of an uploaded but unattached attachment (returned by upload_transaction_attachment) to read instead of content_base64. Use this to avoid sending the same document twice, e.g. after a dry run or a needs_input result"`
	Notes         string                           `json:"notes,omitempty" jsonschema_description:"Additional instructions for the model reading the document (optional, e.g. \"paid with the Amex card\" or \"this is a business expense\")"`
	Timezone      string                           `json:"timezone,omitempty" jsonschema_description:"Timezone of the date and time shown on the document, as IANA timezone name (e.g. Europe/London) or UTC offset (e.g. +01:00). Defaults to the timezone of the server"`
	Overrides     *MCPDocumentTransactionOverrides `json:"overrides,omitempty" jsonschema_description:"Fields which replace the values recognized from the document (optional), e.g. to set the account or category the model could not determine"`
	DryRun        bool                             `json:"dry_run,omitempty" jsonschema_description:"If true, the document is read by the model and the resulting transaction is validated, but nothing is saved (optional). The model may return different values on the next call, to save exactly the previewed values call add_transaction with them"`
}

// MCPDocumentTransactionOverrides represents the fields which replace the values recognized from the document
type MCPDocumentTransactionOverrides struct {
	Type                   string                     `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction type (income, expense, transfer)"`
	Time                   string                     `json:"time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Transaction time in RFC 3339 format (e.g. 2026-10-01T12:00:00+01:00)"`
	Amount                 string                     `json:"amount,omitempty" jsonschema_description:"Transaction amount (e.g. 12.34)"`
	AccountId              string                     `json:"account_id,omitempty" jsonschema_description:"Account ID (takes precedence over account_name)"`
	AccountName            string                     `json:"account_name,omitempty" jsonschema_description:"Account name"`
	CategoryId             string                     `json:"category_id,omitempty" jsonschema_description:"Secondary category ID (takes precedence over category_name)"`
	CategoryName           string                     `json:"category_name,omitempty" jsonschema_description:"Secondary category name"`
	DestinationAccountId   string                     `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID for transfer transactions"`
	DestinationAccountName string                     `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name for transfer transactions"`
	DestinationAmount      string                     `json:"destination_amount,omitempty" jsonschema_description:"Destination amount for transfer transactions"`
	Tags                   []string                   `json:"tags,omitempty" jsonschema_description:"Tag names, replace the recognized tags"`
	TagIds                 []string                   `json:"tag_ids,omitempty" jsonschema_description:"Tag IDs, replace the recognized tags"`
	Comment                string                     `json:"comment,omitempty" jsonschema_description:"Transaction description"`
	References             []*MCPTransactionReference `json:"references,omitempty" jsonschema_description:"Additional evidence / source references, added to the references recognized from the document"`
}

// MCPRecognizedDocumentTransaction represents the transaction information recognized from a document by the large language model
type MCPRecognizedDocumentTransaction struct {
	Type                   string   `json:"type,omitempty" jsonschema_description:"Recognized transaction type"`
	Time                   string   `json:"time,omitempty" jsonschema_description:"Recognized transaction time (YYYY-MM-DD HH:mm:ss, in the requested timezone)"`
	Amount                 string   `json:"amount,omitempty" jsonschema_description:"Recognized amount"`
	Currency               string   `json:"currency,omitempty" jsonschema_description:"Recognized currency code"`
	AccountName            string   `json:"account,omitempty" jsonschema_description:"Recognized account name"`
	CategoryName           string   `json:"category,omitempty" jsonschema_description:"Recognized category name"`
	TagNames               []string `json:"tags,omitempty" jsonschema_description:"Recognized tag names"`
	Description            string   `json:"description,omitempty" jsonschema_description:"Recognized description"`
	DestinationAmount      string   `json:"destination_amount,omitempty" jsonschema_description:"Recognized destination amount"`
	DestinationAccountName string   `json:"destination_account,omitempty" jsonschema_description:"Recognized destination account name"`
	Supplier               string   `json:"supplier,omitempty" jsonschema_description:"Recognized merchant / supplier"`
	InvoiceNumber          string   `json:"invoice_number,omitempty" jsonschema_description:"Recognized invoice number"`
	ReceiptNumber          string   `json:"receipt_number,omitempty" jsonschema_description:"Recognized receipt / order number"`
}

// MCPAddTransactionFromDocumentResponse represents the response structure for add transaction from document
type MCPAddTransactionFromDocumentResponse struct {
	Success                bool                              `json:"success" jsonschema_description:"Indicates whether the transaction has been created (or validated in a dry run)"`
	DryRun                 bool                              `json:"dry_run,omitzero" jsonschema_description:"Indicates whether this operation is a dry run (nothing saved)"`
	Committed              bool                              `json:"committed" jsonschema_description:"Indicates whether anything has been saved (the transaction, or the uploaded document kept as unattached attachment when the status is needs_input)"`
	Status                 string                            `json:"status" jsonschema:"enum=created,enum=would_be_created,enum=needs_input" jsonschema_description:"created: the transaction has been saved with the document attached; would_be_created: dry run passed; needs_input: some required fields could not be recognized or are invalid, see missing_fields / error and next_step"`
	TransactionId          string                            `json:"transaction_id,omitempty" jsonschema_description:"ID of the created transaction"`
	AttachmentId           string                            `json:"attachment_id,omitempty" jsonschema_description:"ID of the saved document attachment (attached to the transaction when created, otherwise kept unattached so it can be reused via attachment_id or attachment_ids of add_transaction)"`
	RecognitionInput       string                            `json:"recognition_input" jsonschema:"enum=image,enum=document,enum=extracted_text" jsonschema_description:"What has been sent to the model: the image, the document file (PDF), or the text extracted from the document"`
	ExtractedTextTruncated bool                              `json:"extracted_text_truncated,omitzero" jsonschema_description:"Whether the text extracted from the document was too long and has been truncated before being sent to the model"`
	Recognized             *MCPRecognizedDocumentTransaction `json:"recognized" jsonschema_description:"The values recognized by the model"`
	Timezone               string                            `json:"timezone" jsonschema_description:"Timezone used to interpret the recognized time"`
	MissingFields          []string                          `json:"missing_fields,omitempty" jsonschema_description:"Required fields which could not be recognized, pass them in overrides (or call add_transaction)"`
	Warnings               []string                          `json:"warnings,omitempty" jsonschema_description:"Recognized values which have been ignored or adjusted, and other things to check"`
	Error                  *MCPToolError                     `json:"error,omitempty" jsonschema_description:"Why the transaction could not be created (when status is needs_input)"`
	Transaction            *MCPAddTransactionResponse        `json:"transaction,omitempty" jsonschema_description:"Result of creating (or validating) the transaction, including the transaction detail and account balances"`
	NextStep               string                            `json:"next_step,omitempty" jsonschema_description:"Suggested next step"`
}

// mcpDocumentContent represents the document which is read by the large language model
type mcpDocumentContent struct {
	data             []byte
	extension        string
	originalFileName string
	attachment       *models.TransactionPictureInfo
}

type mcpAddTransactionFromDocumentToolHandler struct{}

var MCPAddTransactionFromDocumentToolHandler = &mcpAddTransactionFromDocumentToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpAddTransactionFromDocumentToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_ADD_TRANSACTION_FROM_DOCUMENT
}

// Name returns the name of the MCP tool
func (h *mcpAddTransactionFromDocumentToolHandler) Name() string {
	return "add_transaction_from_document"
}

// Description returns the description of the MCP tool
func (h *mcpAddTransactionFromDocumentToolHandler) Description() string {
	return "Create a new transaction from a document which proves it (receipt, invoice, bill, statement entry, payment confirmation: image, PDF, office document, TXT / CSV). " +
		"The server's AI recognition model (configured by the server administrator) reads the document and fills in type, time, amount, account, category, tags, description and references (supplier, invoice / receipt number, source document); " +
		"the document is saved and attached to the new transaction. Pass the document as base64 in content_base64, or the ID of an already uploaded attachment in attachment_id. " +
		"Use overrides to set values the model cannot know (e.g. the account), and notes to give the model extra instructions. " +
		"If required fields cannot be recognized, nothing but the document is saved and status needs_input is returned with missing_fields and the attachment_id to retry with. " +
		"Use dry_run to preview the recognized transaction without saving anything."
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpAddTransactionFromDocumentToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_ADD
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpAddTransactionFromDocumentToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpAddTransactionFromDocumentToolHandler) DependsOnExternalData() bool {
	return true
}

// InputType returns the input type for the MCP tool request
func (h *mcpAddTransactionFromDocumentToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPAddTransactionFromDocumentRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpAddTransactionFromDocumentToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPAddTransactionFromDocumentResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpAddTransactionFromDocumentToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPAddTransactionFromDocumentRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, nil, err
	}

	if err := checkMCPStringLength("notes", request.Notes, false, mcpDocumentRecognitionNotesMaxLength); err != nil {
		return nil, nil, err
	}

	timezone, err := mcpDocumentParseTimezone("timezone", request.Timezone)

	if err != nil {
		return nil, nil, err
	}

	if request.Overrides == nil {
		request.Overrides = &MCPDocumentTransactionOverrides{}
	}

	if request.Overrides.Type != "" && request.Overrides.Type != transactionTypeIncome && request.Overrides.Type != transactionTypeExpense && request.Overrides.Type != transactionTypeTransfer {
		return nil, nil, NewMCPInvalidFieldError("overrides.type", request.Overrides.Type, "\"overrides.type\" must be one of income, expense, transfer")
	}

	document, err := h.getDocument(c, user.Uid, request, currentConfig, services)

	if err != nil {
		return nil, nil, err
	}

	recognitionInput, llmRequest, textTruncated, err := h.buildRecognitionRequest(c, user, document, currentConfig)

	if err != nil {
		return nil, nil, err
	}

	essentialData, err := mcpDocumentGetEssentialData(c, user.Uid, services)

	if err != nil {
		return nil, nil, err
	}

	recognized, err := h.recognize(c, user, request.Notes, timezone, recognitionInput, llmRequest, essentialData, currentConfig)

	if err != nil {
		return nil, nil, err
	}

	response := &MCPAddTransactionFromDocumentResponse{
		RecognitionInput:       recognitionInput,
		ExtractedTextTruncated: textTruncated,
		Recognized:             recognized,
		Timezone:               timezone.String(),
		DryRun:                 request.DryRun,
	}

	if textTruncated {
		response.Warnings = append(response.Warnings, "the text of the document is too long, only the first "+utils.IntToString(mcpDocumentRecognitionMaxTextLength)+" characters have been read")
	}

	addRequest, missingFields, warnings := mcpDocumentBuildAddTransactionRequest(recognized, request.Overrides, essentialData, document.originalFileName, timezone)
	response.Warnings = append(response.Warnings, warnings...)
	response.MissingFields = missingFields

	if document.attachment != nil {
		response.AttachmentId = utils.Int64ToString(document.attachment.PictureId)
	}

	if request.DryRun {
		if document.attachment != nil {
			addRequest.AttachmentIds = []string{response.AttachmentId}
		}

		if len(missingFields) > 0 {
			response.Status = mcpDocumentTransactionStatusNeedsInput
			response.NextStep = "Pass the missing fields (" + strings.Join(missingFields, ", ") + ") in overrides and call again."
			return newMCPJSONTextResult(response)
		}

		addRequest.DryRun = true
		addResponse, toolErr, err := mcpDocumentCallAddTransaction(c, addRequest, user, currentConfig, services)

		if err != nil {
			return nil, nil, err
		}

		if toolErr != nil {
			response.Status = mcpDocumentTransactionStatusNeedsInput
			response.Error = toolErr
			response.NextStep = "Fix \"" + toolErr.Field + "\" via overrides and call again."
			return newMCPJSONTextResult(response)
		}

		response.Success = true
		response.Status = MCPWriteStatusWouldBeCreated
		response.Transaction = addResponse
		return newMCPJSONTextResult(response)
	}

	if document.attachment == nil {
		attachment, err := mcpDocumentSaveAttachment(c, user.Uid, document, services)

		if err != nil {
			return nil, nil, err
		}

		document.attachment = attachment
		response.AttachmentId = utils.Int64ToString(attachment.PictureId)
		response.Committed = true
	}

	retryHint := "Call add_transaction_from_document again with attachment_id \"" + response.AttachmentId + "\" and the missing values in overrides, or call add_transaction with attachment_ids [\"" + response.AttachmentId + "\"]."

	if len(missingFields) > 0 {
		response.Status = mcpDocumentTransactionStatusNeedsInput
		response.NextStep = "The document is saved as unattached attachment \"" + response.AttachmentId + "\" but these fields could not be recognized: " + strings.Join(missingFields, ", ") + ". " + retryHint
		return newMCPJSONTextResult(response)
	}

	addRequest.AttachmentIds = []string{response.AttachmentId}
	addResponse, toolErr, err := mcpDocumentCallAddTransaction(c, addRequest, user, currentConfig, services)

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.Handle] failed to create transaction from document attachment \"id:%d\" for user \"uid:%d\", because %s", document.attachment.PictureId, user.Uid, err.Error())
		return nil, nil, err
	}

	if toolErr != nil {
		response.Status = mcpDocumentTransactionStatusNeedsInput
		response.Error = toolErr
		response.NextStep = "The document is saved as unattached attachment \"" + response.AttachmentId + "\" but the transaction is invalid (" + toolErr.Message + "). " + retryHint
		return newMCPJSONTextResult(response)
	}

	log.Infof(c, "[add_transaction_from_document_tool_handler.Handle] user \"uid:%d\" has created transaction \"id:%s\" from document attachment \"id:%d\" successfully", user.Uid, addResponse.TransactionId, document.attachment.PictureId)

	response.Success = true
	response.Committed = true
	response.Status = MCPWriteStatusCreated
	response.TransactionId = addResponse.TransactionId
	response.Transaction = addResponse

	return newMCPJSONTextResult(response)
}

// getDocument returns the document content from content_base64 or from the stored unattached attachment
func (h *mcpAddTransactionFromDocumentToolHandler) getDocument(c *core.WebContext, uid int64, request *MCPAddTransactionFromDocumentRequest, currentConfig *settings.Config, services MCPAvailableServices) (*mcpDocumentContent, error) {
	if request.AttachmentId != "" {
		if request.ContentBase64 != "" {
			return nil, NewMCPInvalidFieldError("attachment_id", request.AttachmentId, "only one of \"content_base64\" and \"attachment_id\" can be set")
		}

		pictureInfo, err := mcpAttachmentGetPictureInfo(c, uid, "attachment_id", request.AttachmentId, services)

		if err != nil {
			return nil, err
		}

		if pictureInfo.TransactionId != models.TransactionPictureNewPictureTransactionId {
			return nil, NewMCPInvalidFieldError("attachment_id", request.AttachmentId, "attachment \""+request.AttachmentId+"\" is already attached to a transaction, only uploaded but unattached attachments can be used")
		}

		data, contentAvailable, err := mcpAttachmentReadContent(c, uid, pictureInfo, services)

		if err != nil {
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		if !contentAvailable {
			return nil, NewMCPNotFoundError("attachment_id", "attachment file", request.AttachmentId)
		}

		return &mcpDocumentContent{
			data:             data,
			extension:        pictureInfo.PictureExtension,
			originalFileName: pictureInfo.OriginalFileName,
			attachment:       pictureInfo,
		}, nil
	}

	if strings.TrimSpace(request.ContentBase64) == "" {
		return nil, NewMCPRequiredFieldError("content_base64 or attachment_id")
	}

	originalFileName := models.SanitizeTransactionPictureOriginalFileName(request.FileName)
	fileExtension := request.FileExtension

	if fileExtension == "" && request.ContentType == "" && originalFileName != "" {
		fileExtension = utils.GetFileNameExtension(originalFileName)
	}

	extension, err := mcpAttachmentResolveExtension(fileExtension, request.ContentType)

	if err != nil {
		return nil, err
	}

	data, err := mcpAttachmentDecodeContent("content_base64", request.ContentBase64, mcpAttachmentGetMaxFileSize(currentConfig))

	if err != nil {
		return nil, err
	}

	if err := mcpAttachmentValidateContent("content_base64", extension, data); err != nil {
		return nil, err
	}

	return &mcpDocumentContent{
		data:             data,
		extension:        extension,
		originalFileName: originalFileName,
	}, nil
}

// buildRecognitionRequest returns the large language model request (without system prompt) for the document
func (h *mcpAddTransactionFromDocumentToolHandler) buildRecognitionRequest(c *core.WebContext, user *models.User, document *mcpDocumentContent, currentConfig *settings.Config) (string, *data.LargeLanguageModelRequest, bool, error) {
	imageModelEnabled := currentConfig.TransactionFromAIImageRecognition && currentConfig.ReceiptImageRecognitionLLMConfig != nil && currentConfig.ReceiptImageRecognitionLLMConfig.LLMProvider != ""
	imageModelAllowed := !user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION)
	contentType := utils.GetTransactionAttachmentContentType(document.extension)

	if utils.CanExtractTextFromDocument(document.extension) {
		text, truncated, err := utils.ExtractTextFromDocument(document.extension, document.data, mcpDocumentRecognitionMaxTextLength)

		if err != nil {
			log.Warnf(c, "[add_transaction_from_document_tool_handler.buildRecognitionRequest] failed to extract text from \"%s\" document for user \"uid:%d\", because %s", document.extension, user.Uid, err.Error())
			return "", nil, false, NewMCPInvalidFieldError("content_base64", "", "the text of the "+document.extension+" document cannot be read, the file may be damaged or password protected")
		}

		if text == "" {
			return "", nil, false, NewMCPInvalidFieldError("content_base64", "", "the "+document.extension+" document contains no text")
		}

		fileName := document.originalFileName

		if fileName == "" {
			fileName = "document." + document.extension
		}

		userPrompt := "The following is the text content of the document \"" + fileName + "\" (" + contentType + "):\n\n" + text

		return mcpDocumentRecognitionInputExtractedText, &data.LargeLanguageModelRequest{
			UserPrompt:     []byte(userPrompt),
			UserPromptType: data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_TEXT,
		}, truncated, nil
	}

	if !imageModelEnabled {
		return "", nil, false, NewMCPNotSupportedError("", "AI receipt image recognition is not enabled on this server (set transaction_from_ai_image_recognition = true and configure the [llm_image_recognition] section), it is required to read images and PDF documents")
	}

	if !imageModelAllowed {
		return "", nil, false, NewMCPToolError(MCPToolErrorCodeNotAllowed, "", "creating transactions from AI image recognition is restricted for this user")
	}

	if uint32(len(document.data)) > currentConfig.MaxAIRecognitionPictureFileSize {
		return "", nil, false, NewMCPTooLargeError("content_base64", int64(currentConfig.MaxAIRecognitionPictureFileSize), int64(len(document.data)))
	}

	provider := currentConfig.ReceiptImageRecognitionLLMConfig.LLMProvider

	if utils.GetImageContentType(document.extension) != "" {
		return mcpDocumentRecognitionInputImage, &data.LargeLanguageModelRequest{
			UserPrompt:            document.data,
			UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
			UserPromptContentType: contentType,
		}, false, nil
	}

	if document.extension == "heic" || document.extension == "heif" {
		if provider != settings.GoogleAILLMProvider {
			return "", nil, false, NewMCPNotSupportedError("content_base64", "the AI recognition model of this server cannot read HEIC / HEIF images, convert the image to JPEG or PNG first")
		}

		return mcpDocumentRecognitionInputImage, &data.LargeLanguageModelRequest{
			UserPrompt:            document.data,
			UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
			UserPromptContentType: contentType,
		}, false, nil
	}

	if utils.IsPdfFileExtension(document.extension) {
		if provider == settings.OllamaLLMProvider || provider == settings.LMStudioLLMProvider {
			return "", nil, false, NewMCPNotSupportedError("content_base64", "the AI recognition model of this server ("+provider+") cannot read PDF documents, convert the document to an image first or use a provider which supports PDF input (OpenAI, Anthropic, Google AI, OpenRouter)")
		}

		fileName := document.originalFileName

		if fileName == "" {
			fileName = "document.pdf"
		}

		return mcpDocumentRecognitionInputDocument, &data.LargeLanguageModelRequest{
			UserPrompt:            document.data,
			UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_DOCUMENT,
			UserPromptContentType: contentType,
			UserPromptFileName:    fileName,
		}, false, nil
	}

	return "", nil, false, NewMCPNotSupportedError("content_base64", "documents of type \""+document.extension+"\" cannot be read by the AI recognition model")
}

// recognize sends the document to the large language model and returns the recognized transaction
func (h *mcpAddTransactionFromDocumentToolHandler) recognize(c *core.WebContext, user *models.User, notes string, timezone *time.Location, recognitionInput string, llmRequest *data.LargeLanguageModelRequest, essentialData *mcpDocumentEssentialData, currentConfig *settings.Config) (*MCPRecognizedDocumentTransaction, error) {
	systemPrompt, err := templates.GetTemplate(templates.SYSTEM_PROMPT_DOCUMENT_TRANSACTION_RECOGNITION)

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.recognize] failed to get system prompt template for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	additionalNotes := strings.TrimSpace(notes)

	if additionalNotes == "" {
		additionalNotes = "None"
	}

	systemPromptParams := map[string]any{
		"CurrentDateTime":          utils.FormatUnixTimeToLongDateTime(time.Now().Unix(), timezone),
		"AllExpenseCategoryNames":  strings.Join(essentialData.expenseCategoryNames, "\n"),
		"AllIncomeCategoryNames":   strings.Join(essentialData.incomeCategoryNames, "\n"),
		"AllTransferCategoryNames": strings.Join(essentialData.transferCategoryNames, "\n"),
		"AllAccountNames":          strings.Join(essentialData.accountNames, "\n"),
		"AllTagNames":              strings.Join(essentialData.tagNames, "\n"),
		"AdditionalNotes":          additionalNotes,
	}

	var bodyBuffer bytes.Buffer

	if err := systemPrompt.Execute(&bodyBuffer, systemPromptParams); err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.recognize] failed to get final system prompt from template for user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	llmRequest.Stream = false
	llmRequest.SystemPrompt = strings.ReplaceAll(bodyBuffer.String(), "\r\n", "\n")

	var llmResponse *data.LargeLanguageModelTextualResponse
	imageModelEnabled := currentConfig.TransactionFromAIImageRecognition && currentConfig.ReceiptImageRecognitionLLMConfig != nil && currentConfig.ReceiptImageRecognitionLLMConfig.LLMProvider != "" &&
		!user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION)
	textModelEnabled := currentConfig.TransactionFromAITextRecognition && currentConfig.TextRecognitionLLMConfig != nil && currentConfig.TextRecognitionLLMConfig.LLMProvider != "" &&
		!user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_TEXT_RECOGNITION)

	if recognitionInput != mcpDocumentRecognitionInputExtractedText || imageModelEnabled {
		llmResponse, err = llm.Container.GetJsonResponseByReceiptImageRecognitionModel(c, user.Uid, currentConfig, llmRequest)
	} else if textModelEnabled {
		llmResponse, err = llm.Container.GetJsonResponseByTextRecognitionModel(c, user.Uid, currentConfig, llmRequest)
	} else {
		return nil, NewMCPNotSupportedError("", "AI recognition is not enabled on this server (configure the [llm_image_recognition] or [llm_text_recognition] section and enable transaction_from_ai_image_recognition or transaction_from_ai_text_recognition)")
	}

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.recognize] failed to get llm response for user \"uid:%d\", because %s", user.Uid, err.Error())

		if errors.Is(err, errs.ErrLargeLanguageModelProviderNotEnabled) {
			return nil, NewMCPNotSupportedError("", "the AI recognition model is not enabled on this server")
		}

		if errors.Is(err, errs.ErrLargeLanguageModelDocumentNotSupported) {
			return nil, NewMCPNotSupportedError("content_base64", "the AI recognition model of this server cannot read this document type")
		}

		return nil, NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the AI recognition model failed to read the document: "+err.Error())
	}

	content := ""

	if llmResponse != nil {
		content = mcpDocumentTrimJsonResponse(llmResponse.Content)
	}

	recognized := &MCPRecognizedDocumentTransaction{}

	if content != "" {
		if err := json.Unmarshal([]byte(content), recognized); err != nil {
			log.Warnf(c, "[add_transaction_from_document_tool_handler.recognize] failed to parse llm response \"%s\" for user \"uid:%d\", because %s", content, user.Uid, err.Error())
			return nil, NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the AI recognition model returned an invalid result, try again")
		}
	}

	return recognized, nil
}

// mcpDocumentTrimJsonResponse removes markdown code fences which some models put around the json response
func mcpDocumentTrimJsonResponse(content string) string {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(strings.TrimSpace(content), "```")
		content = strings.TrimSpace(content)
	}

	return content
}

// mcpDocumentEssentialData contains the visible accounts, categories and tags of the user which the model can choose from
type mcpDocumentEssentialData struct {
	accountNames          []string
	accountMap            map[string]*models.Account
	expenseCategoryNames  []string
	incomeCategoryNames   []string
	transferCategoryNames []string
	categoryNameTypes     map[string]map[models.TransactionCategoryType]bool
	tagNames              []string
	tagMap                map[string]*models.TransactionTag
}

func mcpDocumentGetEssentialData(c *core.WebContext, uid int64, services MCPAvailableServices) (*mcpDocumentEssentialData, error) {
	accounts, err := services.GetAccountService().GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.mcpDocumentGetEssentialData] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	result := &mcpDocumentEssentialData{
		accountMap:        services.GetAccountService().GetVisibleAccountNameMapByList(accounts),
		categoryNameTypes: make(map[string]map[models.TransactionCategoryType]bool),
	}

	for i := 0; i < len(accounts); i++ {
		if accounts[i].Hidden || accounts[i].Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		result.accountNames = append(result.accountNames, accounts[i].Name)
	}

	categories, err := services.GetTransactionCategoryService().GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.mcpDocumentGetEssentialData] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(categories); i++ {
		category := categories[i]

		if category.Hidden || category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			continue
		}

		if result.categoryNameTypes[category.Name] == nil {
			result.categoryNameTypes[category.Name] = make(map[models.TransactionCategoryType]bool)
		}

		result.categoryNameTypes[category.Name][category.Type] = true

		switch category.Type {
		case models.CATEGORY_TYPE_INCOME:
			result.incomeCategoryNames = append(result.incomeCategoryNames, category.Name)
		case models.CATEGORY_TYPE_EXPENSE:
			result.expenseCategoryNames = append(result.expenseCategoryNames, category.Name)
		case models.CATEGORY_TYPE_TRANSFER:
			result.transferCategoryNames = append(result.transferCategoryNames, category.Name)
		}
	}

	tags, err := services.GetTransactionTagService().GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.mcpDocumentGetEssentialData] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	result.tagMap = services.GetTransactionTagService().GetVisibleTagNameMapByList(tags)

	for i := 0; i < len(tags); i++ {
		if !tags[i].Hidden {
			result.tagNames = append(result.tagNames, tags[i].Name)
		}
	}

	return result, nil
}

// mcpDocumentParseTimezone parses an IANA timezone name or an UTC offset (e.g. +01:00), and returns the server timezone if empty
func mcpDocumentParseTimezone(field string, value string) (*time.Location, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return time.Local, nil
	}

	if value == "Z" || strings.EqualFold(value, "UTC") {
		return time.UTC, nil
	}

	if value[0] == '+' || value[0] == '-' {
		offset, err := time.Parse("-07:00", value)

		if err != nil {
			offset, err = time.Parse("-0700", value)
		}

		if err != nil {
			return nil, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be an IANA timezone name (e.g. Europe/London) or an UTC offset (e.g. +01:00)")
		}

		_, offsetSeconds := offset.Zone()
		return time.FixedZone(value, offsetSeconds), nil
	}

	location, err := time.LoadLocation(value)

	if err != nil {
		return nil, NewMCPInvalidFieldError(field, value, "\""+field+"\" must be an IANA timezone name (e.g. Europe/London) or an UTC offset (e.g. +01:00)")
	}

	return location, nil
}

// mcpDocumentNormalizeLongDateTime converts the date time returned by the model to the long date time format (YYYY-MM-DD HH:mm:ss)
func mcpDocumentNormalizeLongDateTime(dateTime string) string {
	dateTime = strings.TrimSpace(strings.Replace(dateTime, "T", " ", 1))

	if utils.IsValidLongDateTimeFormat(dateTime) {
		return dateTime
	}

	if utils.IsValidLongDateTimeWithoutSecondFormat(dateTime) {
		return dateTime + ":00"
	}

	if utils.IsValidLongDateFormat(dateTime) {
		return dateTime + " 00:00:00"
	}

	return dateTime
}

// mcpDocumentTruncate returns the text truncated to the maximum length in characters
func mcpDocumentTruncate(text string, maxLength int) string {
	text = strings.TrimSpace(text)

	if utf8.RuneCountInString(text) <= maxLength {
		return text
	}

	return strings.TrimSpace(string([]rune(text)[:maxLength]))
}

// mcpDocumentBuildAddTransactionRequest merges the recognized values and the overrides into an add transaction request,
// recognized names which do not exactly match a visible account / category / tag are ignored with a warning, and required fields which are still empty are returned as missing fields
func mcpDocumentBuildAddTransactionRequest(recognized *MCPRecognizedDocumentTransaction, overrides *MCPDocumentTransactionOverrides, essentialData *mcpDocumentEssentialData, originalFileName string, timezone *time.Location) (*MCPAddTransactionRequest, []string, []string) {
	request := &MCPAddTransactionRequest{}
	missingFields := make([]string, 0)
	warnings := make([]string, 0)

	// type
	request.Type = strings.ToLower(strings.TrimSpace(recognized.Type))

	if overrides.Type != "" {
		request.Type = overrides.Type
	} else if request.Type != transactionTypeIncome && request.Type != transactionTypeExpense && request.Type != transactionTypeTransfer {
		if request.Type != "" {
			warnings = append(warnings, "the recognized type \""+recognized.Type+"\" is invalid and has been ignored")
		}

		request.Type = ""

		if recognized.Amount != "" || overrides.Amount != "" {
			request.Type = transactionTypeExpense
			warnings = append(warnings, "the transaction type could not be recognized, expense is assumed")
		} else {
			missingFields = append(missingFields, "type")
		}
	}

	// time
	if overrides.Time != "" {
		request.Time = overrides.Time
	} else if recognized.Time != "" {
		transactionTime, err := utils.ParseFromLongDateTimeInTimeZone(mcpDocumentNormalizeLongDateTime(recognized.Time), timezone)

		if err != nil {
			warnings = append(warnings, "the recognized time \""+recognized.Time+"\" is invalid, the current time is used")
			request.Time = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(time.Now().Unix(), timezone)
		} else {
			request.Time = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(transactionTime.Unix(), timezone)
		}
	} else {
		warnings = append(warnings, "the transaction time could not be recognized, the current time is used")
		request.Time = utils.FormatUnixTimeToLongDateTimeWithTimezoneRFC3339Format(time.Now().Unix(), timezone)
	}

	// amount
	request.Amount = strings.TrimSpace(recognized.Amount)

	if overrides.Amount != "" {
		request.Amount = overrides.Amount
	}

	if request.Amount == "" {
		missingFields = append(missingFields, "amount")
	}

	// account
	if overrides.AccountId != "" || overrides.AccountName != "" {
		request.AccountId = overrides.AccountId
		request.AccountName = overrides.AccountName
	} else if recognized.AccountName != "" && essentialData.accountMap[recognized.AccountName] != nil {
		request.AccountName = recognized.AccountName
	} else {
		if recognized.AccountName != "" {
			warnings = append(warnings, "the recognized account \""+recognized.AccountName+"\" does not exist and has been ignored")
		}

		missingFields = append(missingFields, "account_id")
	}

	if account := essentialData.accountMap[request.AccountName]; account != nil && recognized.Currency != "" && !strings.EqualFold(account.Currency, recognized.Currency) && overrides.Amount == "" {
		warnings = append(warnings, "the document amount is in "+strings.ToUpper(recognized.Currency)+" but the account \""+account.Name+"\" uses "+account.Currency+", check the amount")
	}

	// category
	categoryType := models.CATEGORY_TYPE_EXPENSE

	if request.Type == transactionTypeIncome {
		categoryType = models.CATEGORY_TYPE_INCOME
	} else if request.Type == transactionTypeTransfer {
		categoryType = models.CATEGORY_TYPE_TRANSFER
	}

	if overrides.CategoryId != "" || overrides.CategoryName != "" {
		request.CategoryId = overrides.CategoryId
		request.SecondaryCategoryName = overrides.CategoryName
	} else if recognized.CategoryName != "" && essentialData.categoryNameTypes[recognized.CategoryName][categoryType] {
		request.SecondaryCategoryName = recognized.CategoryName
	} else {
		if recognized.CategoryName != "" {
			warnings = append(warnings, "the recognized category \""+recognized.CategoryName+"\" does not exist for "+request.Type+" transactions and has been ignored")
		}

		missingFields = append(missingFields, "category_id")
	}

	// transfer
	if request.Type == transactionTypeTransfer {
		if overrides.DestinationAccountId != "" || overrides.DestinationAccountName != "" {
			request.DestinationAccountId = overrides.DestinationAccountId
			request.DestinationAccountName = overrides.DestinationAccountName
		} else if recognized.DestinationAccountName != "" && essentialData.accountMap[recognized.DestinationAccountName] != nil {
			request.DestinationAccountName = recognized.DestinationAccountName
		} else {
			if recognized.DestinationAccountName != "" {
				warnings = append(warnings, "the recognized destination account \""+recognized.DestinationAccountName+"\" does not exist and has been ignored")
			}

			missingFields = append(missingFields, "destination_account_id")
		}

		request.DestinationAmount = strings.TrimSpace(recognized.DestinationAmount)

		if overrides.DestinationAmount != "" {
			request.DestinationAmount = overrides.DestinationAmount
		}

		if request.DestinationAmount == "" && request.Amount != "" {
			request.DestinationAmount = request.Amount
			warnings = append(warnings, "the destination amount could not be recognized, the amount is used")
		}
	}

	// tags
	if len(overrides.Tags) > 0 || len(overrides.TagIds) > 0 {
		request.Tags = overrides.Tags
		request.TagIds = overrides.TagIds
	} else {
		for _, tagName := range recognized.TagNames {
			if len(request.Tags) >= models.MaximumTagsCountOfTransaction {
				break
			}

			if essentialData.tagMap[tagName] != nil {
				request.Tags = append(request.Tags, tagName)
			} else if tagName != "" {
				warnings = append(warnings, "the recognized tag \""+tagName+"\" does not exist and has been ignored")
			}
		}
	}

	// comment
	request.Comment = mcpDocumentTruncate(recognized.Description, mcpTransactionCommentMaxLength)

	if overrides.Comment != "" {
		request.Comment = overrides.Comment
	}

	// references
	references := make([]*MCPTransactionReference, 0)
	referenceExists := make(map[string]bool)

	addReference := func(referenceType string, value string) {
		value = mcpDocumentTruncate(value, models.TransactionReferenceValueMaxLength)
		key := referenceType + "\x00" + value

		if value == "" || referenceExists[key] {
			return
		}

		referenceExists[key] = true
		references = append(references, &MCPTransactionReference{Type: referenceType, Value: value})
	}

	addReference(models.TRANSACTION_REFERENCE_TYPE_SUPPLIER, recognized.Supplier)
	addReference(models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER, recognized.InvoiceNumber)
	addReference(models.TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER, recognized.ReceiptNumber)
	addReference(models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT, originalFileName)

	for _, reference := range overrides.References {
		if reference != nil {
			addReference(reference.Type, reference.Value)
		}
	}

	if len(references) > 0 {
		request.References = references
	}

	return request, missingFields, warnings
}

// mcpDocumentCallAddTransaction creates (or validates) the transaction with the add transaction tool, user-correctable errors are returned as tool error
func mcpDocumentCallAddTransaction(c *core.WebContext, addRequest *MCPAddTransactionRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (*MCPAddTransactionResponse, *MCPToolError, error) {
	arguments, err := json.Marshal(addRequest)

	if err != nil {
		return nil, nil, errs.ErrOperationFailed
	}

	result, _, err := MCPAddTransactionToolHandler.Handle(c, &MCPCallToolRequest{
		Name:      MCPAddTransactionToolHandler.Name(),
		Arguments: jsontext.Value(arguments),
	}, user, currentConfig, services)

	if err != nil {
		var toolErr *MCPToolError

		if errors.As(err, &toolErr) {
			return nil, toolErr, nil
		}

		return nil, nil, err
	}

	switch addResponse := result.(type) {
	case *MCPAddTransactionResponse:
		return addResponse, nil, nil
	case MCPAddTransactionResponse:
		return &addResponse, nil, nil
	default:
		log.Errorf(c, "[add_transaction_from_document_tool_handler.mcpDocumentCallAddTransaction] unexpected result type %T of add transaction tool for user \"uid:%d\"", result, user.Uid)
		return nil, nil, errs.ErrOperationFailed
	}
}

// mcpDocumentSaveAttachment saves the document as an unattached transaction attachment
func mcpDocumentSaveAttachment(c *core.WebContext, uid int64, document *mcpDocumentContent, services MCPAvailableServices) (*models.TransactionPictureInfo, error) {
	pictureInfo := &models.TransactionPictureInfo{
		Uid:              uid,
		TransactionId:    models.TransactionPictureNewPictureTransactionId,
		PictureExtension: document.extension,
		CreatedIp:        c.ClientIP(),
		OriginalFileName: document.originalFileName,
		FileSize:         int64(len(document.data)),
		FileSha256:       mcpAttachmentComputeChecksum(document.data),
	}

	err := services.GetTransactionPictureService().UploadPicture(c, pictureInfo, &mcpAttachmentContentFile{Reader: bytes.NewReader(document.data)})

	if err != nil {
		log.Errorf(c, "[add_transaction_from_document_tool_handler.mcpDocumentSaveAttachment] failed to save document for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[add_transaction_from_document_tool_handler.mcpDocumentSaveAttachment] user \"uid:%d\" has saved document attachment \"id:%d\" successfully", uid, pictureInfo.PictureId)

	return pictureInfo, nil
}
