package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetImageContentType(t *testing.T) {
	fileName := "gif"
	expectedContentType := "image/gif"
	actualContentType := GetImageContentType(fileName)
	assert.Equal(t, expectedContentType, actualContentType)

	fileName = "bmp"
	expectedContentType = ""
	actualContentType = GetImageContentType(fileName)
	assert.Equal(t, expectedContentType, actualContentType)
}

func TestGetImageContentType_Pdf(t *testing.T) {
	actualContentType := GetImageContentType("pdf")
	assert.Equal(t, "", actualContentType)
}

func TestGetTransactionAttachmentContentType(t *testing.T) {
	assert.Equal(t, "image/jpeg", GetTransactionAttachmentContentType("jpg"))
	assert.Equal(t, "image/jpeg", GetTransactionAttachmentContentType("jpeg"))
	assert.Equal(t, "image/png", GetTransactionAttachmentContentType("png"))
	assert.Equal(t, "image/gif", GetTransactionAttachmentContentType("gif"))
	assert.Equal(t, "image/webp", GetTransactionAttachmentContentType("webp"))
	assert.Equal(t, "application/pdf", GetTransactionAttachmentContentType("pdf"))
	assert.Equal(t, "", GetTransactionAttachmentContentType("bmp"))
	assert.Equal(t, "", GetTransactionAttachmentContentType("html"))
	assert.Equal(t, "", GetTransactionAttachmentContentType("svg"))
	assert.Equal(t, "", GetTransactionAttachmentContentType(""))
}

func TestIsPdfFileExtension(t *testing.T) {
	assert.True(t, IsPdfFileExtension("pdf"))
	assert.False(t, IsPdfFileExtension("png"))
	assert.False(t, IsPdfFileExtension(""))
}

func TestGetTransactionAttachmentMagicBytesLength(t *testing.T) {
	assert.Equal(t, 5, GetTransactionAttachmentMagicBytesLength("pdf"))
	assert.Equal(t, 0, GetTransactionAttachmentMagicBytesLength("jpg"))
	assert.Equal(t, 0, GetTransactionAttachmentMagicBytesLength("bmp"))
}

func TestIsTransactionAttachmentContentValid_Pdf(t *testing.T) {
	assert.True(t, IsTransactionAttachmentContentValid("pdf", []byte("%PDF-")))
	assert.True(t, IsTransactionAttachmentContentValid("pdf", []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3")))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", []byte("%PDF")))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", []byte("<html><body>%PDF-1.7</body></html>")))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", []byte(" %PDF-1.7")))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", []byte{}))
	assert.False(t, IsTransactionAttachmentContentValid("pdf", nil))
}

func TestIsTransactionAttachmentContentValid_Image(t *testing.T) {
	assert.True(t, IsTransactionAttachmentContentValid("jpg", []byte{0xff, 0xd8, 0xff}))
	assert.True(t, IsTransactionAttachmentContentValid("png", nil))
}

func TestIsTransactionAttachmentContentValid_UnsupportedExtension(t *testing.T) {
	assert.False(t, IsTransactionAttachmentContentValid("html", []byte("%PDF-1.7")))
	assert.False(t, IsTransactionAttachmentContentValid("", []byte("%PDF-1.7")))
}

func TestGetTransactionAttachmentContentType_Documents(t *testing.T) {
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", GetTransactionAttachmentContentType("docx"))
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", GetTransactionAttachmentContentType("xlsx"))
	assert.Equal(t, "application/vnd.oasis.opendocument.text", GetTransactionAttachmentContentType("odt"))
	assert.Equal(t, "application/vnd.oasis.opendocument.spreadsheet", GetTransactionAttachmentContentType("ods"))
	assert.Equal(t, "text/plain", GetTransactionAttachmentContentType("txt"))
	assert.Equal(t, "text/csv", GetTransactionAttachmentContentType("csv"))
	assert.Equal(t, "image/heic", GetTransactionAttachmentContentType("heic"))
	assert.Equal(t, "image/heif", GetTransactionAttachmentContentType("heif"))
	assert.Equal(t, "", GetTransactionAttachmentContentType("doc"))
	assert.Equal(t, "", GetTransactionAttachmentContentType("htm"))
	assert.Equal(t, "", GetImageContentType("heic"))
}

