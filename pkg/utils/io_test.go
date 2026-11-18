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
