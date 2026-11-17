package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const mcpImportTestChecksum = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

func newMCPImportTestFingerprint(row int, references []*models.TransactionReference, checksums []string) *mcpImportRowFingerprint {
	return mcpImportNewRowFingerprint(row, models.TRANSACTION_DB_TYPE_EXPENSE, 1, 0, 12000, 1704164645, references, checksums)
}

func newMCPImportTestTransaction(transactionId int64, unixTime int64) *models.Transaction {
	return &models.Transaction{
		TransactionId:   transactionId,
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		AccountId:       1,
		Amount:          12000,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(unixTime),
		Comment:         "Software",
	}
}

func TestMCPImportFindDuplicateCandidates_NoDuplicates(t *testing.T) {
	fingerprint := newMCPImportTestFingerprint(1, nil, nil)
	index := mcpImportBuildExistingIndex(nil, nil, nil)
	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)
	assert.Equal(t, 0, len(candidates))
}

func TestMCPImportFindDuplicateCandidates_SameIdentifierReference(t *testing.T) {
	references := []*models.TransactionReference{
		{TransactionId: 100, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID, ReferenceValue: "bank-1"},
		{TransactionId: 101, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SUPPLIER, ReferenceValue: "ACME"},
		{TransactionId: 102, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER, ReferenceValue: "bank-1"},
	}
	index := mcpImportBuildExistingIndex(references, nil, []*models.Transaction{
		newMCPImportTestTransaction(100, 1600000000),
		newMCPImportTestTransaction(101, 1600000000),
		newMCPImportTestTransaction(102, 1600000000),
	})

	fingerprint := newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID, ReferenceValue: "bank-1"},
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SUPPLIER, ReferenceValue: "ACME"},
	}, nil)

	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, "100", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonReference}, candidates[0].Reasons)
	assert.Equal(t, "120.00", candidates[0].Amount)
	assert.Equal(t, "Software", candidates[0].Comment)
}

func TestMCPImportFindDuplicateCandidates_DeletedTransactionIgnored(t *testing.T) {
	references := []*models.TransactionReference{
		{TransactionId: 100, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER, ReferenceValue: "R-1"},
	}
	index := mcpImportBuildExistingIndex(references, nil, nil)

	fingerprint := newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER, ReferenceValue: "R-1"},
	}, nil)

	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)
	assert.Equal(t, 0, len(candidates))
}

func TestMCPImportFindDuplicateCandidates_SameSourceDocumentAndRow(t *testing.T) {
	references := []*models.TransactionReference{
		{TransactionId: 100, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT, ReferenceValue: "statement.csv"},
		{TransactionId: 100, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW, ReferenceValue: "7"},
		{TransactionId: 101, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT, ReferenceValue: "statement.csv"},
		{TransactionId: 101, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW, ReferenceValue: "8"},
		{TransactionId: 102, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT, ReferenceValue: "other.csv"},
		{TransactionId: 102, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW, ReferenceValue: "7"},
	}
	index := mcpImportBuildExistingIndex(references, nil, []*models.Transaction{
		newMCPImportTestTransaction(100, 1600000000),
		newMCPImportTestTransaction(101, 1600000000),
		newMCPImportTestTransaction(102, 1600000000),
	})

	fingerprint := newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT, ReferenceValue: "statement.csv"},
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW, ReferenceValue: "7"},
	}, nil)

	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, "100", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonSourceDocumentRow}, candidates[0].Reasons)

	// only the source row without the source document does not identify a record
	fingerprint = newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW, ReferenceValue: "7"},
	}, nil)
	assert.Equal(t, 0, len(mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)))
}

func TestMCPImportFindDuplicateCandidates_SameAttachmentChecksum(t *testing.T) {
	pictureInfos := []*models.TransactionPictureInfo{
		{PictureId: 1, TransactionId: 100, FileSha256: mcpImportTestChecksum},
		{PictureId: 2, TransactionId: models.TransactionPictureNewPictureTransactionId, FileSha256: mcpImportTestChecksum},
	}
	index := mcpImportBuildExistingIndex(nil, pictureInfos, []*models.Transaction{newMCPImportTestTransaction(100, 1600000000)})

	fingerprint := newMCPImportTestFingerprint(1, nil, []string{mcpImportTestChecksum})
	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, nil)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, "100", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonAttachmentChecksum}, candidates[0].Reasons)
}

func TestMCPImportFindDuplicateCandidates_SameAmountDateAccount(t *testing.T) {
	fingerprint := newMCPImportTestFingerprint(1, nil, nil)
	matches := []*models.Transaction{
		newMCPImportTestTransaction(100, 1704164645+23*60*60),
		newMCPImportTestTransaction(101, 1704164645-25*60*60),
		newMCPImportTestTransaction(102, 1704164645),
		newMCPImportTestTransaction(103, 1704164645),
	}
	matches[2].Amount = 12001
	matches[3].Type = models.TRANSACTION_DB_TYPE_INCOME

	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, mcpImportBuildExistingIndex(nil, nil, nil), matches)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, "100", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonAmountDateAccount}, candidates[0].Reasons)
}

