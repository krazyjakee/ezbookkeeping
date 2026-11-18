package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func createTestDocumentEssentialData() *mcpDocumentEssentialData {
	return &mcpDocumentEssentialData{
		accountMap: map[string]*models.Account{
			"Current Account": {AccountId: 1, Name: "Current Account", Currency: "GBP"},
			"Savings":         {AccountId: 2, Name: "Savings", Currency: "GBP"},
		},
		categoryNameTypes: map[string]map[models.TransactionCategoryType]bool{
			"Groceries": {models.CATEGORY_TYPE_EXPENSE: true},
			"Salary":    {models.CATEGORY_TYPE_INCOME: true},
			"Internal":  {models.CATEGORY_TYPE_TRANSFER: true},
		},
		tagMap: map[string]*models.TransactionTag{
			"business": {TagId: 10, Name: "business"},
		},
	}
}

func TestMCPDocumentBuildAddTransactionRequest_Complete(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	recognized := &MCPRecognizedDocumentTransaction{
		Type:          "expense",
		Time:          "2026-10-01 12:30",
		Amount:        "20.00",
		Currency:      "GBP",
		AccountName:   "Current Account",
		CategoryName:  "Groceries",
		TagNames:      []string{"business", "unknown"},
		Description:   "Weekly shop",
		Supplier:      "Corner Shop Ltd",
		ReceiptNumber: "R-123",
	}

	request, missingFields, warnings := mcpDocumentBuildAddTransactionRequest(recognized, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "receipt.pdf", london)

	assert.Empty(t, missingFields)
	assert.Equal(t, []string{"the recognized tag \"unknown\" does not exist and has been ignored"}, warnings)
	assert.Equal(t, "expense", request.Type)
	assert.Equal(t, "2026-10-01T12:30:00+01:00", request.Time)
	assert.Equal(t, "20.00", request.Amount)
	assert.Equal(t, "Current Account", request.AccountName)
	assert.Equal(t, "Groceries", request.SecondaryCategoryName)
	assert.Equal(t, []string{"business"}, request.Tags)
	assert.Equal(t, "Weekly shop", request.Comment)
	assert.Equal(t, []*MCPTransactionReference{
		{Type: "supplier", Value: "Corner Shop Ltd"},
		{Type: "receipt_number", Value: "R-123"},
		{Type: "source_document", Value: "receipt.pdf"},
	}, request.References)
}

func TestMCPDocumentBuildAddTransactionRequest_MissingFieldsAndOverrides(t *testing.T) {
	recognized := &MCPRecognizedDocumentTransaction{
		Amount:       "9.99",
		Currency:     "USD",
		AccountName:  "Credit Card",
		CategoryName: "Salary",
	}

	request, missingFields, warnings := mcpDocumentBuildAddTransactionRequest(recognized, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "", time.UTC)

	assert.Equal(t, "expense", request.Type)
	assert.Equal(t, []string{"account_id", "category_id"}, missingFields)
	assert.Contains(t, warnings, "the transaction type could not be recognized, expense is assumed")
	assert.Contains(t, warnings, "the transaction time could not be recognized, the current time is used")
	assert.Contains(t, warnings, "the recognized account \"Credit Card\" does not exist and has been ignored")
	assert.Contains(t, warnings, "the recognized category \"Salary\" does not exist for expense transactions and has been ignored")
	assert.NotEmpty(t, request.Time)
	assert.Nil(t, request.References)

	overrides := &MCPDocumentTransactionOverrides{
		AccountId:  "1",
		CategoryId: "5",
		Tags:       []string{"business"},
		Comment:    "Override",
		References: []*MCPTransactionReference{{Type: "external_source_id", Value: "bank-1"}},
	}

	request, missingFields, warnings = mcpDocumentBuildAddTransactionRequest(recognized, overrides, createTestDocumentEssentialData(), "", time.UTC)

	assert.Empty(t, missingFields)
	assert.Equal(t, "1", request.AccountId)
	assert.Equal(t, "", request.AccountName)
	assert.Equal(t, "5", request.CategoryId)
	assert.Equal(t, []string{"business"}, request.Tags)
	assert.Equal(t, "Override", request.Comment)
	assert.Equal(t, []*MCPTransactionReference{{Type: "external_source_id", Value: "bank-1"}}, request.References)
	assert.NotContains(t, warnings, "the recognized account \"Credit Card\" does not exist and has been ignored")
}

