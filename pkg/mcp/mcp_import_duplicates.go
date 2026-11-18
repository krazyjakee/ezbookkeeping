package mcp

import (
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Reasons of possible duplicates found when importing transactions
const (
	mcpImportDuplicateReasonReference          = "same_reference"
	mcpImportDuplicateReasonSourceDocumentRow  = "same_source_document_and_row"
	mcpImportDuplicateReasonAttachmentChecksum = "same_attachment_checksum"
	mcpImportDuplicateReasonAmountDateAccount  = "same_amount_date_account"
)

// mcpImportDuplicateTimeWindowSeconds is the maximum difference of transaction time (in seconds) of transactions with the same amount and account to be considered as possible duplicates
const mcpImportDuplicateTimeWindowSeconds = int64(24 * 60 * 60)

// mcpImportMaxDuplicateCandidates is the maximum count of duplicate candidates reported for one row
const mcpImportMaxDuplicateCandidates = 10

// MCPImportDuplicateCandidate represents an existing transaction or an earlier row of the same batch which may be a duplicate of a row
type MCPImportDuplicateCandidate struct {
	TransactionId string   `json:"transaction_id,omitempty" jsonschema_description:"ID of the existing transaction (empty if the candidate is an earlier row of this batch)"`
	Row           int      `json:"row,omitempty" jsonschema_description:"Row number (1-based) of the earlier row of this batch (empty if the candidate is an existing transaction)"`
	Reasons       []string `json:"reasons" jsonschema_description:"Why it is a possible duplicate: same_reference (same external_source_id / invoice_number / receipt_number), same_source_document_and_row, same_attachment_checksum, same_amount_date_account (same type, account and amount within one day)"`
	Time          string   `json:"time,omitempty" jsonschema_description:"Transaction time of the existing transaction in RFC 3339 format"`
	Amount        string   `json:"amount,omitempty" jsonschema_description:"Amount of the existing transaction"`
	AccountId     string   `json:"account_id,omitempty" jsonschema_description:"Account ID of the existing transaction"`
	Comment       string   `json:"comment,omitempty" jsonschema_description:"Description of the existing transaction"`
}

// mcpImportRowFingerprint contains the values of an import row which are used to find possible duplicates
type mcpImportRowFingerprint struct {
	row                  int
	transactionType      models.TransactionDbType
	accountId            int64
	destinationAccountId int64
	amount               int64
	unixTime             int64
	identifierReferences []string
	sourceDocument       string
	sourceRow            string
	checksums            []string
}

// mcpImportExistingIndex contains the existing data which are used to find possible duplicates of import rows
type mcpImportExistingIndex struct {
	referenceTransactionIds         map[string][]int64
	sourceDocumentRowTransactionIds map[string][]int64
	checksumTransactionIds          map[string][]int64
	transactions                    map[int64]*models.Transaction
}

// mcpImportReferenceKey returns the key of the reference type and value used in duplicate indexes
func mcpImportReferenceKey(referenceType string, referenceValue string) string {
	return referenceType + "\n" + referenceValue
}

// mcpImportNewRowFingerprint returns the fingerprint of an import row according to its parsed values
func mcpImportNewRowFingerprint(row int, transactionType models.TransactionDbType, accountId int64, destinationAccountId int64, amount int64, unixTime int64, references []*models.TransactionReference, checksums []string) *mcpImportRowFingerprint {
	fingerprint := &mcpImportRowFingerprint{
		row:                  row,
		transactionType:      transactionType,
		accountId:            accountId,
		destinationAccountId: destinationAccountId,
		amount:               amount,
		unixTime:             unixTime,
		identifierReferences: make([]string, 0),
		checksums:            mcpImportUniqueStrings(checksums),
	}

	for i := 0; i < len(references); i++ {
		reference := references[i]

		if models.IsTransactionReferenceTypeIdentifier(reference.ReferenceType) {
			fingerprint.identifierReferences = append(fingerprint.identifierReferences, mcpImportReferenceKey(reference.ReferenceType, reference.ReferenceValue))
		} else if reference.ReferenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT && fingerprint.sourceDocument == "" {
			fingerprint.sourceDocument = reference.ReferenceValue
		} else if reference.ReferenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW && fingerprint.sourceRow == "" {
			fingerprint.sourceRow = reference.ReferenceValue
		}
	}

	return fingerprint
}

// mcpImportBuildExistingIndex returns the duplicate index of existing references, attachment checksums and transactions
func mcpImportBuildExistingIndex(references []*models.TransactionReference, pictureInfos []*models.TransactionPictureInfo, transactions []*models.Transaction) *mcpImportExistingIndex {
	index := &mcpImportExistingIndex{
		referenceTransactionIds:         make(map[string][]int64),
		sourceDocumentRowTransactionIds: make(map[string][]int64),
		checksumTransactionIds:          make(map[string][]int64),
		transactions:                    make(map[int64]*models.Transaction, len(transactions)),
	}

	for i := 0; i < len(transactions); i++ {
		index.transactions[transactions[i].TransactionId] = transactions[i]
	}

	sourceDocuments := make(map[int64][]string)
	sourceRows := make(map[int64][]string)

	for i := 0; i < len(references); i++ {
		reference := references[i]

		if models.IsTransactionReferenceTypeIdentifier(reference.ReferenceType) {
			key := mcpImportReferenceKey(reference.ReferenceType, reference.ReferenceValue)
			index.referenceTransactionIds[key] = append(index.referenceTransactionIds[key], reference.TransactionId)
		} else if reference.ReferenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT {
			sourceDocuments[reference.TransactionId] = append(sourceDocuments[reference.TransactionId], reference.ReferenceValue)
		} else if reference.ReferenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW {
			sourceRows[reference.TransactionId] = append(sourceRows[reference.TransactionId], reference.ReferenceValue)
		}
	}

	for transactionId, documents := range sourceDocuments {
		for _, document := range documents {
			for _, row := range sourceRows[transactionId] {
				key := mcpImportReferenceKey(document, row)
				index.sourceDocumentRowTransactionIds[key] = append(index.sourceDocumentRowTransactionIds[key], transactionId)
			}
		}
	}

	for i := 0; i < len(pictureInfos); i++ {
		pictureInfo := pictureInfos[i]

		if pictureInfo.FileSha256 == "" || pictureInfo.TransactionId == models.TransactionPictureNewPictureTransactionId {
			continue
		}

		index.checksumTransactionIds[pictureInfo.FileSha256] = append(index.checksumTransactionIds[pictureInfo.FileSha256], pictureInfo.TransactionId)
	}

	return index
}

// mcpImportFindDuplicateCandidates returns the possible duplicates of the import row: existing transactions with the same identifier reference,
// the same source document and row, the same attachment checksum, or the same type, account and amount within one day (amountDateMatches),
// and earlier rows of the same batch matching by the same rules
func mcpImportFindDuplicateCandidates(fingerprint *mcpImportRowFingerprint, earlierRows []*mcpImportRowFingerprint, index *mcpImportExistingIndex, amountDateMatches []*models.Transaction) []*MCPImportDuplicateCandidate {
	transactionReasons := make(map[int64][]string)
	transactionOrder := make([]int64, 0)

	addTransactionReason := func(transactionId int64, reason string) {
		if transactionId <= 0 {
			return
		}

		reasons, exists := transactionReasons[transactionId]

		if !exists {
			transactionOrder = append(transactionOrder, transactionId)
		}

		for i := 0; i < len(reasons); i++ {
			if reasons[i] == reason {
				return
			}
		}

		transactionReasons[transactionId] = append(reasons, reason)
	}

	if index != nil {
		for _, key := range fingerprint.identifierReferences {
			for _, transactionId := range index.referenceTransactionIds[key] {
				addTransactionReason(transactionId, mcpImportDuplicateReasonReference)
			}
		}

		if fingerprint.sourceDocument != "" && fingerprint.sourceRow != "" {
			for _, transactionId := range index.sourceDocumentRowTransactionIds[mcpImportReferenceKey(fingerprint.sourceDocument, fingerprint.sourceRow)] {
				addTransactionReason(transactionId, mcpImportDuplicateReasonSourceDocumentRow)
			}
		}

		for _, checksum := range fingerprint.checksums {
			for _, transactionId := range index.checksumTransactionIds[checksum] {
				addTransactionReason(transactionId, mcpImportDuplicateReasonAttachmentChecksum)
			}
		}
	}

	for i := 0; i < len(amountDateMatches); i++ {
		if mcpImportIsSameAmountDateAccount(fingerprint, amountDateMatches[i]) {
			addTransactionReason(amountDateMatches[i].TransactionId, mcpImportDuplicateReasonAmountDateAccount)
		}
	}

	candidates := make([]*MCPImportDuplicateCandidate, 0)

	for _, transactionId := range transactionOrder {
		var transaction *models.Transaction

		for i := 0; i < len(amountDateMatches); i++ {
			if amountDateMatches[i].TransactionId == transactionId {
				transaction = amountDateMatches[i]
				break
			}
		}

		if transaction == nil && index != nil {
			transaction = index.transactions[transactionId]
		}

		if transaction == nil || transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
			// the transaction has been deleted (references and attachments of deleted transactions are kept), or it is the incoming side of a transfer which is reported by the outgoing side
			continue
		}

		candidates = append(candidates, &MCPImportDuplicateCandidate{
			TransactionId: utils.Int64ToString(transactionId),
			Reasons:       transactionReasons[transactionId],
			Time:          formatMCPTransactionTime(transaction.TransactionTime, transaction.TimezoneUtcOffset),
			Amount:        utils.FormatAmount(transaction.Amount),
			AccountId:     utils.Int64ToString(transaction.AccountId),
			Comment:       transaction.Comment,
		})
	}

	for i := 0; i < len(earlierRows); i++ {
		reasons := mcpImportCompareRows(fingerprint, earlierRows[i])

		if len(reasons) > 0 {
			candidates = append(candidates, &MCPImportDuplicateCandidate{
				Row:     earlierRows[i].row,
				Reasons: reasons,
			})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return len(candidates[i].Reasons) > len(candidates[j].Reasons)
	})

	if len(candidates) > mcpImportMaxDuplicateCandidates {
		candidates = candidates[:mcpImportMaxDuplicateCandidates]
	}

	return candidates
}

