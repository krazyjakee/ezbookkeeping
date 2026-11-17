package mcp

import (
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

var mcpAttachmentTestPngContent = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
var mcpAttachmentTestPdfContent = []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")

func getMCPAttachmentTestToolError(t *testing.T, err error) *MCPToolError {
	var toolErr *MCPToolError

	if !assert.True(t, errors.As(err, &toolErr)) {
		t.FailNow()
	}

	return toolErr
}

func TestMCPAttachmentDecodeContent(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(mcpAttachmentTestPdfContent)

	data, err := mcpAttachmentDecodeContent("content_base64", encoded, 1024)
	assert.Nil(t, err)
	assert.Equal(t, mcpAttachmentTestPdfContent, data)

	data, err = mcpAttachmentDecodeContent("content_base64", "data:application/pdf;base64,"+encoded, 1024)
	assert.Nil(t, err)
	assert.Equal(t, mcpAttachmentTestPdfContent, data)

	data, err = mcpAttachmentDecodeContent("content_base64", " "+encoded[:8]+"\n"+encoded[8:]+"\r\n", 1024)
	assert.Nil(t, err)
	assert.Equal(t, mcpAttachmentTestPdfContent, data)

	data, err = mcpAttachmentDecodeContent("content_base64", base64.RawStdEncoding.EncodeToString([]byte("abcd")), 1024)
	assert.Nil(t, err)
	assert.Equal(t, []byte("abcd"), data)
}

func TestMCPAttachmentDecodeContent_InvalidContent(t *testing.T) {
	_, err := mcpAttachmentDecodeContent("content_base64", "", 1024)
	assert.Equal(t, MCPToolErrorCodeRequired, getMCPAttachmentTestToolError(t, err).Code)

	_, err = mcpAttachmentDecodeContent("content_base64", "   \n", 1024)
	assert.Equal(t, MCPToolErrorCodeRequired, getMCPAttachmentTestToolError(t, err).Code)

	// a client-local file name or path is never treated as an uploaded file
	_, err = mcpAttachmentDecodeContent("content_base64", "/home/user/receipt.pdf", 1024)
	toolErr := getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "content_base64", toolErr.Field)

	_, err = mcpAttachmentDecodeContent("content_base64", "receipt.pdf", 1024)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)

	_, err = mcpAttachmentDecodeContent("content_base64", "data:application/pdf,"+base64.StdEncoding.EncodeToString([]byte("abc")), 1024)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)
}

func TestMCPAttachmentDecodeContent_SizeCap(t *testing.T) {
	// the encoded content is longer than the encoded length of the maximum size, rejected before decoding
	_, err := mcpAttachmentDecodeContent("content_base64", base64.StdEncoding.EncodeToString(make([]byte, 100)), 10)
	toolErr := getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLarge, toolErr.Code)
	assert.Equal(t, "content_base64", toolErr.Field)
	assert.Equal(t, int64(10), *toolErr.Limit)
	assert.True(t, *toolErr.Actual > 10)

	// the encoded length of 5 bytes equals to the encoded length of 4 bytes, rejected after decoding
	_, err = mcpAttachmentDecodeContent("content_base64", base64.StdEncoding.EncodeToString([]byte("12345")), 4)
	toolErr = getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLarge, toolErr.Code)
	assert.Equal(t, int64(4), *toolErr.Limit)
	assert.Equal(t, int64(5), *toolErr.Actual)

	data, err := mcpAttachmentDecodeContent("content_base64", base64.StdEncoding.EncodeToString([]byte("1234")), 4)
	assert.Nil(t, err)
	assert.Equal(t, []byte("1234"), data)

	assert.Equal(t, int64(0), mcpAttachmentGetMaxBase64Length(0))
	assert.Equal(t, int64(8), mcpAttachmentGetMaxBase64Length(4))
	assert.Equal(t, int64(1368), mcpAttachmentGetMaxBase64Length(1024))
}

func TestMCPAttachmentResolveExtension(t *testing.T) {
	extension, err := mcpAttachmentResolveExtension("JPG", "")
	assert.Nil(t, err)
	assert.Equal(t, "jpg", extension)

	extension, err = mcpAttachmentResolveExtension(".pdf", "")
	assert.Nil(t, err)
	assert.Equal(t, "pdf", extension)

	extension, err = mcpAttachmentResolveExtension("", "image/jpeg")
	assert.Nil(t, err)
	assert.Equal(t, "jpg", extension)

	extension, err = mcpAttachmentResolveExtension("", "Application/PDF; charset=binary")
	assert.Nil(t, err)
	assert.Equal(t, "pdf", extension)

	extension, err = mcpAttachmentResolveExtension("jpeg", "image/jpeg")
	assert.Nil(t, err)
	assert.Equal(t, "jpeg", extension)

	_, err = mcpAttachmentResolveExtension("", "")
	assert.Equal(t, MCPToolErrorCodeRequired, getMCPAttachmentTestToolError(t, err).Code)

	_, err = mcpAttachmentResolveExtension("exe", "")
	toolErr := getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "file_extension", toolErr.Field)

	_, err = mcpAttachmentResolveExtension("", "text/plain")
	toolErr = getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "content_type", toolErr.Field)

	_, err = mcpAttachmentResolveExtension("png", "application/pdf")
	toolErr = getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "content_type", toolErr.Field)
}