func TestMCPDocumentBuildAddTransactionRequest_CurrencyMismatchWarning(t *testing.T) {
	recognized := &MCPRecognizedDocumentTransaction{Type: "expense", Time: "2026-10-01", Amount: "10", Currency: "usd", AccountName: "Current Account", CategoryName: "Groceries"}

	request, missingFields, warnings := mcpDocumentBuildAddTransactionRequest(recognized, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "", time.UTC)

	assert.Empty(t, missingFields)
	assert.Equal(t, "2026-10-01T00:00:00Z", request.Time)
	assert.Contains(t, warnings, "the document amount is in USD but the account \"Current Account\" uses GBP, check the amount")
}

func TestMCPDocumentBuildAddTransactionRequest_Transfer(t *testing.T) {
	recognized := &MCPRecognizedDocumentTransaction{Type: "transfer", Time: "2026-10-01 09:00:00", Amount: "100", AccountName: "Current Account", DestinationAccountName: "Savings", CategoryName: "Internal"}

	request, missingFields, warnings := mcpDocumentBuildAddTransactionRequest(recognized, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "", time.UTC)

	assert.Empty(t, missingFields)
	assert.Equal(t, "Savings", request.DestinationAccountName)
	assert.Equal(t, "100", request.DestinationAmount)
	assert.Contains(t, warnings, "the destination amount could not be recognized, the amount is used")

	recognized.DestinationAccountName = ""
	_, missingFields, _ = mcpDocumentBuildAddTransactionRequest(recognized, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "", time.UTC)
	assert.Equal(t, []string{"destination_account_id"}, missingFields)
}

func TestMCPDocumentBuildAddTransactionRequest_Empty(t *testing.T) {
	_, missingFields, _ := mcpDocumentBuildAddTransactionRequest(&MCPRecognizedDocumentTransaction{}, &MCPDocumentTransactionOverrides{}, createTestDocumentEssentialData(), "", time.UTC)
	assert.Equal(t, []string{"type", "amount", "account_id", "category_id"}, missingFields)
}

func TestMCPDocumentParseTimezone(t *testing.T) {
	location, err := mcpDocumentParseTimezone("timezone", "")
	assert.Nil(t, err)
	assert.Equal(t, time.Local, location)

	location, err = mcpDocumentParseTimezone("timezone", "Europe/London")
	assert.Nil(t, err)
	assert.Equal(t, "Europe/London", location.String())

	location, err = mcpDocumentParseTimezone("timezone", "+05:30")
	assert.Nil(t, err)
	_, offset := time.Date(2026, 1, 1, 0, 0, 0, 0, location).Zone()
	assert.Equal(t, 5*3600+30*60, offset)

	location, err = mcpDocumentParseTimezone("timezone", "-0800")
	assert.Nil(t, err)
	_, offset = time.Date(2026, 1, 1, 0, 0, 0, 0, location).Zone()
	assert.Equal(t, -8*3600, offset)

	location, err = mcpDocumentParseTimezone("timezone", "utc")
	assert.Nil(t, err)
	assert.Equal(t, time.UTC, location)

	_, err = mcpDocumentParseTimezone("timezone", "Mars/Olympus")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTestGetToolError(t, err).Code)

	_, err = mcpDocumentParseTimezone("timezone", "+25:00")
	assert.Equal(t, MCPToolErrorCodeInvalidValue, mcpTestGetToolError(t, err).Code)
}

