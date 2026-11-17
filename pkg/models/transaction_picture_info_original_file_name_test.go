package models

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeTransactionPictureOriginalFileName_KeepNormalName(t *testing.T) {
	assert.Equal(t, "receipt 2024-01-02.pdf", SanitizeTransactionPictureOriginalFileName("receipt 2024-01-02.pdf"))
	assert.Equal(t, "收据.jpg", SanitizeTransactionPictureOriginalFileName("收据.jpg"))
}

func TestSanitizeTransactionPictureOriginalFileName_RemoveDirectories(t *testing.T) {
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("/home/user/receipt.pdf"))
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("C:\\Users\\user\\receipt.pdf"))
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("../../receipt.pdf"))
	assert.Equal(t, "", SanitizeTransactionPictureOriginalFileName("../.."))
	assert.Equal(t, "", SanitizeTransactionPictureOriginalFileName("/tmp/"))
	assert.Equal(t, "", SanitizeTransactionPictureOriginalFileName("."))
}

func TestSanitizeTransactionPictureOriginalFileName_RemoveControlAndFormatCharacters(t *testing.T) {
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("rec\x00ei\npt.pdf\r"))
	assert.Equal(t, "receiptfdp.exe", SanitizeTransactionPictureOriginalFileName("receipt\u202efdp.exe"))
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("  receipt.pdf\t "))
	assert.Equal(t, "receipt.pdf", SanitizeTransactionPictureOriginalFileName("rec\xffeipt.pdf"))
	assert.Equal(t, "", SanitizeTransactionPictureOriginalFileName("\x01\x02"))
}

func TestSanitizeTransactionPictureOriginalFileName_TruncateKeepExtension(t *testing.T) {
	name := strings.Repeat("中", 300) + ".pdf"
	actualValue := SanitizeTransactionPictureOriginalFileName(name)
	assert.Equal(t, TransactionPictureOriginalFileNameMaxLength, utf8.RuneCountInString(actualValue))
	assert.True(t, strings.HasSuffix(actualValue, ".pdf"))
	assert.True(t, utf8.ValidString(actualValue))

	name = strings.Repeat("a", 300)
	actualValue = SanitizeTransactionPictureOriginalFileName(name)
	assert.Equal(t, TransactionPictureOriginalFileNameMaxLength, utf8.RuneCountInString(actualValue))

	name = "a." + strings.Repeat("b", 300)
	actualValue = SanitizeTransactionPictureOriginalFileName(name)
	assert.Equal(t, TransactionPictureOriginalFileNameMaxLength, utf8.RuneCountInString(actualValue))
	assert.True(t, strings.HasPrefix(actualValue, "a.b"))
}
