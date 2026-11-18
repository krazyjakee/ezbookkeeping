package utils

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestZipDocument(t *testing.T, entries map[string]string) []byte {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for name, content := range entries {
		entryWriter, err := writer.Create(name)
		assert.Nil(t, err)
		_, err = entryWriter.Write([]byte(content))
		assert.Nil(t, err)
	}

	assert.Nil(t, writer.Close())
	return buffer.Bytes()
}

func TestCanExtractTextFromDocument(t *testing.T) {
	for _, extension := range []string{"txt", "csv", "docx", "xlsx", "odt", "ods"} {
		assert.True(t, CanExtractTextFromDocument(extension), extension)
	}

	for _, extension := range []string{"pdf", "jpg", "heic", ""} {
		assert.False(t, CanExtractTextFromDocument(extension), extension)
	}
}

func TestExtractTextFromDocument_Text(t *testing.T) {
	text, truncated, err := ExtractTextFromDocument("csv", []byte("\xef\xbb\xbfdate,amount\n2026-10-01,20.00\n"), 0)
	assert.Nil(t, err)
	assert.False(t, truncated)
	assert.Equal(t, "date,amount\n2026-10-01,20.00", text)

	text, _, err = ExtractTextFromDocument("txt", []byte("caf\xe9 \xa320"), 0)
	assert.Nil(t, err)
	assert.Equal(t, "café £20", text)

	text, _, err = ExtractTextFromDocument("txt", []byte{0xff, 0xfe, 'H', 0, 'i', 0}, 0)
	assert.Nil(t, err)
	assert.Equal(t, "Hi", text)

	text, truncated, err = ExtractTextFromDocument("txt", []byte("£12345"), 3)
	assert.Nil(t, err)
	assert.True(t, truncated)
	assert.Equal(t, "£12", text)
}

func TestExtractTextFromDocument_Docx(t *testing.T) {
	data := createTestZipDocument(t, map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml": `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			`<w:p><w:r><w:t>Invoice</w:t></w:r><w:r><w:tab/><w:t xml:space="preserve">INV-001</w:t></w:r></w:p>` +
			`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Total</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>£20.00</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
			`<w:p><w:r><w:instrText>PAGE</w:instrText><w:delText>deleted</w:delText><w:t>End</w:t></w:r></w:p>` +
			`</w:body></w:document>`,
	})

	text, _, err := ExtractTextFromDocument("docx", data, 0)
	assert.Nil(t, err)
	assert.Equal(t, "Invoice\tINV-001\nTotal\t£20.00\nEnd", text)
}

func TestExtractTextFromDocument_Xlsx(t *testing.T) {
	data := createTestZipDocument(t, map[string]string{
		"xl/sharedStrings.xml": `<sst><si><t>Item</t></si><si><t>Amount</t></si><si><r><t>Coff</t></r><r><t>ee</t></r></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData>` +
			`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>` +
			`<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><v>3.5</v></c><c r="C2" t="inlineStr"><is><t>note</t></is></c></row>` +
			`</sheetData></worksheet>`,
		"xl/worksheets/_rels/sheet1.xml.rels": `<Relationships/>`,
	})

	text, _, err := ExtractTextFromDocument("xlsx", data, 0)
	assert.Nil(t, err)
	assert.Equal(t, "Item\tAmount\nCoffee\t3.5\tnote", text)
}

func TestExtractTextFromDocument_OpenDocument(t *testing.T) {
	data := createTestZipDocument(t, map[string]string{
		"mimetype": "application/vnd.oasis.opendocument.text",
		"content.xml": `<office:document-content xmlns:office="o" xmlns:text="t" xmlns:table="tb"><office:body><office:text>` +
			`<text:h>Receipt</text:h><text:p>Shop<text:s/>Ltd<text:line-break/>London</text:p>` +
			`<office:annotation><text:p>hidden comment</text:p></office:annotation>` +
			`<table:table><table:table-row><table:table-cell><text:p>Total</text:p></table:table-cell><table:table-cell><text:p>9.99</text:p></table:table-cell></table:table-row></table:table>` +
			`</office:text></office:body></office:document-content>`,
	})

	text, _, err := ExtractTextFromDocument("odt", data, 0)
	assert.Nil(t, err)
	assert.Equal(t, "Receipt\nShop Ltd\nLondon\nTotal\t9.99", text)

	text, _, err = ExtractTextFromDocument("ods", data, 0)
	assert.Nil(t, err)
	assert.Contains(t, text, "9.99")
}

func TestExtractTextFromDocument_Invalid(t *testing.T) {
	_, _, err := ExtractTextFromDocument("docx", []byte("PK\x03\x04garbage"), 0)
	assert.Equal(t, ErrDocumentTextInvalid, err)

	_, _, err = ExtractTextFromDocument("docx", createTestZipDocument(t, map[string]string{"other.xml": "<a/>"}), 0)
	assert.Equal(t, ErrDocumentTextInvalid, err)

	_, _, err = ExtractTextFromDocument("pdf", []byte("%PDF-1.7"), 0)
	assert.Equal(t, ErrDocumentTextNotSupported, err)
}