// mcpImportIsSameAmountDateAccount returns whether the existing transaction has the same type, account (and destination account for transfers) and amount as the import row, and its time is within one day
func mcpImportIsSameAmountDateAccount(fingerprint *mcpImportRowFingerprint, transaction *models.Transaction) bool {
	if transaction.Type != fingerprint.transactionType || transaction.AccountId != fingerprint.accountId || transaction.Amount != fingerprint.amount {
		return false
	}

	if fingerprint.transactionType == models.TRANSACTION_DB_TYPE_TRANSFER_OUT && transaction.RelatedAccountId != fingerprint.destinationAccountId {
		return false
	}

	timeDifference := utils.GetUnixTimeFromTransactionTime(transaction.TransactionTime) - fingerprint.unixTime

	return timeDifference <= mcpImportDuplicateTimeWindowSeconds && timeDifference >= -mcpImportDuplicateTimeWindowSeconds
}

// mcpImportCompareRows returns the reasons why the import row may be a duplicate of an earlier row of the same batch
func mcpImportCompareRows(fingerprint *mcpImportRowFingerprint, earlierRow *mcpImportRowFingerprint) []string {
	reasons := make([]string, 0)

	if mcpImportStringSlicesIntersect(fingerprint.identifierReferences, earlierRow.identifierReferences) {
		reasons = append(reasons, mcpImportDuplicateReasonReference)
	}

	if fingerprint.sourceDocument != "" && fingerprint.sourceRow != "" && fingerprint.sourceDocument == earlierRow.sourceDocument && fingerprint.sourceRow == earlierRow.sourceRow {
		reasons = append(reasons, mcpImportDuplicateReasonSourceDocumentRow)
	}

	if mcpImportStringSlicesIntersect(fingerprint.checksums, earlierRow.checksums) {
		reasons = append(reasons, mcpImportDuplicateReasonAttachmentChecksum)
	}

	timeDifference := fingerprint.unixTime - earlierRow.unixTime

	if fingerprint.transactionType == earlierRow.transactionType && fingerprint.accountId == earlierRow.accountId && fingerprint.amount == earlierRow.amount &&
		(fingerprint.transactionType != models.TRANSACTION_DB_TYPE_TRANSFER_OUT || fingerprint.destinationAccountId == earlierRow.destinationAccountId) &&
		timeDifference <= mcpImportDuplicateTimeWindowSeconds && timeDifference >= -mcpImportDuplicateTimeWindowSeconds {
		reasons = append(reasons, mcpImportDuplicateReasonAmountDateAccount)
	}

	return reasons
}

// mcpImportStringSlicesIntersect returns whether both slices have at least one same item
func mcpImportStringSlicesIntersect(items1 []string, items2 []string) bool {
	for i := 0; i < len(items1); i++ {
		for j := 0; j < len(items2); j++ {
			if items1[i] == items2[j] {
				return true
			}
		}
	}

	return false
}

// mcpImportUniqueStrings returns the non-empty unique items of the slice in the original order
func mcpImportUniqueStrings(items []string) []string {
	result := make([]string, 0, len(items))
	exists := make(map[string]bool, len(items))

	for i := 0; i < len(items); i++ {
		if items[i] == "" || exists[items[i]] {
			continue
		}

		exists[items[i]] = true
		result = append(result, items[i])
	}

	return result
}
