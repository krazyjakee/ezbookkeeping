package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

func TestMCPReferenceParse(t *testing.T) {
	references, err := mcpReferenceParse("references", []*MCPTransactionReference{
		{Type: "supplier", Value: " ACME Ltd "},
		{Type: "invoice_number", Value: "INV-001"},
		{Type: "supplier", Value: "ACME Ltd"},
	})

	assert.Nil(t, err)
	assert.Equal(t, 2, len(references))
	assert.Equal(t, models.TRANSACTION_REFERENCE_TYPE_SUPPLIER, references[0].ReferenceType)
	assert.Equal(t, "ACME Ltd", references[0].ReferenceValue)
	assert.Equal(t, models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER, references[1].ReferenceType)

	references, err = mcpReferenceParse("references", nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(references))
}

func TestMCPReferenceParse_InvalidReferences(t *testing.T) {
	_, err := mcpReferenceParse("references", []*MCPTransactionReference{{Type: "vendor", Value: "ACME"}})
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeInvalidValue, toolErr.Code)
	assert.Equal(t, "references[0].type", toolErr.Field)

	_, err = mcpReferenceParse("references", []*MCPTransactionReference{{Type: "supplier", Value: "  "}})
	toolErr = mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeRequired, toolErr.Code)
	assert.Equal(t, "references[0].value", toolErr.Field)

	_, err = mcpReferenceParse("references", []*MCPTransactionReference{{Type: "supplier", Value: "ok"}, {Type: "supplier", Value: strings.Repeat("中", models.TransactionReferenceValueMaxLength+1)}})
	toolErr = mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooLong, toolErr.Code)
	assert.Equal(t, "references[1].value", toolErr.Field)
	assert.Equal(t, int64(models.TransactionReferenceValueMaxLength), *toolErr.Limit)

	tooManyReferences := make([]*MCPTransactionReference, models.MaximumReferencesCountOfTransaction+1)

	for i := 0; i < len(tooManyReferences); i++ {
		tooManyReferences[i] = &MCPTransactionReference{Type: "source_row", Value: utils.IntToString(i)}
	}

	_, err = mcpReferenceParse("references", tooManyReferences)
	assert.Equal(t, MCPToolErrorCodeTooMany, mcpTestGetToolError(t, err).Code)

	_, err = mcpReferenceParse("references", []*MCPTransactionReference{nil})
	assert.Equal(t, MCPToolErrorCodeRequired, mcpTestGetToolError(t, err).Code)
}

func TestMCPReferenceListEquals(t *testing.T) {
	references1 := []*models.TransactionReference{
		{ReferenceType: "supplier", ReferenceValue: "ACME"},
		{ReferenceType: "invoice_number", ReferenceValue: "INV-1"},
	}
	references2 := []*models.TransactionReference{
		{ReferenceId: 10, TransactionId: 100, ReferenceType: "supplier", ReferenceValue: "ACME"},
		{ReferenceId: 11, TransactionId: 100, ReferenceType: "invoice_number", ReferenceValue: "INV-1"},
	}

	assert.True(t, mcpReferenceListEquals(references1, references2))
	assert.True(t, mcpReferenceListEquals(nil, []*models.TransactionReference{}))
	assert.False(t, mcpReferenceListEquals(references1, references2[:1]))
	assert.False(t, mcpReferenceListEquals(references1, []*models.TransactionReference{references1[1], references1[0]}))
}

func TestMCPTxnBuildDetail_References(t *testing.T) {
	transaction := &models.Transaction{
		TransactionId:   100,
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(1704164645),
		AccountId:       1,
		Amount:          12000,
	}

	data := &mcpTxnDetailData{
		references: map[int64][]*models.TransactionReference{
			100: {
				{ReferenceType: "supplier", ReferenceValue: "ACME"},
				{ReferenceType: "receipt_number", ReferenceValue: "R-1"},
			},
		},
		attachments: map[int64][]*models.TransactionPictureInfo{
			100: {{PictureId: 9, TransactionId: 100, PictureExtension: "pdf", OriginalFileName: "receipt.pdf", FileSize: 3, FileSha256: "abc"}},
		},
	}

	detail := mcpTxnBuildDetail(transaction, data, nil, true)
	assert.Equal(t, 2, len(detail.References))
	assert.Equal(t, "supplier", detail.References[0].Type)
	assert.Equal(t, "ACME", detail.References[0].Value)
	assert.Equal(t, "receipt_number", detail.References[1].Type)
	assert.Equal(t, 1, len(detail.Attachments))
	assert.Equal(t, "receipt.pdf", detail.Attachments[0].OriginalFileName)
	assert.Equal(t, int64(3), detail.Attachments[0].SizeBytes)
	assert.Equal(t, "abc", detail.Attachments[0].SHA256)

	transaction.TransactionId = 101
	detail = mcpTxnBuildDetail(transaction, data, nil, true)
	assert.Nil(t, detail.References)
}

