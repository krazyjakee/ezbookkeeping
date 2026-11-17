package models

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const TransactionPictureNewPictureTransactionId = int64(0)

// TransactionPictureOriginalFileNameMaxLength is the maximum length (in characters) of the stored original file name of a transaction picture
const TransactionPictureOriginalFileNameMaxLength = 255

// transactionPictureOriginalFileNameMaxExtensionLength is the maximum length (in characters) of the file extension which is kept when the original file name is truncated
const transactionPictureOriginalFileNameMaxExtensionLength = 16

// TransactionPictureInfo represents transaction picture file info stored in database
type TransactionPictureInfo struct {
	Uid              int64  `xorm:"INDEX(IDX_transaction_picture_uid_deleted_transaction_id_picture_id) INDEX(IDX_transaction_picture_uid_deleted_picture_id) NOT NULL"`
	Deleted          bool   `xorm:"INDEX(IDX_transaction_picture_uid_deleted_transaction_id_picture_id) INDEX(IDX_transaction_picture_uid_deleted_picture_id) NOT NULL"`
	TransactionId    int64  `xorm:"INDEX(IDX_transaction_picture_uid_deleted_transaction_id_picture_id) NOT NULL"`
	PictureId        int64  `xorm:"PK INDEX(IDX_transaction_picture_uid_deleted_transaction_id_picture_id) INDEX(IDX_transaction_picture_uid_deleted_picture_id)"`
	PictureExtension string `xorm:"VARCHAR(10) NOT NULL"`
	CreatedIp        string `xorm:"VARCHAR(39)"`
	OriginalFileName string `xorm:"VARCHAR(255) NOT NULL DEFAULT ''"`
	FileSize         int64  `xorm:"NOT NULL DEFAULT 0"`
	FileSha256       string `xorm:"VARCHAR(64) NOT NULL DEFAULT ''"`
	CreatedUnixTime  int64
	UpdatedUnixTime  int64
	DeletedUnixTime  int64
}

// TransactionPictureUnusedDeleteRequest represents all parameters of unused transaction picture deleting request
type TransactionPictureUnusedDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// TransactionPictureInfoBasicResponse represents a view-object of transaction picture basic info
type TransactionPictureInfoBasicResponse struct {
	PictureId   int64  `json:"pictureId,string"`
	OriginalUrl string `json:"originalUrl"`
}

// ToTransactionPictureInfoBasicResponse returns a view-object according to database model
func (p *TransactionPictureInfo) ToTransactionPictureInfoBasicResponse(originalUrl string, pictureToken string) *TransactionPictureInfoBasicResponse {
	if pictureToken != "" {
		separator := "?"

		if strings.Contains(originalUrl, "?") {
			separator = "&"
		}

		originalUrl += separator + "token=" + url.QueryEscape(pictureToken)
	}

	return &TransactionPictureInfoBasicResponse{
		PictureId:   p.PictureId,
		OriginalUrl: originalUrl,
	}
}

// TransactionPictureInfoBasicResponseSlice represents the slice data structure of TransactionPictureInfoBasicResponse
type TransactionPictureInfoBasicResponseSlice []*TransactionPictureInfoBasicResponse

// Len returns the count of items
func (s TransactionPictureInfoBasicResponseSlice) Len() int {
	return len(s)
}

// Swap swaps two items
func (s TransactionPictureInfoBasicResponseSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the first item is less than the second one
func (s TransactionPictureInfoBasicResponseSlice) Less(i, j int) bool {
	return s[i].PictureId < s[j].PictureId
}

// SanitizeTransactionPictureOriginalFileName returns the original file name which is safe to store and display:
// directory components are removed, control and format characters (e.g. bidirectional overrides) are removed,
// invalid UTF-8 sequences are dropped, and the name is truncated to TransactionPictureOriginalFileNameMaxLength characters (keeping a short file extension).
// Returns empty string if nothing remains
func SanitizeTransactionPictureOriginalFileName(fileName string) string {
	fileName = strings.ToValidUTF8(fileName, "")

	if index := strings.LastIndexAny(fileName, "/\\"); index >= 0 {
		fileName = fileName[index+1:]
	}

	fileName = strings.Map(func(r rune) rune {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			return -1
		}

		return r
	}, fileName)

	fileName = strings.TrimSpace(fileName)

	if fileName == "." || fileName == ".." {
		return ""
	}

	runes := []rune(fileName)

	if len(runes) <= TransactionPictureOriginalFileNameMaxLength {
		return fileName
	}

	extensionRunes := make([]rune, 0)

	if index := strings.LastIndex(fileName, "."); index > 0 {
		extension := []rune(fileName[index:])

		if len(extension) <= transactionPictureOriginalFileNameMaxExtensionLength {
			extensionRunes = extension
		}
	}

	baseRunes := runes[:len(runes)-len(extensionRunes)]
	maxBaseLength := TransactionPictureOriginalFileNameMaxLength - len(extensionRunes)

	if len(baseRunes) > maxBaseLength {
		baseRunes = baseRunes[:maxBaseLength]
	}

	return strings.TrimSpace(string(baseRunes)) + string(extensionRunes)
}
