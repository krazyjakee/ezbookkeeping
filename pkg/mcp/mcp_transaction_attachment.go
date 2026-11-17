package mcp

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpAttachmentResourceUriPrefix is the prefix of the uri of transaction attachments returned as embedded resources
const mcpAttachmentResourceUriPrefix = "ezbookkeeping://transaction-attachments/"

// mcpAttachmentLimitationNote describes the limitations of transaction attachments which are shown in tool descriptions
const mcpAttachmentLimitationNote = "The original file name, byte size and SHA-256 checksum are recorded at upload (attachments uploaded by older versions have no recorded metadata), and the SHA-256 checksum is also computed from the stored bytes when the attachment is read, so integrity can be verified."

// Sources of the SHA-256 checksum of transaction attachments
const (
	mcpAttachmentChecksumSourceStoredFile     = "stored_file"
	mcpAttachmentChecksumSourceUploadMetadata = "upload_metadata"
)

// mcpAttachmentContentTypeExtensionMap is the map of accepted content types (MIME types) and the file extensions which are saved
var mcpAttachmentContentTypeExtensionMap = map[string]string{
	"image/jpeg":      "jpg",
	"image/jpg":       "jpg",
	"image/pjpeg":     "jpg",
	"image/png":       "png",
	"image/gif":       "gif",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

// MCPTransactionAttachmentInfo represents the information of a transaction attachment (receipt picture or document)
type MCPTransactionAttachmentInfo struct {
	Id                string `json:"id" jsonschema_description:"Attachment ID"`
	Attached          bool   `json:"attached" jsonschema_description:"Whether the attachment is attached to a transaction (false means it is an uploaded but unattached attachment, which can be attached via add_transaction or update_transaction)"`
	TransactionId     string `json:"transaction_id,omitempty" jsonschema_description:"ID of the transaction which the attachment is attached to (empty if unattached)"`
	Extension         string `json:"extension" jsonschema_description:"Stored file extension of the attachment"`
	ContentType       string `json:"content_type" jsonschema_description:"MIME type of the attachment"`
	OriginalFileName  string `json:"original_file_name,omitempty" jsonschema_description:"Original file name recorded at upload (sanitized, empty for attachments uploaded by older versions or without a file name)"`
	SizeBytes         int64  `json:"size_bytes,omitempty" jsonschema_description:"Size in bytes of the stored file (or the size recorded at upload if the stored file is not available)"`
	SHA256            string `json:"sha256,omitempty" jsonschema_description:"Hex encoded SHA-256 checksum computed from the stored file (or the checksum recorded at upload if the stored file is not available)"`
	ChecksumSource    string `json:"checksum_source,omitempty" jsonschema:"enum=stored_file,enum=upload_metadata" jsonschema_description:"Where sha256 comes from: stored_file (computed from the stored bytes) or upload_metadata (recorded at upload, the stored file is not available)"`
	RecordedSizeBytes int64  `json:"recorded_size_bytes,omitempty" jsonschema_description:"Size in bytes recorded at upload (empty for attachments uploaded by older versions)"`
	RecordedSHA256    string `json:"recorded_sha256,omitempty" jsonschema_description:"Hex encoded SHA-256 checksum recorded at upload (empty for attachments uploaded by older versions)"`
	IntegrityVerified *bool  `json:"integrity_verified,omitempty" jsonschema_description:"Whether the stored file matches the size and checksum recorded at upload (only set when both are available)"`
	ContentAvailable  bool   `json:"content_available" jsonschema_description:"Whether the stored file can be read from the server storage"`
	CreatedTime       string `json:"created_time,omitempty" jsonschema_description:"Upload time in RFC 3339 format (UTC)"`
	UpdatedTime       string `json:"updated_time,omitempty" jsonschema_description:"Last update (attach / detach) time in RFC 3339 format (UTC)"`
}

// mcpAttachmentCheckEnabled returns an error if transaction attachments are disabled on this server
func mcpAttachmentCheckEnabled(currentConfig *settings.Config) error {
	if currentConfig == nil || !currentConfig.EnableTransactionPictures {
		return NewMCPNotSupportedError("", "transaction attachments are disabled on this server (enable_transaction_picture is false)")
	}

	return nil
}

// mcpAttachmentGetMaxFileSize returns the maximum size (in bytes) of one attachment file
func mcpAttachmentGetMaxFileSize(currentConfig *settings.Config) int64 {
	if currentConfig == nil {
		return 0
	}

	return int64(currentConfig.MaxTransactionPictureFileSize)
}

// mcpAttachmentGetMaxBase64Length returns the maximum length of the base64 encoded content of one attachment file
func mcpAttachmentGetMaxBase64Length(maxFileSize int64) int64 {
	if maxFileSize <= 0 {
		return 0
	}

	return int64(base64.StdEncoding.EncodedLen(int(maxFileSize)))
}

// mcpAttachmentResolveExtension returns the file extension which is saved according to the specified file extension and / or content type
func mcpAttachmentResolveExtension(fileExtension string, contentType string) (string, error) {
	extension := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(fileExtension), "."))
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	if index := strings.Index(contentType, ";"); index >= 0 {
		contentType = strings.TrimSpace(contentType[:index])
	}

	if extension == "" && contentType == "" {
		return "", NewMCPRequiredFieldError("file_extension or content_type")
	}

	if extension != "" && utils.GetTransactionAttachmentContentType(extension) == "" {
		return "", NewMCPInvalidFieldError("file_extension", fileExtension, "\"file_extension\" is not supported, supported file types are reported by get_server_capabilities")
	}

	if contentType != "" {
		contentTypeExtension, exists := mcpAttachmentContentTypeExtensionMap[contentType]

		if !exists || utils.GetTransactionAttachmentContentType(contentTypeExtension) == "" {
			return "", NewMCPInvalidFieldError("content_type", contentType, "\"content_type\" is not supported, supported file types are reported by get_server_capabilities")
		}

		if extension == "" {
			extension = contentTypeExtension
		} else if utils.GetTransactionAttachmentContentType(extension) != utils.GetTransactionAttachmentContentType(contentTypeExtension) {
			return "", NewMCPInvalidFieldError("content_type", contentType, "\"content_type\" does not match \"file_extension\" ("+extension+")")
		}
	}

	return extension, nil
}