func TestMCPImportFindDuplicateCandidates_MultipleReasonsSortedFirst(t *testing.T) {
	references := []*models.TransactionReference{
		{TransactionId: 100, ReferenceType: models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER, ReferenceValue: "INV-1"},
	}
	index := mcpImportBuildExistingIndex(references, nil, []*models.Transaction{newMCPImportTestTransaction(100, 1704164645)})
	fingerprint := newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER, ReferenceValue: "INV-1"},
	}, nil)
	matches := []*models.Transaction{
		newMCPImportTestTransaction(99, 1704164645),
		newMCPImportTestTransaction(100, 1704164645),
	}

	candidates := mcpImportFindDuplicateCandidates(fingerprint, nil, index, matches)
	assert.Equal(t, 2, len(candidates))
	assert.Equal(t, "100", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonReference, mcpImportDuplicateReasonAmountDateAccount}, candidates[0].Reasons)
	assert.Equal(t, "99", candidates[1].TransactionId)
}

func TestMCPImportFindDuplicateCandidates_EarlierRowsOfBatch(t *testing.T) {
	earlierRow := newMCPImportTestFingerprint(1, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID, ReferenceValue: "bank-1"},
	}, nil)
	otherRow := mcpImportNewRowFingerprint(2, models.TRANSACTION_DB_TYPE_EXPENSE, 2, 0, 500, 1704164645, nil, nil)

	fingerprint := newMCPImportTestFingerprint(3, []*models.TransactionReference{
		{ReferenceType: models.TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID, ReferenceValue: "bank-1"},
	}, nil)

	candidates := mcpImportFindDuplicateCandidates(fingerprint, []*mcpImportRowFingerprint{earlierRow, otherRow}, nil, nil)
	assert.Equal(t, 1, len(candidates))
	assert.Equal(t, 1, candidates[0].Row)
	assert.Equal(t, "", candidates[0].TransactionId)
	assert.Equal(t, []string{mcpImportDuplicateReasonReference, mcpImportDuplicateReasonAmountDateAccount}, candidates[0].Reasons)
}

func TestMCPImportFindDuplicateCandidates_TransferDestinationMustMatch(t *testing.T) {
	fingerprint := mcpImportNewRowFingerprint(1, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, 1, 2, 12000, 1704164645, nil, nil)
	transaction := newMCPImportTestTransaction(100, 1704164645)
	transaction.Type = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	transaction.RelatedAccountId = 3

	assert.Equal(t, 0, len(mcpImportFindDuplicateCandidates(fingerprint, nil, nil, []*models.Transaction{transaction})))

	transaction.RelatedAccountId = 2
	assert.Equal(t, 1, len(mcpImportFindDuplicateCandidates(fingerprint, nil, nil, []*models.Transaction{transaction})))
}

func TestMCPImportValidateRequest(t *testing.T) {
	_, err := mcpImportValidateRequest(&MCPImportTransactionsRequest{})
	assert.Equal(t, MCPToolErrorCodeRequired, mcpTestGetToolError(t, err).Code)

	rows := make([]*MCPImportTransactionsRow, mcpImportMaxRows+1)
	_, err = mcpImportValidateRequest(&MCPImportTransactionsRequest{Rows: rows})
	toolErr := mcpTestGetToolError(t, err)
	assert.Equal(t, MCPToolErrorCodeTooMany, toolErr.Code)
	assert.Equal(t, int64(mcpImportMaxRows), *toolErr.Limit)

	rows = []*MCPImportTransactionsRow{{}, {}}
	_, err = mcpImportValidateRequest(&MCPImportTransactionsRequest{Rows: rows, AllowDuplicatesRows: []int{3}})
	assert.Equal(t, "allow_duplicates_rows", mcpTestGetToolError(t, err).Field)

	allowDuplicateRows, err := mcpImportValidateRequest(&MCPImportTransactionsRequest{Rows: rows, AllowDuplicatesRows: []int{2}})
	assert.Nil(t, err)
	assert.True(t, allowDuplicateRows[2])
	assert.False(t, allowDuplicateRows[1])
}

