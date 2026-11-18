package mcp

import (
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// Settings of importing transactions through MCP
const (
	mcpImportMaxRows                = mcpMaximumBatchItemsCount
	mcpImportRowIdempotencyToolName = "import_transactions/row"
	mcpImportRowActionCreate        = "create"
	mcpImportRowActionUpdate        = "update"
	mcpImportBatchSemantics         = "Each row is validated and committed independently (not atomic): a failed or skipped row does not affect other rows, and every row reports its own status and transaction ID. Rerun the same rows with the same per-row idempotency keys to resume safely after a partial failure or timeout: rows which were already committed return their stored result with idempotent_replay=true and are not created again."
)

// MCPImportTransactionsRow represents one row of the import transactions request, it has the same fields as add_transaction (or update_transaction when transaction_id is set)
type MCPImportTransactionsRow struct {
	TransactionId          string                           `json:"transaction_id,omitempty" jsonschema_description:"ID of an existing transaction to update (optional). If set, the row updates that transaction and only non-empty fields are changed; otherwise a new transaction is created"`
	Type                   string                           `json:"type,omitempty" jsonschema:"enum=income,enum=expense,enum=transfer" jsonschema_description:"Transaction type (income, expense, transfer), required for new transactions"`
	Time                   string                           `json:"time,omitempty" jsonschema:"format=date-time" jsonschema_description:"Transaction time in RFC 3339 format, required for new transactions, the UTC offset is saved as the transaction timezone"`
	CategoryId             string                           `json:"category_id,omitempty" jsonschema_description:"Secondary category ID (takes precedence over category_name)"`
	SecondaryCategoryName  string                           `json:"category_name,omitempty" jsonschema_description:"Secondary category name (must match exactly one visible secondary category of the transaction type, ambiguous names are rejected with candidates)"`
	AccountId              string                           `json:"account_id,omitempty" jsonschema_description:"Account ID (takes precedence over account_name)"`
	AccountName            string                           `json:"account_name,omitempty" jsonschema_description:"Account name (must match exactly one visible account)"`
	Amount                 string                           `json:"amount,omitempty" jsonschema_description:"Transaction amount (e.g. 12.34), required for new transactions"`
	DestinationAccountId   string                           `json:"destination_account_id,omitempty" jsonschema_description:"Destination account ID for transfers"`
	DestinationAccountName string                           `json:"destination_account_name,omitempty" jsonschema_description:"Destination account name for transfers"`
	DestinationAmount      string                           `json:"destination_amount,omitempty" jsonschema_description:"Destination amount for transfers"`
	TagIds                 []string                         `json:"tag_ids,omitempty" jsonschema_description:"Tag IDs"`
	Tags                   []string                         `json:"tags,omitempty" jsonschema_description:"Tag names"`
	Comment                string                           `json:"comment,omitempty" jsonschema_description:"Transaction description"`
	GeoLocation            *MCPTransactionDetailGeoLocation `json:"geo_location,omitempty" jsonschema_description:"Geographic location"`
	AttachmentIds          []string                         `json:"attachment_ids,omitempty" jsonschema_description:"IDs of uploaded but unattached attachments to attach (their recorded SHA-256 checksums are also used to find duplicates)"`
	AttachmentSHA256       []string                         `json:"attachment_sha256,omitempty" jsonschema_description:"Hex encoded SHA-256 checksums of the source documents of this row (optional), used only to find existing transactions with the same attachment as possible duplicates"`
	References             []*MCPTransactionReference       `json:"references,omitempty" jsonschema_description:"Evidence / source references of the row, e.g. external_source_id, invoice_number, receipt_number, supplier, source_document and source_row (the original row / line number)"`
	IdempotencyKey         string                           `json:"idempotency_key,omitempty" jsonschema_description:"Idempotency key of this row (optional but recommended, maximum 128 characters, e.g. <source file>:<row>). Rerunning a committed row with the same key returns its stored result instead of creating it again"`
}

// MCPImportTransactionsRequest represents all parameters of the import transactions request
type MCPImportTransactionsRequest struct {
	Rows                []*MCPImportTransactionsRow `json:"rows" jsonschema_description:"Rows to import (maximum count is reported by limits.import_max_rows of get_server_capabilities)"`
	SourceDocument      string                      `json:"source_document,omitempty" jsonschema_description:"Name of the source document (e.g. bank-statement-2024-01.csv) added as source_document reference to every row without one (optional)"`
	AllowDuplicatesRows []int                       `json:"allow_duplicates_rows,omitempty" jsonschema_description:"Row numbers (1-based) which are imported even if they are flagged as possible duplicates (optional)"`
	DryRun              *bool                       `json:"dry_run,omitempty" jsonschema:"default=true" jsonschema_description:"If true (default), only validate rows and report mappings and possible duplicates without saving anything; set to false to commit"`
}

// MCPImportTransactionsRowMapping represents the resolved accounts, category and tags of a row
type MCPImportTransactionsRowMapping struct {
	AccountId              string                     `json:"account_id,omitempty" jsonschema_description:"Resolved account ID"`
	AccountName            string                     `json:"account_name,omitempty" jsonschema_description:"Resolved account name"`
	DestinationAccountId   string                     `json:"destination_account_id,omitempty" jsonschema_description:"Resolved destination account ID (transfers)"`
	DestinationAccountName string                     `json:"destination_account_name,omitempty" jsonschema_description:"Resolved destination account name (transfers)"`
	CategoryId             string                     `json:"category_id,omitempty" jsonschema_description:"Resolved secondary category ID"`
	CategoryName           string                     `json:"category_name,omitempty" jsonschema_description:"Resolved secondary category name"`
	PrimaryCategoryName    string                     `json:"primary_category_name,omitempty" jsonschema_description:"Primary category name of the resolved category"`
	Tags                   []*MCPTransactionDetailTag `json:"tags,omitempty" jsonschema_description:"Resolved tags"`
}

// MCPImportTransactionsRowResult represents the result of one row
type MCPImportTransactionsRowResult struct {
	Row                 int                              `json:"row" jsonschema_description:"Row number (1-based)"`
	Action              string                           `json:"action" jsonschema:"enum=create,enum=update" jsonschema_description:"Whether the row creates a new transaction or updates an existing one"`
	Status              string                           `json:"status" jsonschema:"enum=would_be_created,enum=would_be_updated,enum=created,enum=updated,enum=unchanged,enum=skipped,enum=failed" jsonschema_description:"Row status"`
	TransactionId       string                           `json:"transaction_id,omitempty" jsonschema_description:"ID of the created or updated transaction"`
	IdempotencyKey      string                           `json:"idempotency_key,omitempty" jsonschema_description:"Idempotency key of the row"`
	IdempotentReplay    bool                             `json:"idempotent_replay,omitzero" jsonschema_description:"True if the row was committed by an earlier call with the same idempotency key, and this is its stored result"`
	PossibleDuplicate   bool                             `json:"possible_duplicate,omitzero" jsonschema_description:"True if possible duplicates were found"`
	DuplicateAllowed    bool                             `json:"duplicate_allowed,omitzero" jsonschema_description:"True if the row is listed in allow_duplicates_rows"`
	DuplicateCandidates []*MCPImportDuplicateCandidate   `json:"duplicate_candidates,omitempty" jsonschema_description:"Possible duplicates (existing transactions or earlier rows)"`
	Mapping             *MCPImportTransactionsRowMapping `json:"mapping,omitempty" jsonschema_description:"Resolved accounts, category and tags"`
	ChangedFields       []string                         `json:"changed_fields,omitempty" jsonschema_description:"Fields which are (or would be) changed (update rows)"`
	Message             string                           `json:"message,omitempty" jsonschema_description:"Additional information (e.g. why the row is skipped)"`
	Error               *MCPToolError                    `json:"error,omitempty" jsonschema_description:"Structured error of a failed row (code, message, field, limit, actual, value, candidates)"`
}

// MCPImportTransactionsResponse represents the response structure for import transactions
type MCPImportTransactionsResponse struct {
	Success                bool                              `json:"success" jsonschema_description:"True if no row failed"`
	DryRun                 bool                              `json:"dry_run" jsonschema_description:"Whether this is a dry run (nothing saved)"`
	Committed              bool                              `json:"committed" jsonschema_description:"Whether at least one row has been saved by this call"`
	TotalRows              int                               `json:"total_rows" jsonschema_description:"Count of rows"`
	WouldBeCreatedCount    int                               `json:"would_be_created_count" jsonschema_description:"Count of valid new rows (dry run)"`
	WouldBeUpdatedCount    int                               `json:"would_be_updated_count" jsonschema_description:"Count of valid update rows (dry run)"`
	CreatedCount           int                               `json:"created_count" jsonschema_description:"Count of created transactions"`
	UpdatedCount           int                               `json:"updated_count" jsonschema_description:"Count of updated transactions"`
	UnchangedCount         int                               `json:"unchanged_count" jsonschema_description:"Count of update rows without changes"`
	SkippedCount           int                               `json:"skipped_count" jsonschema_description:"Count of rows skipped as possible duplicates"`
	FailedCount            int                               `json:"failed_count" jsonschema_description:"Count of invalid or failed rows"`
	ReplayedCount          int                               `json:"replayed_count" jsonschema_description:"Count of rows returned from an earlier call with the same idempotency key"`
	PossibleDuplicateCount int                               `json:"possible_duplicate_count" jsonschema_description:"Count of rows with possible duplicates"`
	Results                []*MCPImportTransactionsRowResult `json:"results" jsonschema_description:"Per-row results in request order"`
	BatchSemantics         string                            `json:"batch_semantics" jsonschema_description:"Commit semantics of this tool"`
	NextStep               string                            `json:"next_step,omitempty" jsonschema_description:"Suggested next step"`
}

// mcpImportRowToolHandler represents the tool handler (add_transaction or update_transaction) which validates and saves an import row
type mcpImportRowToolHandler interface {
	Handle(*core.WebContext, *MCPCallToolRequest, *models.User, *settings.Config, MCPAvailableServices) (any, []*MCPTextContent, error)
}

// mcpImportPreparedRow represents an import row after validation which does not depend on stored data
type mcpImportPreparedRow struct {
	index          int
	row            *MCPImportTransactionsRow
	result         *MCPImportTransactionsRowResult
	references     []*models.TransactionReference
	checksums      []string
	attachmentIds  []int64
	idempotencyKey string
}

type mcpImportTransactionsToolHandler struct{}

var MCPImportTransactionsToolHandler = &mcpImportTransactionsToolHandler{}

// ToolType returns the MCP tool type
func (h *mcpImportTransactionsToolHandler) ToolType() core.MCPToolType {
	return core.MCP_TOOL_TYPE_IMPORT_TRANSACTIONS
}

// Name returns the name of the MCP tool
func (h *mcpImportTransactionsToolHandler) Name() string {
	return "import_transactions"
}

// Description returns the description of the MCP tool
func (h *mcpImportTransactionsToolHandler) Description() string {
	return "Import a batch of transactions (rows with the same fields as add_transaction, or updates of existing transactions when transaction_id is set), with per-row idempotency keys and evidence references (e.g. external_source_id, invoice_number, source_document, source_row). dry_run defaults to true and returns per-row validation results, the resolved account / category / tag mappings (ambiguous names fail with candidates) and possible duplicates found by references, attachment checksums and the same amount, account and date (within one day). Rows with possible duplicates are skipped unless listed in allow_duplicates_rows. " + mcpImportBatchSemantics
}

// DataOperationType returns how the MCP tool operates on business records
func (h *mcpImportTransactionsToolHandler) DataOperationType() MCPToolDataOperationType {
	return MCP_TOOL_DATA_OPERATION_MODIFY
}

// IsIdempotent returns whether repeated calls with the same arguments have no additional effects
func (h *mcpImportTransactionsToolHandler) IsIdempotent() bool {
	return false
}

// DependsOnExternalData returns whether the MCP tool may depend on third-party data
func (h *mcpImportTransactionsToolHandler) DependsOnExternalData() bool {
	return false
}

// DefaultDryRun returns that the dry_run argument of this tool defaults to true
func (h *mcpImportTransactionsToolHandler) DefaultDryRun() bool {
	return true
}

// InputType returns the input type for the MCP tool request
func (h *mcpImportTransactionsToolHandler) InputType() reflect.Type {
	return reflect.TypeOf(&MCPImportTransactionsRequest{})
}

// OutputType returns the output type for the MCP tool response
func (h *mcpImportTransactionsToolHandler) OutputType() reflect.Type {
	return reflect.TypeOf(&MCPImportTransactionsResponse{})
}

// Handle processes the MCP call tool request and returns the response
func (h *mcpImportTransactionsToolHandler) Handle(c *core.WebContext, callToolReq *MCPCallToolRequest, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, []*MCPTextContent, error) {
	request, err := parseMCPToolArguments[MCPImportTransactionsRequest](callToolReq, false)

	if err != nil {
		return nil, nil, err
	}

	dryRun := request.DryRun == nil || *request.DryRun
	allowDuplicateRows, err := mcpImportValidateRequest(request)

	if err != nil {
		return nil, nil, err
	}

	uid := user.Uid
	preparedRows := mcpImportPrepareRows(request)
	index, err := h.buildExistingIndex(c, uid, preparedRows, services)

	if err != nil {
		return nil, nil, err
	}

	store := mcpIdempotencyStore(mcpGetIdempotencyRecordService(services))
	earlierRows := make([]*mcpImportRowFingerprint, 0, len(preparedRows))
	createdTransactionIds := make(map[string]bool)
	response := &MCPImportTransactionsResponse{
		DryRun:         dryRun,
		TotalRows:      len(preparedRows),
		Results:        make([]*MCPImportTransactionsRowResult, 0, len(preparedRows)),
		BatchSemantics: mcpImportBatchSemantics,
	}

	for i := 0; i < len(preparedRows); i++ {
		preparedRow := preparedRows[i]

		if preparedRow.result.Error == nil {
			fingerprint := h.processRow(c, user, preparedRow, dryRun, allowDuplicateRows[preparedRow.index+1], earlierRows, createdTransactionIds, index, store, currentConfig, services)

			if fingerprint != nil {
				earlierRows = append(earlierRows, fingerprint)
			}

			if preparedRow.result.Status == MCPWriteStatusCreated && preparedRow.result.TransactionId != "" {
				createdTransactionIds[preparedRow.result.TransactionId] = true
			}
		}

		if preparedRow.result.Error != nil {
			preparedRow.result.Status = MCPWriteStatusFailed
		}

		response.Results = append(response.Results, preparedRow.result)
	}

	mcpImportSummarizeResponse(response)

	return newMCPJSONTextResult(response)
}

// buildExistingIndex loads the existing references, attachment checksums and transactions which may be duplicates of the rows
func (h *mcpImportTransactionsToolHandler) buildExistingIndex(c *core.WebContext, uid int64, preparedRows []*mcpImportPreparedRow, services MCPAvailableServices) (*mcpImportExistingIndex, error) {
	referenceValues := make([]string, 0)
	attachmentIds := make([]int64, 0)

	for i := 0; i < len(preparedRows); i++ {
		preparedRow := preparedRows[i]

		for j := 0; j < len(preparedRow.references); j++ {
			referenceType := preparedRow.references[j].ReferenceType

			if models.IsTransactionReferenceTypeIdentifier(referenceType) || referenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT || referenceType == models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW {
				referenceValues = append(referenceValues, preparedRow.references[j].ReferenceValue)
			}
		}

		attachmentIds = append(attachmentIds, preparedRow.attachmentIds...)
	}

	referenceValues = mcpImportUniqueStrings(referenceValues)
	attachmentIds = utils.ToUniqueInt64Slice(attachmentIds)

	if len(attachmentIds) > 0 {
		pictureInfos, err := services.GetTransactionPictureService().GetNewPictureInfosByPictureIds(c, uid, attachmentIds)

		if err != nil {
			log.Errorf(c, "[import_transactions_tool_handler.buildExistingIndex] failed to get attachments for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}

		checksumsMap := make(map[int64]string, len(pictureInfos))

		for i := 0; i < len(pictureInfos); i++ {
			if pictureInfos[i].FileSha256 != "" {
				checksumsMap[pictureInfos[i].PictureId] = pictureInfos[i].FileSha256
			}
		}

		for i := 0; i < len(preparedRows); i++ {
			for _, attachmentId := range preparedRows[i].attachmentIds {
				if checksum, exists := checksumsMap[attachmentId]; exists {
					preparedRows[i].checksums = append(preparedRows[i].checksums, checksum)
				}
			}
		}
	}

	checksums := make([]string, 0)

	for i := 0; i < len(preparedRows); i++ {
		checksums = append(checksums, preparedRows[i].checksums...)
	}

	checksums = mcpImportUniqueStrings(checksums)

	var references []*models.TransactionReference
	var pictureInfos []*models.TransactionPictureInfo
	var err error

	if len(referenceValues) > 0 {
		referenceTypes := []string{
			models.TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID,
			models.TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER,
			models.TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER,
			models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT,
			models.TRANSACTION_REFERENCE_TYPE_SOURCE_ROW,
		}

		references, err = mcpGetTransactionReferenceService(services).GetReferencesByTypesAndValues(c, uid, referenceTypes, referenceValues)

		if err != nil {
			log.Errorf(c, "[import_transactions_tool_handler.buildExistingIndex] failed to get references for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	if len(checksums) > 0 {
		pictureInfos, err = services.GetTransactionPictureService().GetAttachedPictureInfosBySha256(c, uid, checksums)

		if err != nil {
			log.Errorf(c, "[import_transactions_tool_handler.buildExistingIndex] failed to get attachments by checksums for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	transactionIds := make([]int64, 0, len(references)+len(pictureInfos))

	for i := 0; i < len(references); i++ {
		transactionIds = append(transactionIds, references[i].TransactionId)
	}

	for i := 0; i < len(pictureInfos); i++ {
		transactionIds = append(transactionIds, pictureInfos[i].TransactionId)
	}

	transactionIds = utils.ToUniqueInt64Slice(transactionIds)
	var transactions []*models.Transaction

	if len(transactionIds) > 0 {
		transactions, err = services.GetTransactionService().GetTransactionsByTransactionIds(c, uid, transactionIds)

		if err != nil {
			log.Errorf(c, "[import_transactions_tool_handler.buildExistingIndex] failed to get transactions for user \"uid:%d\", because %s", uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	return mcpImportBuildExistingIndex(references, pictureInfos, transactions), nil
}

// processRow validates (and commits if not dry run) one prepared row, and returns the fingerprint of the row if it is valid and not skipped, so that later rows can be compared with it
func (h *mcpImportTransactionsToolHandler) processRow(c *core.WebContext, user *models.User, preparedRow *mcpImportPreparedRow, dryRun bool, duplicateAllowed bool, earlierRows []*mcpImportRowFingerprint, createdTransactionIds map[string]bool, index *mcpImportExistingIndex, store mcpIdempotencyStore, currentConfig *settings.Config, services MCPAvailableServices) *mcpImportRowFingerprint {
	result := preparedRow.result
	row := preparedRow.row
	uid := user.Uid

	var toolHandler mcpImportRowToolHandler
	var arguments any
	var addRequest *MCPAddTransactionRequest
	var updateRequest *MCPUpdateTransactionRequest

	if result.Action == mcpImportRowActionUpdate {
		updateRequest = mcpImportBuildUpdateRequest(row, preparedRow.references)
		toolHandler = MCPUpdateTransactionToolHandler
		arguments = updateRequest
	} else {
		addRequest = mcpImportBuildAddRequest(row, preparedRow.references)
		toolHandler = MCPAddTransactionToolHandler
		arguments = addRequest
	}

	argumentsContent, err := json.Marshal(arguments)

	if err != nil {
		result.Error = mcpImportConvertError(err)
		return nil
	}

	var pendingRecord *models.MCPIdempotencyRecord

	if preparedRow.idempotencyKey != "" {
		requestHash, err := mcpComputeWriteRequestHash(mcpImportRowIdempotencyToolName+"/"+result.Action, jsontext.Value(argumentsContent))

		if err != nil {
			result.Error = mcpImportConvertError(err)
			return nil
		}

		var replayResponse []byte

		if dryRun {
			replayResponse, err = mcpImportCheckRowIdempotencyKey(c, store, uid, preparedRow.idempotencyKey, mcpImportRowIdempotencyToolName+"/"+result.Action, requestHash)
		} else {
			pendingRecord, replayResponse, err = mcpIdempotencyBegin(c, store, uid, preparedRow.idempotencyKey, mcpImportRowIdempotencyToolName+"/"+result.Action, requestHash, time.Now())
		}

		if err != nil {
			result.Error = mcpImportConvertError(err)
			return nil
		}

		if replayResponse != nil {
			storedResult := &MCPImportTransactionsRowResult{}

			if err := json.Unmarshal(replayResponse, storedResult); err != nil {
				result.Error = mcpImportConvertError(err)
				return nil
			}

			storedResult.Row = result.Row
			storedResult.IdempotentReplay = true
			*result = *storedResult
			return nil
		}
	}

	abort := func() {
		if pendingRecord != nil {
			if err := store.DeleteRecord(c, uid, pendingRecord.RecordId); err != nil {
				log.Warnf(c, "[import_transactions_tool_handler.processRow] failed to delete pending idempotency record of row %d for user \"uid:%d\", because %s", result.Row, uid, err.Error())
			}
		}
	}

	// validate the row with the same rules as the actual write by a dry run of add_transaction / update_transaction
	dryRunResponse, err := h.callTool(c, toolHandler, arguments, true, user, currentConfig, services)

	if err != nil {
		result.Error = mcpImportConvertError(err)
		abort()
		return nil
	}

	var fingerprint *mcpImportRowFingerprint

	if addRequest != nil {
		addResponse := mcpImportGetAddResponse(dryRunResponse)

		if addResponse == nil || addResponse.Transaction == nil {
			result.Error = NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the row cannot be validated")
			abort()
			return nil
		}

		result.Mapping = mcpImportBuildMapping(addResponse.Transaction)
		fingerprint = h.buildFingerprint(preparedRow, addResponse.Transaction)

		if fingerprint != nil {
			amountDateMatches := h.findAmountDateAccountMatches(c, uid, fingerprint, services)
			result.DuplicateCandidates = mcpImportExcludeCandidates(mcpImportFindDuplicateCandidates(fingerprint, earlierRows, index, amountDateMatches), createdTransactionIds)
			result.PossibleDuplicate = len(result.DuplicateCandidates) > 0
			result.DuplicateAllowed = result.PossibleDuplicate && duplicateAllowed
		}

		if result.PossibleDuplicate && !duplicateAllowed {
			result.Status = MCPWriteStatusSkipped
			result.Message = "skipped as a possible duplicate, list row " + strconv.Itoa(result.Row) + " in allow_duplicates_rows to import it anyway"
			abort()
			return nil
		}
	} else {
		updateResponse := mcpImportGetUpdateResponse(dryRunResponse)

		if updateResponse == nil {
			result.Error = NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the row cannot be validated")
			abort()
			return nil
		}

		result.ChangedFields = updateResponse.ChangedFields

		if updateResponse.Transaction != nil {
			result.Mapping = mcpImportBuildMapping(updateResponse.Transaction)
		}

		if updateResponse.Status == MCPWriteStatusUnchanged {
			result.Status = MCPWriteStatusUnchanged
			result.TransactionId = row.TransactionId
			h.completeRow(c, store, uid, pendingRecord, result)
			return nil
		}
	}

	if dryRun {
		if result.Action == mcpImportRowActionUpdate {
			result.Status = MCPWriteStatusWouldBeUpdated
			result.TransactionId = row.TransactionId
		} else {
			result.Status = MCPWriteStatusWouldBeCreated
		}

		return fingerprint
	}

	commitResponse, err := h.callTool(c, toolHandler, arguments, false, user, currentConfig, services)

	if err != nil {
		result.Error = mcpImportConvertError(err)
		abort()
		return nil
	}

	if addRequest != nil {
		addResponse := mcpImportGetAddResponse(commitResponse)

		if addResponse == nil || !addResponse.Committed {
			result.Error = NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the transaction has not been saved")
			abort()
			return nil
		}

		result.Status = MCPWriteStatusCreated
		result.TransactionId = addResponse.TransactionId

		if addResponse.ReferencesError != "" {
			result.Message = "the transaction has been saved but its references could not be saved: " + addResponse.ReferencesError
		}
	} else {
		updateResponse := mcpImportGetUpdateResponse(commitResponse)

		if updateResponse == nil {
			result.Error = NewMCPToolError(MCPToolErrorCodeOperationFailed, "", "the transaction has not been updated")
			abort()
			return nil
		}

		result.TransactionId = row.TransactionId
		result.ChangedFields = updateResponse.ChangedFields

		if updateResponse.Committed {
			result.Status = MCPWriteStatusUpdated
		} else {
			result.Status = MCPWriteStatusUnchanged
		}

		if updateResponse.ReferencesError != "" {
			result.Message = "the transaction has been updated but its references could not be saved: " + updateResponse.ReferencesError
		}
	}

	h.completeRow(c, store, uid, pendingRecord, result)

	return fingerprint
}

// completeRow saves the row result for the idempotency key of the row
func (h *mcpImportTransactionsToolHandler) completeRow(c *core.WebContext, store mcpIdempotencyStore, uid int64, pendingRecord *models.MCPIdempotencyRecord, result *MCPImportTransactionsRowResult) {
	if pendingRecord == nil {
		return
	}

	content, err := json.Marshal(result)

	if err == nil {
		err = store.CompleteRecord(c, uid, pendingRecord.RecordId, string(content))
	}

	if err != nil {
		log.Errorf(c, "[import_transactions_tool_handler.completeRow] failed to save the result of row %d for user \"uid:%d\", because %s", result.Row, uid, err.Error())
	}
}

// callTool calls add_transaction or update_transaction with the arguments of the row
func (h *mcpImportTransactionsToolHandler) callTool(c *core.WebContext, toolHandler mcpImportRowToolHandler, arguments any, dryRun bool, user *models.User, currentConfig *settings.Config, services MCPAvailableServices) (any, error) {
	switch typedArguments := arguments.(type) {
	case *MCPAddTransactionRequest:
		typedArguments.DryRun = dryRun
	case *MCPUpdateTransactionRequest:
		typedArguments.DryRun = dryRun
	}

	content, err := json.Marshal(arguments)

	if err != nil {
		return nil, err
	}

	structuredResponse, _, err := toolHandler.Handle(c, &MCPCallToolRequest{
		Arguments: jsontext.Value(content),
	}, user, currentConfig, services)

	return structuredResponse, err
}

// buildFingerprint returns the duplicate fingerprint of the validated new row
func (h *mcpImportTransactionsToolHandler) buildFingerprint(preparedRow *mcpImportPreparedRow, detail *MCPTransactionDetail) *mcpImportRowFingerprint {
	accountId, err := utils.StringToInt64(detail.AccountId)

	if err != nil {
		return nil
	}

	destinationAccountId := int64(0)

	if detail.DestinationAccountId != "" {
		destinationAccountId, _ = utils.StringToInt64(detail.DestinationAccountId)
	}

	amount, err := parseMCPAmount("amount", preparedRow.row.Amount, true)

	if err != nil {
		return nil
	}

	transactionTime, err := parseMCPTime("time", preparedRow.row.Time, true)

	if err != nil {
		return nil
	}

	var transactionType models.TransactionDbType

	switch preparedRow.row.Type {
	case transactionTypeIncome:
		transactionType = models.TRANSACTION_DB_TYPE_INCOME
	case transactionTypeExpense:
		transactionType = models.TRANSACTION_DB_TYPE_EXPENSE
	case transactionTypeTransfer:
		transactionType = models.TRANSACTION_DB_TYPE_TRANSFER_OUT
	default:
		return nil
	}

	return mcpImportNewRowFingerprint(preparedRow.index+1, transactionType, accountId, destinationAccountId, amount, transactionTime.Unix(), preparedRow.references, preparedRow.checksums)
}

// findAmountDateAccountMatches returns the existing transactions of the same account and amount within one day of the row
func (h *mcpImportTransactionsToolHandler) findAmountDateAccountMatches(c *core.WebContext, uid int64, fingerprint *mcpImportRowFingerprint, services MCPAvailableServices) []*models.Transaction {
	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(fingerprint.unixTime + mcpImportDuplicateTimeWindowSeconds)
	minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(fingerprint.unixTime - mcpImportDuplicateTimeWindowSeconds)
	amountFilter := "eq:" + utils.Int64ToString(fingerprint.amount)

	transactions, err := services.GetTransactionService().GetTransactionsByMaxTime(c, uid, maxTransactionTime, minTransactionTime, 0, nil, []int64{fingerprint.accountId}, nil, false, amountFilter, "", core.MATCH_MODE_DEFAULT, false, 1, mcpImportMaxDuplicateCandidates*2, false, false)

	if err != nil {
		log.Warnf(c, "[import_transactions_tool_handler.findAmountDateAccountMatches] failed to get transactions of account \"id:%d\" for user \"uid:%d\", because %s", fingerprint.accountId, uid, err.Error())
		return nil
	}

	return transactions
}

// mcpImportExcludeCandidates removes the candidates which are transactions created by earlier rows of the same call (they are reported as earlier rows)
func mcpImportExcludeCandidates(candidates []*MCPImportDuplicateCandidate, createdTransactionIds map[string]bool) []*MCPImportDuplicateCandidate {
	if len(createdTransactionIds) < 1 {
		return candidates
	}

	result := make([]*MCPImportDuplicateCandidate, 0, len(candidates))

	for i := 0; i < len(candidates); i++ {
		if candidates[i].TransactionId != "" && createdTransactionIds[candidates[i].TransactionId] {
			continue
		}

		result = append(result, candidates[i])
	}

	return result
}

// mcpImportValidateRequest validates the batch-level parameters and returns the set of row numbers which allow duplicates
func mcpImportValidateRequest(request *MCPImportTransactionsRequest) (map[int]bool, error) {
	if len(request.Rows) < 1 {
		return nil, NewMCPRequiredFieldError("rows")
	}

	if len(request.Rows) > mcpImportMaxRows {
		return nil, NewMCPTooManyItemsError("rows", mcpImportMaxRows, len(request.Rows))
	}

	if err := checkMCPStringLength("source_document", strings.TrimSpace(request.SourceDocument), false, models.TransactionReferenceValueMaxLength); err != nil {
		return nil, err
	}

	allowDuplicateRows := make(map[int]bool, len(request.AllowDuplicatesRows))

	for i := 0; i < len(request.AllowDuplicatesRows); i++ {
		rowNumber := request.AllowDuplicatesRows[i]

		if rowNumber < 1 || rowNumber > len(request.Rows) {
			return nil, NewMCPInvalidFieldError("allow_duplicates_rows", strconv.Itoa(rowNumber), "\"allow_duplicates_rows\" must only contain row numbers between 1 and "+strconv.Itoa(len(request.Rows)))
		}

		allowDuplicateRows[rowNumber] = true
	}

	return allowDuplicateRows, nil
}

// mcpImportPrepareRows validates all fields of the rows which do not depend on stored data, invalid rows have an error in their result
func mcpImportPrepareRows(request *MCPImportTransactionsRequest) []*mcpImportPreparedRow {
	preparedRows := make([]*mcpImportPreparedRow, 0, len(request.Rows))
	sourceDocument := strings.TrimSpace(request.SourceDocument)
	idempotencyKeyRows := make(map[string]int)
	attachmentIdRows := make(map[int64]int)

	for i := 0; i < len(request.Rows); i++ {
		preparedRow := &mcpImportPreparedRow{
			index: i,
			row:   request.Rows[i],
			result: &MCPImportTransactionsRowResult{
				Row:    i + 1,
				Action: mcpImportRowActionCreate,
			},
		}

		preparedRows = append(preparedRows, preparedRow)
		row := request.Rows[i]

		if row == nil {
			preparedRow.result.Error = NewMCPRequiredFieldError("rows[" + strconv.Itoa(i) + "]")
			continue
		}

		if row.TransactionId != "" {
			preparedRow.result.Action = mcpImportRowActionUpdate

			if _, err := parseMCPRequiredId("transaction_id", row.TransactionId); err != nil {
				preparedRow.result.Error = mcpImportConvertError(err)
				continue
			}
		}

		if row.IdempotencyKey != "" {
			if err := mcpCheckClientProvidedId(mcpIdempotencyKeyArgumentName, row.IdempotencyKey, models.MCPIdempotencyKeyMaxLength); err != nil {
				preparedRow.result.Error = mcpImportConvertError(err)
				continue
			}

			if previousRow, exists := idempotencyKeyRows[row.IdempotencyKey]; exists {
				preparedRow.result.Error = NewMCPInvalidFieldError(mcpIdempotencyKeyArgumentName, row.IdempotencyKey, "\""+mcpIdempotencyKeyArgumentName+"\" is already used by row "+strconv.Itoa(previousRow))
				continue
			}

			idempotencyKeyRows[row.IdempotencyKey] = i + 1
			preparedRow.idempotencyKey = row.IdempotencyKey
			preparedRow.result.IdempotencyKey = row.IdempotencyKey
		}

		references, err := mcpReferenceParse("references", row.References)

		if err != nil {
			preparedRow.result.Error = mcpImportConvertError(err)
			continue
		}

		// the source document is only added to update rows which also replace their references, so that existing references are not replaced by the source document only
		if sourceDocument != "" && (preparedRow.result.Action == mcpImportRowActionCreate || len(references) > 0) && mcpReferenceFindValue(references, models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT) == "" {
			references = append(references, &models.TransactionReference{
				ReferenceType:  models.TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT,
				ReferenceValue: sourceDocument,
			})

			if len(references) > models.MaximumReferencesCountOfTransaction {
				preparedRow.result.Error = NewMCPTooManyItemsError("references", models.MaximumReferencesCountOfTransaction, len(references))
				continue
			}
		}

		preparedRow.references = references
		checksums, err := mcpImportParseChecksums("attachment_sha256", row.AttachmentSHA256)

		if err != nil {
			preparedRow.result.Error = mcpImportConvertError(err)
			continue
		}

		preparedRow.checksums = checksums
		attachmentIds := make([]int64, 0, len(row.AttachmentIds))
		var attachmentErr error

		for j := 0; j < len(row.AttachmentIds); j++ {
			attachmentId, err := parseMCPRequiredId("attachment_ids", row.AttachmentIds[j])

			if err != nil {
				attachmentErr = err
				break
			}

			if previousRow, exists := attachmentIdRows[attachmentId]; exists && previousRow != i+1 {
				attachmentErr = NewMCPInvalidFieldError("attachment_ids", row.AttachmentIds[j], "attachment \""+row.AttachmentIds[j]+"\" is already used by row "+strconv.Itoa(previousRow)+", an attachment can only be attached to one transaction")
				break
			}

			attachmentIdRows[attachmentId] = i + 1
			attachmentIds = append(attachmentIds, attachmentId)
		}

		if attachmentErr != nil {
			preparedRow.result.Error = mcpImportConvertError(attachmentErr)
			continue
		}

		preparedRow.attachmentIds = attachmentIds

		if preparedRow.result.Action == mcpImportRowActionCreate {
			if row.Type == "" {
				preparedRow.result.Error = NewMCPRequiredFieldError("type")
			} else if row.Time == "" {
				preparedRow.result.Error = NewMCPRequiredFieldError("time")
			} else if row.Amount == "" {
				preparedRow.result.Error = NewMCPRequiredFieldError("amount")
			}
		}
	}

	return preparedRows
}

// mcpImportParseChecksums validates the hex encoded SHA-256 checksums and returns them in lower case
func mcpImportParseChecksums(field string, checksums []string) ([]string, error) {
	if len(checksums) > models.MaximumPicturesCountOfTransaction {
		return nil, NewMCPTooManyItemsError(field, models.MaximumPicturesCountOfTransaction, len(checksums))
	}

	result := make([]string, 0, len(checksums))

	for i := 0; i < len(checksums); i++ {
		checksum := strings.ToLower(strings.TrimSpace(checksums[i]))
		decoded, err := hex.DecodeString(checksum)

		if err != nil || len(decoded) != 32 {
			return nil, NewMCPInvalidFieldError(field, checksums[i], "\""+field+"\" must only contain hex encoded SHA-256 checksums (64 hexadecimal characters)")
		}

		result = append(result, checksum)
	}

	return mcpImportUniqueStrings(result), nil
}

// mcpImportCheckRowIdempotencyKey checks the idempotency key of a row in dry run (without creating a pending record), returns the stored result of the row if it has already been committed
func mcpImportCheckRowIdempotencyKey(c core.Context, store mcpIdempotencyStore, uid int64, idempotencyKey string, toolName string, requestHash string) ([]byte, error) {
	record, err := store.GetRecordByKey(c, uid, idempotencyKey)

	if err != nil {
		return nil, err
	}

	now := time.Now()

	if record == nil || record.CreatedUnixTime < now.Add(-mcpIdempotencyRecordRetention).Unix() {
		return nil, nil
	}

	return mcpIdempotencyCheckExistingRecord(record, idempotencyKey, toolName, requestHash, now)
}

// mcpImportBuildAddRequest returns the add_transaction arguments of the row
func mcpImportBuildAddRequest(row *MCPImportTransactionsRow, references []*models.TransactionReference) *MCPAddTransactionRequest {
	return &MCPAddTransactionRequest{
		Type:                   row.Type,
		Time:                   row.Time,
		SecondaryCategoryName:  row.SecondaryCategoryName,
		CategoryId:             row.CategoryId,
		AccountName:            row.AccountName,
		AccountId:              row.AccountId,
		Amount:                 row.Amount,
		DestinationAccountName: row.DestinationAccountName,
		DestinationAccountId:   row.DestinationAccountId,
		DestinationAmount:      row.DestinationAmount,
		Tags:                   row.Tags,
		TagIds:                 row.TagIds,
		Comment:                row.Comment,
		GeoLocation:            row.GeoLocation,
		AttachmentIds:          row.AttachmentIds,
		References:             mcpReferenceBuildInfos(references),
	}
}

// mcpImportBuildUpdateRequest returns the update_transaction arguments of the row, only non-empty fields are changed
func mcpImportBuildUpdateRequest(row *MCPImportTransactionsRow, references []*models.TransactionReference) *MCPUpdateTransactionRequest {
	optional := func(value string) *string {
		if value == "" {
			return nil
		}

		return &value
	}

	return &MCPUpdateTransactionRequest{
		TransactionId:          row.TransactionId,
		Type:                   optional(row.Type),
		Time:                   optional(row.Time),
		Amount:                 optional(row.Amount),
		AccountId:              optional(row.AccountId),
		AccountName:            optional(row.AccountName),
		DestinationAccountId:   optional(row.DestinationAccountId),
		DestinationAccountName: optional(row.DestinationAccountName),
		DestinationAmount:      optional(row.DestinationAmount),
		CategoryId:             optional(row.CategoryId),
		CategoryName:           optional(row.SecondaryCategoryName),
		TagIds:                 row.TagIds,
		TagNames:               row.Tags,
		Comment:                optional(row.Comment),
		GeoLocation:            row.GeoLocation,
		AddAttachmentIds:       row.AttachmentIds,
		References:             mcpReferenceBuildInfos(references),
	}
}

// mcpImportBuildMapping returns the resolved accounts, category and tags of the validated transaction
func mcpImportBuildMapping(detail *MCPTransactionDetail) *MCPImportTransactionsRowMapping {
	return &MCPImportTransactionsRowMapping{
		AccountId:              detail.AccountId,
		AccountName:            detail.AccountName,
		DestinationAccountId:   detail.DestinationAccountId,
		DestinationAccountName: detail.DestinationAccountName,
		CategoryId:             detail.CategoryId,
		CategoryName:           detail.CategoryName,
		PrimaryCategoryName:    detail.PrimaryCategoryName,
		Tags:                   detail.Tags,
	}
}

// mcpImportGetAddResponse returns the add_transaction response of the structured result
func mcpImportGetAddResponse(structuredResponse any) *MCPAddTransactionResponse {
	switch response := structuredResponse.(type) {
	case MCPAddTransactionResponse:
		return &response
	case *MCPAddTransactionResponse:
		return response
	default:
		return nil
	}
}

// mcpImportGetUpdateResponse returns the update_transaction response of the structured result
func mcpImportGetUpdateResponse(structuredResponse any) *MCPUpdateTransactionResponse {
	switch response := structuredResponse.(type) {
	case MCPUpdateTransactionResponse:
		return &response
	case *MCPUpdateTransactionResponse:
		return response
	default:
		return nil
	}
}

// mcpImportConvertError returns the structured error of a failed row
func mcpImportConvertError(err error) *MCPToolError {
	var toolErr *MCPToolError

	if errors.As(err, &toolErr) {
		return toolErr
	}

	return NewMCPToolError(MCPToolErrorCodeOperationFailed, "", err.Error())
}

// mcpImportSummarizeResponse counts the row statuses and sets the summary fields of the response
func mcpImportSummarizeResponse(response *MCPImportTransactionsResponse) {
	for i := 0; i < len(response.Results); i++ {
		result := response.Results[i]

		if result.IdempotentReplay {
			response.ReplayedCount++
		}

		if result.PossibleDuplicate {
			response.PossibleDuplicateCount++
		}

		switch result.Status {
		case MCPWriteStatusWouldBeCreated:
			response.WouldBeCreatedCount++
		case MCPWriteStatusWouldBeUpdated:
			response.WouldBeUpdatedCount++
		case MCPWriteStatusCreated:
			response.CreatedCount++

			if !result.IdempotentReplay {
				response.Committed = true
			}
		case MCPWriteStatusUpdated:
			response.UpdatedCount++

			if !result.IdempotentReplay {
				response.Committed = true
			}
		case MCPWriteStatusUnchanged:
			response.UnchangedCount++
		case MCPWriteStatusSkipped:
			response.SkippedCount++
		case MCPWriteStatusFailed:
			response.FailedCount++
		}
	}

	response.Success = response.FailedCount == 0

	if response.DryRun {
		response.NextStep = "Review the per-row results and mappings, fix failed rows, then call again with dry_run=false and the same rows (and idempotency keys) to commit. Rows flagged as possible duplicates are skipped unless listed in allow_duplicates_rows."
	} else if response.FailedCount > 0 {
		response.NextStep = "Fix the failed rows and rerun them (rerunning all rows with the same per-row idempotency keys is safe, committed rows are replayed and not created again)."
	}
}
