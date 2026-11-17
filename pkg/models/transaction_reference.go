package models

// Transaction reference types
const (
	TRANSACTION_REFERENCE_TYPE_SUPPLIER           = "supplier"
	TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER     = "invoice_number"
	TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER     = "receipt_number"
	TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID = "external_source_id"
	TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT    = "source_document"
	TRANSACTION_REFERENCE_TYPE_SOURCE_ROW         = "source_row"
)

// TransactionReferenceValueMaxLength is the maximum length (in characters) of the value of a transaction reference
const TransactionReferenceValueMaxLength = 255

// MaximumReferencesCountOfTransaction is the maximum count of references of one transaction
const MaximumReferencesCountOfTransaction = 20

// AllTransactionReferenceTypes contains all supported transaction reference types
var AllTransactionReferenceTypes = []string{
	TRANSACTION_REFERENCE_TYPE_SUPPLIER,
	TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER,
	TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER,
	TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID,
	TRANSACTION_REFERENCE_TYPE_SOURCE_DOCUMENT,
	TRANSACTION_REFERENCE_TYPE_SOURCE_ROW,
}

// TransactionReference represents a structured evidence / source reference of a transaction stored in database,
// e.g. the supplier, invoice number, receipt number, external source id, source document or source row
type TransactionReference struct {
	ReferenceId     int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_transaction_reference_uid_transaction_id) INDEX(IDX_transaction_reference_uid_type_value) NOT NULL"`
	TransactionId   int64  `xorm:"INDEX(IDX_transaction_reference_uid_transaction_id) NOT NULL"`
	ReferenceType   string `xorm:"VARCHAR(32) INDEX(IDX_transaction_reference_uid_type_value) NOT NULL"`
	ReferenceValue  string `xorm:"VARCHAR(255) INDEX(IDX_transaction_reference_uid_type_value) NOT NULL"`
	DisplayOrder    int32  `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
}

// IsValidTransactionReferenceType returns whether the reference type is supported
func IsValidTransactionReferenceType(referenceType string) bool {
	for i := 0; i < len(AllTransactionReferenceTypes); i++ {
		if AllTransactionReferenceTypes[i] == referenceType {
			return true
		}
	}

	return false
}

// IsTransactionReferenceTypeIdentifier returns whether the reference type identifies one source record by itself
// (external source id, invoice number and receipt number), so that the same value indicates a possible duplicate
func IsTransactionReferenceTypeIdentifier(referenceType string) bool {
	return referenceType == TRANSACTION_REFERENCE_TYPE_EXTERNAL_SOURCE_ID ||
		referenceType == TRANSACTION_REFERENCE_TYPE_INVOICE_NUMBER ||
		referenceType == TRANSACTION_REFERENCE_TYPE_RECEIPT_NUMBER
}