func TestMCPImportPrepareRows(t *testing.T) {
	request := &MCPImportTransactionsRequest{
		SourceDocument: "statement.csv",
		Rows: []*MCPImportTransactionsRow{
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", IdempotencyKey: "statement.csv:1", References: []*MCPTransactionReference{{Type: "source_row", Value: "1"}}},
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", IdempotencyKey: "statement.csv:1"},
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", AttachmentSHA256: []string{"abc"}},
			{Type: "expense", Amount: "1"},
			{TransactionId: "100", Comment: "fixed"},
			{TransactionId: "100", References: []*MCPTransactionReference{{Type: "invoice_number", Value: "INV-1"}}},
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", AttachmentSHA256: []string{strings.ToUpper(mcpImportTestChecksum)}, References: []*MCPTransactionReference{{Type: "unknown", Value: "1"}}},
			nil,
		},
	}

	preparedRows := mcpImportPrepareRows(request)
	assert.Equal(t, 8, len(preparedRows))

	assert.Nil(t, preparedRows[0].result.Error)
	assert.Equal(t, "statement.csv:1", preparedRows[0].idempotencyKey)
	assert.Equal(t, 2, len(preparedRows[0].references))
	assert.Equal(t, "statement.csv", mcpReferenceFindValue(preparedRows[0].references, models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT))

	assert.Equal(t, mcpIdempotencyKeyArgumentName, preparedRows[1].result.Error.Field)
	assert.Equal(t, "attachment_sha256", preparedRows[2].result.Error.Field)
	assert.Equal(t, "time", preparedRows[3].result.Error.Field)

	assert.Nil(t, preparedRows[4].result.Error)
	assert.Equal(t, mcpImportRowActionUpdate, preparedRows[4].result.Action)
	assert.Equal(t, 0, len(preparedRows[4].references))

	assert.Nil(t, preparedRows[5].result.Error)
	assert.Equal(t, 2, len(preparedRows[5].references))

	assert.Equal(t, "references[0].type", preparedRows[6].result.Error.Field)
	assert.Equal(t, MCPToolErrorCodeRequired, preparedRows[7].result.Error.Code)
}

func TestMCPImportPrepareRows_DuplicatedAttachmentIds(t *testing.T) {
	request := &MCPImportTransactionsRequest{
		Rows: []*MCPImportTransactionsRow{
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", AttachmentIds: []string{"5"}},
			{Type: "expense", Time: "2024-01-02T03:04:05Z", Amount: "1", AttachmentIds: []string{"5"}},
		},
	}

	preparedRows := mcpImportPrepareRows(request)
	assert.Nil(t, preparedRows[0].result.Error)
	assert.Equal(t, []int64{5}, preparedRows[0].attachmentIds)
	assert.Equal(t, "attachment_ids", preparedRows[1].result.Error.Field)
}

func TestMCPImportParseChecksums(t *testing.T) {
	checksums, err := mcpImportParseChecksums("attachment_sha256", []string{strings.ToUpper(mcpImportTestChecksum), " " + mcpImportTestChecksum + " "})
	assert.Nil(t, err)
	assert.Equal(t, []string{mcpImportTestChecksum}, checksums)

	_, err = mcpImportParseChecksums("attachment_sha256", []string{mcpImportTestChecksum[:62]})
	assert.Equal(t, "attachment_sha256", mcpTestGetToolError(t, err).Field)
}

func TestMCPImportBuildUpdateRequest_OnlyNonEmptyFields(t *testing.T) {
	row := &MCPImportTransactionsRow{
		TransactionId:         "100",
		SecondaryCategoryName: "Software",
		Comment:               "fixed",
	}

	request := mcpImportBuildUpdateRequest(row, nil)
	assert.Equal(t, "100", request.TransactionId)
	assert.Equal(t, "Software", *request.CategoryName)
	assert.Equal(t, "fixed", *request.Comment)
	assert.Nil(t, request.Amount)
	assert.Nil(t, request.Time)
	assert.Nil(t, request.Type)
	assert.Nil(t, request.References)
}

func TestMCPImportSummarizeResponse(t *testing.T) {
	response := &MCPImportTransactionsResponse{
		DryRun: false,
		Results: []*MCPImportTransactionsRowResult{
			{Row: 1, Status: MCPWriteStatusCreated, TransactionId: "1"},
			{Row: 2, Status: MCPWriteStatusCreated, TransactionId: "2", IdempotentReplay: true},
			{Row: 3, Status: MCPWriteStatusSkipped, PossibleDuplicate: true},
			{Row: 4, Status: MCPWriteStatusFailed},
			{Row: 5, Status: MCPWriteStatusUpdated, TransactionId: "5"},
		},
	}

	mcpImportSummarizeResponse(response)
	assert.False(t, response.Success)
	assert.True(t, response.Committed)
	assert.Equal(t, 2, response.CreatedCount)
	assert.Equal(t, 1, response.UpdatedCount)
	assert.Equal(t, 1, response.SkippedCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Equal(t, 1, response.ReplayedCount)
	assert.Equal(t, 1, response.PossibleDuplicateCount)
	assert.NotEqual(t, "", response.NextStep)

	response = &MCPImportTransactionsResponse{
		DryRun: false,
		Results: []*MCPImportTransactionsRowResult{
			{Row: 1, Status: MCPWriteStatusCreated, TransactionId: "1", IdempotentReplay: true},
		},
	}

	mcpImportSummarizeResponse(response)
	assert.True(t, response.Success)
	assert.False(t, response.Committed)
}