// mcpAttachmentDecodeContent decodes the base64 encoded attachment content (a "data:<mime>;base64," prefix is allowed), and checks the size is not empty and does not exceed the maximum size.
// The length of the encoded content is checked before decoding, so oversized content is rejected without being decoded
func mcpAttachmentDecodeContent(field string, content string, maxFileSize int64) ([]byte, error) {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "data:") {
		index := strings.Index(content, ",")

		if index < 0 || !strings.HasSuffix(strings.ToLower(content[:index]), ";base64") {
			return nil, NewMCPInvalidFieldError(field, "", "\""+field+"\" must be base64 encoded, data URLs must use \";base64\" encoding")
		}

		content = content[index+1:]
	}

	content = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}

		return r
	}, content)

	if content == "" {
		return nil, NewMCPRequiredFieldError(field)
	}

	if maxFileSize > 0 && int64(len(content)) > mcpAttachmentGetMaxBase64Length(maxFileSize) {
		return nil, NewMCPTooLargeError(field, maxFileSize, int64(base64.StdEncoding.DecodedLen(len(content))))
	}

	data, err := base64.StdEncoding.DecodeString(content)

	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(content)
	}

	if err != nil {
		return nil, NewMCPInvalidFieldError(field, "", "\""+field+"\" must be base64 encoded (standard alphabet, RFC 4648)")
	}

	if len(data) < 1 {
		return nil, NewMCPInvalidFieldError(field, "", "\""+field+"\" must not be empty")
	}

	if maxFileSize > 0 && int64(len(data)) > maxFileSize {
		return nil, NewMCPTooLargeError(field, maxFileSize, int64(len(data)))
	}

	return data, nil
}