func TestMCPDocumentTrimJsonResponse(t *testing.T) {
	assert.Equal(t, `{"a":1}`, mcpDocumentTrimJsonResponse(" {\"a\":1} "))
	assert.Equal(t, `{"a":1}`, mcpDocumentTrimJsonResponse("```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, mcpDocumentTrimJsonResponse("```\n{\"a\":1}\n```\n"))
}

func TestMCPAddTransactionFromDocument_BuildRecognitionRequest(t *testing.T) {
	handler := MCPAddTransactionFromDocumentToolHandler
	user := &models.User{Uid: 1}
	csvDocument := &mcpDocumentContent{data: []byte("date,amount\n2026-10-01,20.00"), extension: "csv", originalFileName: "statement.csv"}
	pdfDocument := &mcpDocumentContent{data: []byte("%PDF-1.7"), extension: "pdf"}
	heicDocument := &mcpDocumentContent{data: []byte("\x00\x00\x00\x18ftypheic"), extension: "heic"}

	disabledConfig := &settings.Config{}

	input, llmRequest, truncated, err := handler.buildRecognitionRequest(nil, user, csvDocument, disabledConfig)
	assert.Nil(t, err)
	assert.False(t, truncated)
	assert.Equal(t, mcpDocumentRecognitionInputExtractedText, input)
	assert.Contains(t, string(llmRequest.UserPrompt), "\"statement.csv\" (text/csv)")
	assert.Contains(t, string(llmRequest.UserPrompt), "2026-10-01,20.00")

	_, _, _, err = handler.buildRecognitionRequest(nil, user, pdfDocument, disabledConfig)
	assert.Equal(t, MCPToolErrorCodeNotSupported, mcpTestGetToolError(t, err).Code)

	openAIConfig := &settings.Config{
		TransactionFromAIImageRecognition: true,
		ReceiptImageRecognitionLLMConfig:  &settings.LLMConfig{LLMProvider: settings.OpenAILLMProvider},
		MaxAIRecognitionPictureFileSize:   1 << 20,
	}

	input, llmRequest, _, err = handler.buildRecognitionRequest(nil, user, pdfDocument, openAIConfig)
	assert.Nil(t, err)
	assert.Equal(t, mcpDocumentRecognitionInputDocument, input)
	assert.Equal(t, "application/pdf", llmRequest.UserPromptContentType)
	assert.Equal(t, "document.pdf", llmRequest.UserPromptFileName)

	_, _, _, err = handler.buildRecognitionRequest(nil, user, heicDocument, openAIConfig)
	assert.Equal(t, MCPToolErrorCodeNotSupported, mcpTestGetToolError(t, err).Code)

	ollamaConfig := &settings.Config{
		TransactionFromAIImageRecognition: true,
		ReceiptImageRecognitionLLMConfig:  &settings.LLMConfig{LLMProvider: settings.OllamaLLMProvider},
		MaxAIRecognitionPictureFileSize:   1 << 20,
	}

	_, _, _, err = handler.buildRecognitionRequest(nil, user, pdfDocument, ollamaConfig)
	assert.Equal(t, MCPToolErrorCodeNotSupported, mcpTestGetToolError(t, err).Code)

	smallConfig := &settings.Config{
		TransactionFromAIImageRecognition: true,
		ReceiptImageRecognitionLLMConfig:  &settings.LLMConfig{LLMProvider: settings.GoogleAILLMProvider},
		MaxAIRecognitionPictureFileSize:   4,
	}

	_, _, _, err = handler.buildRecognitionRequest(nil, user, pdfDocument, smallConfig)
	assert.Equal(t, MCPToolErrorCodeTooLarge, mcpTestGetToolError(t, err).Code)

	restrictedUser := &models.User{Uid: 1, FeatureRestriction: core.UserFeatureRestrictions(1 << (core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION - 1))}
	_, _, _, err = handler.buildRecognitionRequest(nil, restrictedUser, pdfDocument, openAIConfig)
	assert.Equal(t, MCPToolErrorCodeNotAllowed, mcpTestGetToolError(t, err).Code)
}

func TestMCPGetServerCapabilities_DocumentRecognition(t *testing.T) {
	handler := MCPGetServerCapabilitiesToolHandler

	result := handler.getDocumentRecognitionCapabilities(&settings.Config{EnableTransactionPictures: true}, &models.User{})
	assert.False(t, result.Enabled)
	assert.Empty(t, result.SupportedTypes)

	result = handler.getDocumentRecognitionCapabilities(&settings.Config{
		EnableTransactionPictures:         true,
		TransactionFromAIImageRecognition: true,
		ReceiptImageRecognitionLLMConfig:  &settings.LLMConfig{LLMProvider: settings.AnthropicLLMProvider},
	}, &models.User{})
	assert.True(t, result.Enabled)
	assert.Contains(t, result.SupportedTypes, "pdf")
	assert.Contains(t, result.SupportedTypes, "docx")
	assert.NotContains(t, result.SupportedTypes, "heic")

	result = handler.getDocumentRecognitionCapabilities(&settings.Config{
		EnableTransactionPictures:        true,
		TransactionFromAITextRecognition: true,
		TextRecognitionLLMConfig:         &settings.LLMConfig{LLMProvider: settings.OllamaLLMProvider},
	}, &models.User{})
	assert.True(t, result.Enabled)
	assert.Equal(t, []string{"docx", "xlsx", "odt", "ods", "txt", "csv"}, result.SupportedTypes)
}
