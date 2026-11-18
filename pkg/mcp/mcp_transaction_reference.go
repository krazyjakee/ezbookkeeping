package mcp

import (
	"strconv"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// mcpReferenceTypesDescription is the description of all supported reference types
const mcpReferenceTypesDescription = "supplier, invoice_number, receipt_number, external_source_id, source_document, source_row"

// MCPTransactionReference represents a structured evidence / source reference of a transaction
type MCPTransactionReference struct {
	Type  string `json:"type" jsonschema:"enum=supplier,enum=invoice_number,enum=receipt_number,enum=external_source_id,enum=source_document,enum=source_row" jsonschema_description:"Reference type: supplier (supplier / merchant name), invoice_number, receipt_number, external_source_id (ID of the record in the source system, e.g. bank transaction ID), source_document (e.g. statement file name), source_row (row / line number in the source document)"`
	Value string `json:"value" jsonschema_description:"Reference value (maximum length is reported by get_server_capabilities)"`
}

// mcpReferenceParse validates the references of the request and returns the reference models (without ids) in request order,
// duplicated references (same type and value) are removed
func mcpReferenceParse(field string, references []*MCPTransactionReference) ([]*models.TransactionReference, error) {
	result := make([]*models.TransactionReference, 0, len(references))

	if len(references) < 1 {
		return result, nil
	}

	if len(references) > models.MaximumReferencesCountOfTransaction {
		return nil, NewMCPTooManyItemsError(field, models.MaximumReferencesCountOfTransaction, len(references))
	}

	exists := make(map[string]bool, len(references))

	for i := 0; i < len(references); i++ {
		itemField := field + "[" + strconv.Itoa(i) + "]"
		reference := references[i]

		if reference == nil {
			return nil, NewMCPRequiredFieldError(itemField)
		}

		referenceType := strings.TrimSpace(reference.Type)

		if !models.IsValidTransactionReferenceType(referenceType) {
			return nil, NewMCPInvalidFieldError(itemField+".type", reference.Type, "\""+itemField+".type\" must be one of "+mcpReferenceTypesDescription)
		}

		value := strings.TrimSpace(reference.Value)

		if err := checkMCPStringLength(itemField+".value", value, true, models.TransactionReferenceValueMaxLength); err != nil {
			return nil, err
		}

		key := referenceType + "\n" + value

		if exists[key] {
			continue
		}

		exists[key] = true
		result = append(result, &models.TransactionReference{
			ReferenceType:  referenceType,
			ReferenceValue: value,
		})
	}

	return result, nil
}

// mcpReferenceBuildInfos returns the textual references of the reference models
func mcpReferenceBuildInfos(references []*models.TransactionReference) []*MCPTransactionReference {
	if len(references) < 1 {
		return nil
	}

	result := make([]*MCPTransactionReference, 0, len(references))

	for i := 0; i < len(references); i++ {
		result = append(result, &MCPTransactionReference{
			Type:  references[i].ReferenceType,
			Value: references[i].ReferenceValue,
		})
	}

	return result
}

// mcpReferenceListEquals returns whether both reference lists contain the same references in the same order
func mcpReferenceListEquals(references1 []*models.TransactionReference, references2 []*models.TransactionReference) bool {
	if len(references1) != len(references2) {
		return false
	}

	for i := 0; i < len(references1); i++ {
		if references1[i].ReferenceType != references2[i].ReferenceType || references1[i].ReferenceValue != references2[i].ReferenceValue {
			return false
		}
	}

	return true
}

// mcpReferenceFindValue returns the first value of the given reference type, or empty if not found
func mcpReferenceFindValue(references []*models.TransactionReference, referenceType string) string {
	for i := 0; i < len(references); i++ {
		if references[i].ReferenceType == referenceType {
			return references[i].ReferenceValue
		}
	}

	return ""
}

// mcpReferenceLoad returns the references of the given transactions, errors are logged and an empty map is returned,
// so that reading transactions does not fail if references cannot be read
func mcpReferenceLoad(c *core.WebContext, uid int64, transactionIds []int64, services MCPAvailableServices) map[int64][]*models.TransactionReference {
	if len(transactionIds) < 1 {
		return make(map[int64][]*models.TransactionReference)
	}

	references, err := mcpGetTransactionReferenceService(services).GetReferencesByTransactionIds(c, uid, transactionIds)

	if err != nil {
		log.Warnf(c, "[mcp_transaction_reference.mcpReferenceLoad] failed to get references of transactions for user \"uid:%d\", because %s", uid, err.Error())
		return make(map[int64][]*models.TransactionReference)
	}

	return references
}