// mcpAttachmentValidateContent checks whether the content matches the file type, using the same magic bytes check as the HTTP upload, and additionally checks the detected type of images
func mcpAttachmentValidateContent(field string, extension string, data []byte) error {
	contentType := utils.GetTransactionAttachmentContentType(extension)

	if contentType == "" {
		return NewMCPInvalidFieldError("file_extension", extension, "\"file_extension\" is not supported, supported file types are reported by get_server_capabilities")
	}

	headerLength := utils.GetTransactionAttachmentMagicBytesLength(extension)
	fileHeader := data

	if headerLength > 0 && len(fileHeader) > headerLength {
		fileHeader = fileHeader[:headerLength]
	}

	if !utils.IsTransactionAttachmentContentValid(extension, fileHeader) {
		return NewMCPInvalidFieldError(field, "", "the content of \""+field+"\" does not match the file type "+contentType)
	}

	if imageContentType := utils.GetImageContentType(extension); imageContentType != "" {
		detectedContentType := http.DetectContentType(data)

		if detectedContentType != imageContentType {
			return NewMCPInvalidFieldError(field, "", "the content of \""+field+"\" is detected as "+detectedContentType+" which does not match the file type "+imageContentType)
		}
	}

	return nil
}

// mcpAttachmentComputeChecksum returns the hex encoded SHA-256 checksum of the content
func mcpAttachmentComputeChecksum(data []byte) string {
	checksum := sha256.Sum256(data)
	return hex.EncodeToString(checksum[:])
}

// mcpAttachmentGetResourceUri returns the uri of the attachment which is used in embedded resources
func mcpAttachmentGetResourceUri(pictureInfo *models.TransactionPictureInfo) string {
	return mcpAttachmentResourceUriPrefix + utils.Int64ToString(pictureInfo.PictureId) + "." + pictureInfo.PictureExtension
}

// mcpAttachmentBuildInfo returns the attachment information according to the picture info and the stored content (nil if not available)
func mcpAttachmentBuildInfo(pictureInfo *models.TransactionPictureInfo, data []byte, contentAvailable bool) *MCPTransactionAttachmentInfo {
	info := &MCPTransactionAttachmentInfo{
		Id:               utils.Int64ToString(pictureInfo.PictureId),
		Attached:         pictureInfo.TransactionId != models.TransactionPictureNewPictureTransactionId,
		Extension:        pictureInfo.PictureExtension,
		ContentType:      utils.GetTransactionAttachmentContentType(pictureInfo.PictureExtension),
		ContentAvailable: contentAvailable,
	}

	if info.Attached {
		info.TransactionId = utils.Int64ToString(pictureInfo.TransactionId)
	}

	info.OriginalFileName = pictureInfo.OriginalFileName
	info.RecordedSizeBytes = pictureInfo.FileSize
	info.RecordedSHA256 = pictureInfo.FileSha256

	if contentAvailable {
		info.SizeBytes = int64(len(data))
		info.SHA256 = mcpAttachmentComputeChecksum(data)
		info.ChecksumSource = mcpAttachmentChecksumSourceStoredFile

		if pictureInfo.FileSha256 != "" {
			verified := info.SHA256 == pictureInfo.FileSha256 && (pictureInfo.FileSize <= 0 || info.SizeBytes == pictureInfo.FileSize)
			info.IntegrityVerified = &verified
		}
	} else if pictureInfo.FileSha256 != "" {
		info.SizeBytes = pictureInfo.FileSize
		info.SHA256 = pictureInfo.FileSha256
		info.ChecksumSource = mcpAttachmentChecksumSourceUploadMetadata
	}

	if pictureInfo.CreatedUnixTime > 0 {
		info.CreatedTime = formatMCPUnixTime(pictureInfo.CreatedUnixTime)
	}

	if pictureInfo.UpdatedUnixTime > 0 {
		info.UpdatedTime = formatMCPUnixTime(pictureInfo.UpdatedUnixTime)
	}

	return info
}