func TestMCPAttachmentValidateContent(t *testing.T) {
	assert.Nil(t, mcpAttachmentValidateContent("content_base64", "pdf", mcpAttachmentTestPdfContent))
	assert.Nil(t, mcpAttachmentValidateContent("content_base64", "png", mcpAttachmentTestPngContent))
	assert.Nil(t, mcpAttachmentValidateContent("content_base64", "jpg", []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00")))
	assert.Nil(t, mcpAttachmentValidateContent("content_base64", "gif", []byte("GIF89a\x01\x00\x01\x00")))
	assert.Nil(t, mcpAttachmentValidateContent("content_base64", "webp", []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")))

	err := mcpAttachmentValidateContent("content_base64", "pdf", []byte("hello world"))
	toolErr := getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "content_base64", toolErr.Field)

	err = mcpAttachmentValidateContent("content_base64", "pdf", mcpAttachmentTestPngContent)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)

	err = mcpAttachmentValidateContent("content_base64", "jpg", mcpAttachmentTestPngContent)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)

	err = mcpAttachmentValidateContent("content_base64", "png", mcpAttachmentTestPdfContent)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)

	err = mcpAttachmentValidateContent("content_base64", "exe", []byte("MZ"))
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)
}

func TestMCPAttachmentComputeChecksum(t *testing.T) {
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", mcpAttachmentComputeChecksum([]byte("abc")))
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", mcpAttachmentComputeChecksum([]byte{}))
}

func TestMCPAttachmentBuildInfo(t *testing.T) {
	pictureInfo := &models.TransactionPictureInfo{
		Uid:              1,
		PictureId:        3001,
		TransactionId:    models.TransactionPictureNewPictureTransactionId,
		PictureExtension: "pdf",
		CreatedUnixTime:  1700000000,
	}

	info := mcpAttachmentBuildInfo(pictureInfo, []byte("abc"), true)
	assert.Equal(t, "3001", info.Id)
	assert.False(t, info.Attached)
	assert.Equal(t, "", info.TransactionId)
	assert.Equal(t, "application/pdf", info.ContentType)
	assert.Equal(t, int64(3), info.SizeBytes)
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", info.SHA256)
	assert.True(t, info.ContentAvailable)
	assert.Equal(t, "2023-11-14T22:13:20Z", info.CreatedTime)
	assert.Equal(t, "ezbookkeeping://transaction-attachments/3001.pdf", mcpAttachmentGetResourceUri(pictureInfo))

	pictureInfo.TransactionId = 4001
	pictureInfo.PictureExtension = "png"
	info = mcpAttachmentBuildInfo(pictureInfo, nil, false)
	assert.True(t, info.Attached)
	assert.Equal(t, "4001", info.TransactionId)
	assert.Equal(t, "image/png", info.ContentType)
	assert.Equal(t, int64(0), info.SizeBytes)
	assert.Equal(t, "", info.SHA256)
	assert.False(t, info.ContentAvailable)
}

func TestMCPAttachmentCheckEnabled(t *testing.T) {
	err := mcpAttachmentCheckEnabled(nil)
	assert.Equal(t, MCPToolErrorCodeNotSupported, getMCPAttachmentTestToolError(t, err).Code)

	err = mcpAttachmentCheckEnabled(&settings.Config{EnableTransactionPictures: false})
	assert.Equal(t, MCPToolErrorCodeNotSupported, getMCPAttachmentTestToolError(t, err).Code)

	assert.Nil(t, mcpAttachmentCheckEnabled(&settings.Config{EnableTransactionPictures: true}))
}