func TestMCPTxnFilterTransactions(t *testing.T) {
	transactions := []*models.Transaction{
		{TransactionId: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, AccountId: 10, CategoryId: 100, TransactionTime: 1000},
		{TransactionId: 2, Type: models.TRANSACTION_DB_TYPE_INCOME, AccountId: 10, CategoryId: 200, TransactionTime: 3000},
		{TransactionId: 3, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, AccountId: 20, RelatedAccountId: 10, CategoryId: 300, TransactionTime: 2000},
		{TransactionId: 4, Type: models.TRANSACTION_DB_TYPE_TRANSFER_IN, AccountId: 10, RelatedAccountId: 20, CategoryId: 300, TransactionTime: 2000},
		{TransactionId: 5, Type: models.TRANSACTION_DB_TYPE_EXPENSE, AccountId: 30, CategoryId: 100, TransactionTime: 5000},
	}

	result := mcpTxnFilterTransactions(transactions, 0, 0, 0, nil, nil)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, int64(5), result[0].TransactionId)
	assert.Equal(t, int64(2), result[1].TransactionId)
	assert.Equal(t, int64(3), result[2].TransactionId)
	assert.Equal(t, int64(1), result[3].TransactionId)

	result = mcpTxnFilterTransactions(transactions, 4000, 1500, 0, nil, nil)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, int64(2), result[0].TransactionId)
	assert.Equal(t, int64(3), result[1].TransactionId)

	result = mcpTxnFilterTransactions(transactions, 0, 0, models.TRANSACTION_TYPE_EXPENSE, nil, nil)
	assert.Equal(t, 2, len(result))

	result = mcpTxnFilterTransactions(transactions, 0, 0, 0, []int64{10}, nil)
	assert.Equal(t, 3, len(result))

	result = mcpTxnFilterTransactions(transactions, 0, 0, 0, nil, []int64{100})
	assert.Equal(t, 2, len(result))

	result = mcpTxnFilterTransactions(transactions, 0, 0, models.TRANSACTION_TYPE_TRANSFER, []int64{10}, []int64{300})
	assert.Equal(t, 1, len(result))
	assert.Equal(t, int64(3), result[0].TransactionId)
}

func TestMCPAttachmentBuildInfo_RecordedMetadata(t *testing.T) {
	pictureInfo := &models.TransactionPictureInfo{
		Uid:              1,
		PictureId:        3001,
		TransactionId:    4001,
		PictureExtension: "pdf",
		OriginalFileName: "receipt.pdf",
		FileSize:         3,
		FileSha256:       "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
	}

	info := mcpAttachmentBuildInfo(pictureInfo, []byte("abc"), true)
	assert.Equal(t, "receipt.pdf", info.OriginalFileName)
	assert.Equal(t, mcpAttachmentChecksumSourceStoredFile, info.ChecksumSource)
	assert.Equal(t, pictureInfo.FileSha256, info.RecordedSHA256)
	assert.Equal(t, int64(3), info.RecordedSizeBytes)
	assert.True(t, *info.IntegrityVerified)

	info = mcpAttachmentBuildInfo(pictureInfo, []byte("abd"), true)
	assert.False(t, *info.IntegrityVerified)

	info = mcpAttachmentBuildInfo(pictureInfo, nil, false)
	assert.Equal(t, mcpAttachmentChecksumSourceUploadMetadata, info.ChecksumSource)
	assert.Equal(t, pictureInfo.FileSha256, info.SHA256)
	assert.Equal(t, int64(3), info.SizeBytes)
	assert.Nil(t, info.IntegrityVerified)

	pictureInfo.FileSha256 = ""
	pictureInfo.FileSize = 0
	info = mcpAttachmentBuildInfo(pictureInfo, []byte("abc"), true)
	assert.Equal(t, mcpAttachmentChecksumSourceStoredFile, info.ChecksumSource)
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", info.SHA256)
	assert.Nil(t, info.IntegrityVerified)
}