// mcpAttachmentReadContent reads the stored content of the attachment, returns false if the file does not exist in the storage
func mcpAttachmentReadContent(c *core.WebContext, uid int64, pictureInfo *models.TransactionPictureInfo, services MCPAvailableServices) ([]byte, bool, error) {
	data, err := services.GetTransactionPictureService().GetPictureByPictureId(c, uid, pictureInfo.PictureId, pictureInfo.PictureExtension)

	if err != nil {
		if err == errs.ErrTransactionPictureNoExists {
			log.Warnf(c, "[mcp_transaction_attachment.mcpAttachmentReadContent] the file of transaction attachment \"id:%d\" does not exist for user \"uid:%d\"", pictureInfo.PictureId, uid)
			return nil, false, nil
		}

		log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentReadContent] failed to read transaction attachment \"id:%d\" for user \"uid:%d\", because %s", pictureInfo.PictureId, uid, err.Error())
		return nil, false, err
	}

	return data, true, nil
}

// mcpAttachmentLoadInfo returns the attachment information including the size and checksum computed from the stored content
func mcpAttachmentLoadInfo(c *core.WebContext, uid int64, pictureInfo *models.TransactionPictureInfo, services MCPAvailableServices) (*MCPTransactionAttachmentInfo, error) {
	data, contentAvailable, err := mcpAttachmentReadContent(c, uid, pictureInfo, services)

	if err != nil {
		return nil, err
	}

	return mcpAttachmentBuildInfo(pictureInfo, data, contentAvailable), nil
}

// mcpAttachmentGetPictureInfo returns the picture info of the attachment specified by the textual id
func mcpAttachmentGetPictureInfo(c *core.WebContext, uid int64, field string, attachmentId string, services MCPAvailableServices) (*models.TransactionPictureInfo, error) {
	pictureId, err := parseMCPRequiredId(field, attachmentId)

	if err != nil {
		return nil, err
	}

	pictureInfo, err := services.GetTransactionPictureService().GetPictureInfoByPictureId(c, uid, pictureId)

	if err != nil {
		if err == errs.ErrTransactionPictureNotFound {
			return nil, NewMCPNotFoundError(field, "attachment", attachmentId)
		}

		log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentGetPictureInfo] failed to get transaction attachment \"id:%d\" for user \"uid:%d\", because %s", pictureId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return pictureInfo, nil
}

// mcpAttachmentGetTransaction returns the transaction specified by the textual id, the incoming side of a transfer is redirected to the transfer (outgoing side) which holds the attachments
func mcpAttachmentGetTransaction(c *core.WebContext, uid int64, field string, transactionId string, services MCPAvailableServices) (*models.Transaction, error) {
	id, err := parseMCPRequiredId(field, transactionId)

	if err != nil {
		return nil, err
	}

	transaction, err := services.GetTransactionService().GetTransactionByTransactionId(c, uid, id)

	if err != nil {
		if err == errs.ErrTransactionNotFound {
			return nil, NewMCPNotFoundError(field, "transaction", transactionId)
		}

		log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentGetTransaction] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_IN {
		transaction, err = services.GetTransactionService().GetTransactionByTransactionId(c, uid, transaction.RelatedId)

		if err != nil {
			log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentGetTransaction] failed to get related transaction of transaction \"id:%d\" for user \"uid:%d\", because %s", id, uid, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
	}

	return transaction, nil
}

// mcpAttachmentCheckTransactionEditable checks whether the attachments of the transaction can be changed by the user with the same rules as modifying the transaction
// (accounts must exist and be visible, and the transaction must be in the user's transaction edit scope)
func mcpAttachmentCheckTransactionEditable(c *core.WebContext, uid int64, user *models.User, field string, transaction *models.Transaction, services MCPAvailableServices) error {
	accountIds := []int64{transaction.AccountId}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		accountIds = append(accountIds, transaction.RelatedAccountId)
	}

	accountMap, err := services.GetAccountService().GetAccountsByAccountIds(c, uid, accountIds)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentCheckTransactionEditable] failed to get accounts for user \"uid:%d\", because %s", uid, err.Error())
		return errs.Or(err, errs.ErrOperationFailed)
	}

	sourceAccount := accountMap[transaction.AccountId]

	if sourceAccount == nil {
		return errs.ErrSourceAccountNotFound
	}

	var destinationAccount *models.Account

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		destinationAccount = accountMap[transaction.RelatedAccountId]

		if destinationAccount == nil {
			return errs.ErrDestinationAccountNotFound
		}
	}

	if sourceAccount.Hidden || (destinationAccount != nil && destinationAccount.Hidden) {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, field, "transaction \""+utils.Int64ToString(transaction.TransactionId)+"\" belongs to a hidden account, transactions in hidden accounts cannot be modified")
	}

	transactionTimezone := time.FixedZone("Transaction Timezone", int(transaction.TimezoneUtcOffset)*60)

	if user == nil || !user.CanEditTransactionByTransactionTime(transaction.TransactionTime, transactionTimezone, sourceAccount, destinationAccount) {
		return NewMCPToolError(MCPToolErrorCodeNotAllowed, field, "transaction \""+utils.Int64ToString(transaction.TransactionId)+"\" is out of the user's transaction edit scope (see get_server_capabilities), its attachments cannot be changed")
	}

	return nil
}