func TestIsTransactionAttachmentPreviewable(t *testing.T) {
	assert.True(t, IsTransactionAttachmentPreviewable("jpg"))
	assert.True(t, IsTransactionAttachmentPreviewable("pdf"))
	assert.False(t, IsTransactionAttachmentPreviewable("docx"))
	assert.False(t, IsTransactionAttachmentPreviewable("txt"))
	assert.False(t, IsTransactionAttachmentPreviewable("heic"))
}

func TestIsTextDocumentFileExtension(t *testing.T) {
	assert.True(t, IsTextDocumentFileExtension("txt"))
	assert.True(t, IsTextDocumentFileExtension("csv"))
	assert.False(t, IsTextDocumentFileExtension("pdf"))
	assert.False(t, IsTextDocumentFileExtension("jpg"))
}

func TestGetTransactionAttachmentMagicBytesLength_Documents(t *testing.T) {
	assert.Equal(t, 4, GetTransactionAttachmentMagicBytesLength("docx"))
	assert.Equal(t, 4, GetTransactionAttachmentMagicBytesLength("ods"))
	assert.Equal(t, 4096, GetTransactionAttachmentMagicBytesLength("csv"))
	assert.Equal(t, 12, GetTransactionAttachmentMagicBytesLength("heic"))
}

func TestIsTransactionAttachmentContentValid_Zip(t *testing.T) {
	for _, extension := range []string{"docx", "xlsx", "odt", "ods"} {
		assert.True(t, IsTransactionAttachmentContentValid(extension, []byte("PK\x03\x04\x14\x00")))
		assert.False(t, IsTransactionAttachmentContentValid(extension, []byte("PK\x05\x06")))
		assert.False(t, IsTransactionAttachmentContentValid(extension, []byte("%PDF-1.7")))
		assert.False(t, IsTransactionAttachmentContentValid(extension, nil))
	}
}

func TestIsTransactionAttachmentContentValid_Text(t *testing.T) {
	assert.True(t, IsTransactionAttachmentContentValid("csv", []byte("date,amount\n2026-01-01,1.00\n")))
	assert.True(t, IsTransactionAttachmentContentValid("txt", []byte("Receipt \xc2\xa3 20.00")))
	assert.True(t, IsTransactionAttachmentContentValid("txt", []byte("caf\xe9")))
	assert.False(t, IsTransactionAttachmentContentValid("txt", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00}))
	assert.False(t, IsTransactionAttachmentContentValid("csv", []byte{}))
}

func TestIsTransactionAttachmentContentValid_Heif(t *testing.T) {
	assert.True(t, IsTransactionAttachmentContentValid("heic", []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00")))
	assert.True(t, IsTransactionAttachmentContentValid("heif", []byte("\x00\x00\x00\x18ftypmif1")))
	assert.False(t, IsTransactionAttachmentContentValid("heic", []byte("\x00\x00\x00\x18ftypisom")))
	assert.False(t, IsTransactionAttachmentContentValid("heic", []byte("\x00\x00\x00\x18ftyp")))
	assert.False(t, IsTransactionAttachmentContentValid("heic", []byte{0xff, 0xd8, 0xff}))
}

func TestGetFileNameWithoutExtension(t *testing.T) {
	fileName := "name.ext"
	expectedName := "name"
	actualName := GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "C:\\name.ext"
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "/root/name.ext"
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "name"
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "C:\\name"
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "/root/name"
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "name."
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "C:\\name."
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "/root/name."
	expectedName = "name"
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = ".ext"
	expectedName = ""
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "C:\\.ext"
	expectedName = ""
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)

	fileName = "/root/.ext"
	expectedName = ""
	actualName = GetFileNameWithoutExtension(fileName)
	assert.Equal(t, expectedName, actualName)
}

func TestGetFileNameExtension(t *testing.T) {
	fileName := "name.ext"
	expectedExt := "ext"
	actualExt := GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "C:\\name.ext"
	expectedExt = "ext"
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "/root/name.ext"
	expectedExt = "ext"
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "name"
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "C:\\name"
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "/root/name"
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "name."
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "C:\\name."
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "/root/name."
	expectedExt = ""
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = ".ext"
	expectedExt = "ext"
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "C:\\.ext"
	expectedExt = "ext"
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)

	fileName = "/root/.ext"
	expectedExt = "ext"
	actualExt = GetFileNameExtension(fileName)
	assert.Equal(t, expectedExt, actualExt)
}