func TestMCPAttachmentCopyTransactionForModify(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:        1,
		Uid:                  2,
		Type:                 models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		CategoryId:           3,
		TransactionTime:      4,
		TimezoneUtcOffset:    60,
		AccountId:            5,
		Amount:               12000,
		RelatedId:            6,
		RelatedAccountId:     7,
		RelatedAccountAmount: 12000,
		Comment:              "comment",
		GeoLatitude:          1.5,
		GeoLongitude:         2.5,
	}

	newTransaction := mcpAttachmentCopyTransactionForModify(transaction)
	assert.Equal(t, transaction.TransactionId, newTransaction.TransactionId)
	assert.Equal(t, transaction.Type, newTransaction.Type)
	assert.Equal(t, transaction.CategoryId, newTransaction.CategoryId)
	assert.Equal(t, transaction.TransactionTime, newTransaction.TransactionTime)
	assert.Equal(t, transaction.TimezoneUtcOffset, newTransaction.TimezoneUtcOffset)
	assert.Equal(t, transaction.Amount, newTransaction.Amount)
	assert.Equal(t, transaction.RelatedAccountId, newTransaction.RelatedAccountId)
	assert.Equal(t, transaction.RelatedAccountAmount, newTransaction.RelatedAccountAmount)
	assert.Equal(t, transaction.Comment, newTransaction.Comment)
	assert.Equal(t, transaction.GeoLatitude, newTransaction.GeoLatitude)

	transaction.Type = models.TRANSACTION_DB_TYPE_EXPENSE
	newTransaction = mcpAttachmentCopyTransactionForModify(transaction)
	assert.Equal(t, int64(0), newTransaction.RelatedAccountId)
	assert.Equal(t, int64(0), newTransaction.RelatedAccountAmount)
}

func TestUploadTransactionAttachmentToolHandler_ValidationBeforeSaving(t *testing.T) {
	newRequest := func(arguments map[string]any) *MCPCallToolRequest {
		data, err := json.Marshal(arguments)
		assert.Nil(t, err)
		return &MCPCallToolRequest{
			Name:      "upload_transaction_attachment",
			Arguments: data,
		}
	}

	enabledConfig := &settings.Config{
		EnableTransactionPictures:     true,
		MaxTransactionPictureFileSize: 16,
	}

	// attachments are disabled
	_, _, err := MCPUploadTransactionAttachmentToolHandler.Handle(nil, newRequest(map[string]any{
		"content_base64": base64.StdEncoding.EncodeToString(mcpAttachmentTestPdfContent),
		"file_extension": "pdf",
	}), nil, &settings.Config{}, nil)
	assert.Equal(t, MCPToolErrorCodeNotSupported, getMCPAttachmentTestToolError(t, err).Code)

	// file type is required
	_, _, err = MCPUploadTransactionAttachmentToolHandler.Handle(nil, newRequest(map[string]any{
		"content_base64": base64.StdEncoding.EncodeToString([]byte("%PDF-1.4")),
	}), nil, enabledConfig, nil)
	assert.Equal(t, MCPToolErrorCodeRequired, getMCPAttachmentTestToolError(t, err).Code)

	// file content exceeds the maximum size
	_, _, err = MCPUploadTransactionAttachmentToolHandler.Handle(nil, newRequest(map[string]any{
		"content_base64": base64.StdEncoding.EncodeToString(mcpAttachmentTestPdfContent),
		"file_extension": "pdf",
	}), nil, enabledConfig, nil)
	toolErr := getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLarge, toolErr.Code)
	assert.Equal(t, int64(16), *toolErr.Limit)

	// file content does not match the file type
	_, _, err = MCPUploadTransactionAttachmentToolHandler.Handle(nil, newRequest(map[string]any{
		"content_base64": base64.StdEncoding.EncodeToString([]byte("not a pdf")),
		"content_type":   "application/pdf",
	}), nil, enabledConfig, nil)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, getMCPAttachmentTestToolError(t, err).Code)

	// a file name is not a file content
	_, _, err = MCPUploadTransactionAttachmentToolHandler.Handle(nil, newRequest(map[string]any{
		"content_base64": "receipt.pdf",
		"file_extension": "pdf",
	}), nil, enabledConfig, nil)
	toolErr = getMCPAttachmentTestToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.True(t, strings.Contains(toolErr.Message, "base64"))
}

func TestAttachmentToolHandlers_Annotations(t *testing.T) {
	initializeMCPContainerTest(t)
	tools := make(map[string]*MCPTool)

	for _, tool := range Container.GetMCPTools() {
		tools[tool.Name] = tool
	}

	readOnlyTools := []string{"list_transaction_attachments", "get_transaction_attachment", "download_transaction_attachment", "query_account_balances_as_of", "query_transaction_totals", "get_account_reconciliation_statement"}

	for _, name := range readOnlyTools {
		tool, exists := tools[name]

		if !assert.True(t, exists, name) {
			continue
		}

		assert.True(t, tool.Annotations.ReadOnlyHint, name)
		assert.False(t, tool.Annotations.DestructiveHint, name)
		assert.True(t, tool.Annotations.IdempotentHint, name)
	}

	upload := tools["upload_transaction_attachment"]
	assert.False(t, upload.Annotations.ReadOnlyHint)
	assert.False(t, upload.Annotations.DestructiveHint)
	assert.False(t, upload.Annotations.IdempotentHint)

	for _, name := range []string{"detach_transaction_attachment", "delete_transaction_attachment"} {
		assert.False(t, tools[name].Annotations.ReadOnlyHint, name)
		assert.True(t, tools[name].Annotations.DestructiveHint, name)
		assert.True(t, tools[name].Annotations.IdempotentHint, name)
	}
}