// mcpAttachmentCheckNewAttachments checks whether the specified attachments are uploaded but unattached attachments of the user, and the count of attachments of the transaction would not exceed the limit
func mcpAttachmentCheckNewAttachments(c *core.WebContext, uid int64, field string, attachmentIds []string, existingCount int, currentConfig *settings.Config, services MCPAvailableServices) ([]int64, error) {
	if len(attachmentIds) < 1 {
		return nil, nil
	}

	if err := mcpAttachmentCheckEnabled(currentConfig); err != nil {
		return nil, err
	}

	pictureIds := make([]int64, 0, len(attachmentIds))
	pictureIdSet := make(map[int64]bool, len(attachmentIds))

	for i := 0; i < len(attachmentIds); i++ {
		pictureId, err := parseMCPRequiredId(field, attachmentIds[i])

		if err != nil {
			return nil, err
		}

		if pictureIdSet[pictureId] {
			continue
		}

		pictureIdSet[pictureId] = true
		pictureIds = append(pictureIds, pictureId)
	}

	if count := existingCount + len(pictureIds); count > models.MaximumPicturesCountOfTransaction {
		return nil, NewMCPTooManyItemsError(field, models.MaximumPicturesCountOfTransaction, count)
	}

	pictureInfos, err := services.GetTransactionPictureService().GetNewPictureInfosByPictureIds(c, uid, pictureIds)

	if err != nil {
		log.Errorf(c, "[mcp_transaction_attachment.mcpAttachmentCheckNewAttachments] failed to get transaction attachments for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	notExistsPictureIds := utils.Int64SliceMinus(pictureIds, services.GetTransactionPictureService().GetTransactionPictureIds(pictureInfos))

	if len(notExistsPictureIds) > 0 {
		value := utils.Int64ToString(notExistsPictureIds[0])
		err := NewMCPNotFoundError(field, "unattached attachment", value)
		err.Message = "attachment \"" + value + "\" is not found or is already attached to a transaction, only uploaded but unattached attachments (see list_transaction_attachments) can be attached"
		return nil, err
	}

	return pictureIds, nil
}

// mcpAttachmentCopyTransactionForModify returns a copy of the transaction which can be passed to the transaction service to change only the attachments of the transaction
func mcpAttachmentCopyTransactionForModify(transaction *models.Transaction) *models.Transaction {
	newTransaction := &models.Transaction{
		TransactionId:     transaction.TransactionId,
		Uid:               transaction.Uid,
		Type:              transaction.Type,
		CategoryId:        transaction.CategoryId,
		TransactionTime:   transaction.TransactionTime,
		TimezoneUtcOffset: transaction.TimezoneUtcOffset,
		AccountId:         transaction.AccountId,
		Amount:            transaction.Amount,
		HideAmount:        transaction.HideAmount,
		Comment:           transaction.Comment,
		GeoLongitude:      transaction.GeoLongitude,
		GeoLatitude:       transaction.GeoLatitude,
	}

	if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
		newTransaction.RelatedAccountId = transaction.RelatedAccountId
		newTransaction.RelatedAccountAmount = transaction.RelatedAccountAmount
	}

	return newTransaction
}
