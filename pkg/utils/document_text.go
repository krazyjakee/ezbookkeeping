package utils

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// documentTextMaxZipEntrySize is the maximum decompressed size of one entry of an office document which is read for extracting text
const documentTextMaxZipEntrySize = 16 * 1024 * 1024

// ErrDocumentTextNotSupported indicates that text cannot be extracted from the specified document type
var ErrDocumentTextNotSupported = errors.New("text cannot be extracted from this document type")

// ErrDocumentTextInvalid indicates that the document cannot be parsed
var ErrDocumentTextInvalid = errors.New("document cannot be parsed")

// CanExtractTextFromDocument returns whether the text of the specified transaction attachment file extension can be extracted by ExtractTextFromDocument
func CanExtractTextFromDocument(fileExtension string) bool {
	switch fileExtension {
	case "txt", "csv", "docx", "xlsx", "odt", "ods":
		return true
	default:
		return false
	}
}

// ExtractTextFromDocument returns the plain text of a text document (txt, csv) or an office document (docx, xlsx, odt, ods),
// table cells are separated by tabs and paragraphs / rows by new lines. The returned text is truncated to maxLength characters (if maxLength > 0),
// and the second returned value indicates whether the text has been truncated
func ExtractTextFromDocument(fileExtension string, data []byte, maxLength int) (string, bool, error) {
	var text string
	var err error

	switch fileExtension {
	case "txt", "csv":
		text = decodeTextDocument(data)
	case "docx":
		text, err = extractTextFromZipXmlEntries(data, []string{"word/document.xml"}, docxTextRules)
	case "odt":
		text, err = extractTextFromZipXmlEntries(data, []string{"content.xml"}, openDocumentTextRules)
	case "ods":
		text, err = extractTextFromZipXmlEntries(data, []string{"content.xml"}, openDocumentTextRules)
	case "xlsx":
		text, err = extractTextFromXlsx(data)
	default:
		return "", false, ErrDocumentTextNotSupported
	}

	if err != nil {
		return "", false, err
	}

	text = strings.TrimSpace(text)

	if maxLength > 0 && utf8.RuneCountInString(text) > maxLength {
		runes := []rune(text)
		return string(runes[:maxLength]), true, nil
	}

	return text, false, nil
}

// decodeTextDocument returns the text of a plain text document, UTF-8 (with or without BOM) and UTF-16 (with BOM) are detected, other content is treated as ISO-8859-1
func decodeTextDocument(data []byte) string {
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		data = data[3:]
	} else if bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		littleEndian := data[0] == 0xff
		data = data[2:]
		units := make([]uint16, 0, len(data)/2)

		for i := 0; i+1 < len(data); i += 2 {
			if littleEndian {
				units = append(units, uint16(data[i])|uint16(data[i+1])<<8)
			} else {
				units = append(units, uint16(data[i])<<8|uint16(data[i+1]))
			}
		}

		return string(utf16.Decode(units))
	}

	if utf8.Valid(data) {
		return string(data)
	}

	runes := make([]rune, len(data))

	for i := 0; i < len(data); i++ {
		runes[i] = rune(data[i])
	}

	return string(runes)
}

// xmlTextRules defines how the elements of an office document xml are converted to plain text (by local element name)
type xmlTextRules struct {
	textElements    map[string]bool // only character data inside these elements is collected (all character data if empty)
	newLineAfter    map[string]bool
	tabAfter        map[string]bool
	newLineElements map[string]bool // empty elements which represent a line break
	tabElements     map[string]bool // empty elements which represent a tab
	spaceElements   map[string]bool // empty elements which represent a space
	ignoredElements map[string]bool // character data inside these elements is ignored
}

var docxTextRules = &xmlTextRules{
	textElements:    map[string]bool{"t": true},
	newLineAfter:    map[string]bool{"p": true, "tr": true},
	tabAfter:        map[string]bool{"tc": true},
	newLineElements: map[string]bool{"br": true, "cr": true},
	tabElements:     map[string]bool{"tab": true},
	ignoredElements: map[string]bool{"instrText": true, "delText": true},
}

var openDocumentTextRules = &xmlTextRules{
	newLineAfter:    map[string]bool{"p": true, "h": true, "table-row": true},
	tabAfter:        map[string]bool{"table-cell": true},
	newLineElements: map[string]bool{"line-break": true},
	tabElements:     map[string]bool{"tab": true},
	spaceElements:   map[string]bool{"s": true},
	ignoredElements: map[string]bool{"annotation": true, "tracked-changes": true, "notes": true},
}

// extractTextFromZipXmlEntries returns the text of the specified xml entries of a zip archive (entries ending with "*" match all entries with the prefix)
func extractTextFromZipXmlEntries(data []byte, entryNames []string, rules *xmlTextRules) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))

	if err != nil {
		return "", ErrDocumentTextInvalid
	}

	var builder strings.Builder
	found := false

	for _, entryName := range entryNames {
		content, exists, err := readZipEntry(reader, entryName)

		if err != nil {
			return "", err
		}

		if !exists {
			continue
		}

		found = true

		if err := appendXmlText(content, rules, &builder); err != nil {
			return "", err
		}
	}

	if !found {
		return "", ErrDocumentTextInvalid
	}

	return builder.String(), nil
}

// extractTextFromXlsx returns the text of all worksheets of a xlsx workbook, cells are separated by tabs and rows by new lines
func extractTextFromXlsx(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))

	if err != nil {
		return "", ErrDocumentTextInvalid
	}

	sharedStrings := make([]string, 0)
	sharedStringsContent, exists, err := readZipEntry(reader, "xl/sharedStrings.xml")

	if err != nil {
		return "", err
	}

	if exists {
		sharedStrings, err = parseXlsxSharedStrings(sharedStringsContent)

		if err != nil {
			return "", err
		}
	}

	sheetNames := make([]string, 0)

	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && path.Ext(file.Name) == ".xml" && path.Dir(file.Name) == "xl/worksheets" {
			sheetNames = append(sheetNames, file.Name)
		}
	}

	if len(sheetNames) < 1 {
		return "", ErrDocumentTextInvalid
	}

	sort.Slice(sheetNames, func(i, j int) bool {
		if len(sheetNames[i]) != len(sheetNames[j]) {
			return len(sheetNames[i]) < len(sheetNames[j])
		}

		return sheetNames[i] < sheetNames[j]
	})

	var builder strings.Builder

	for _, sheetName := range sheetNames {
		content, _, err := readZipEntry(reader, sheetName)

		if err != nil {
			return "", err
		}

		if err := appendXlsxSheetText(content, sharedStrings, &builder); err != nil {
			return "", err
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func parseXlsxSharedStrings(content []byte) ([]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	sharedStrings := make([]string, 0)
	var current strings.Builder
	inItem := false
	inText := false

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, ErrDocumentTextInvalid
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "si" {
				inItem = true
				current.Reset()
			} else if element.Name.Local == "t" && inItem {
				inText = true
			}
		case xml.EndElement:
			if element.Name.Local == "si" {
				inItem = false
				sharedStrings = append(sharedStrings, current.String())
			} else if element.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inItem && inText {
				current.Write(element)
			}
		}
	}

	return sharedStrings, nil
}

func appendXlsxSheetText(content []byte, sharedStrings []string, builder *strings.Builder) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	cellType := ""
	inValue := false
	inInlineText := false
	firstCellInRow := true
	var value strings.Builder

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			return nil
		} else if err != nil {
			return ErrDocumentTextInvalid
		}

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "row":
				firstCellInRow = true
			case "c":
				cellType = ""
				value.Reset()

				for _, attr := range element.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
			case "v":
				inValue = true
			case "t":
				inInlineText = cellType == "inlineStr"
			}
		case xml.EndElement:
			switch element.Name.Local {
			case "row":
				builder.WriteString("\n")
			case "c":
				cellText := value.String()

				if cellType == "s" {
					index, err := StringToInt(strings.TrimSpace(cellText))

					if err == nil && index >= 0 && index < len(sharedStrings) {
						cellText = sharedStrings[index]
					}
				}

				if !firstCellInRow {
					builder.WriteString("\t")
				}

				builder.WriteString(cellText)
				firstCellInRow = false
			case "v":
				inValue = false
			case "t":
				inInlineText = false
			}
		case xml.CharData:
			if inValue || inInlineText {
				value.Write(element)
			}
		}
	}
}

// xmlTextSeparatorWeight returns the weight of the separator, a stronger separator replaces a weaker pending separator
func xmlTextSeparatorWeight(separator string) int {
	switch separator {
	case " ":
		return 1
	case "\t":
		return 2
	case "\n":
		return 3
	default:
		return 0
	}
}

func appendXmlText(content []byte, rules *xmlTextRules, builder *strings.Builder) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	textDepth := 0
	ignoredDepth := 0
	cellDepth := 0
	pendingSeparator := ""

	setSeparator := func(separator string) {
		if xmlTextSeparatorWeight(separator) > xmlTextSeparatorWeight(pendingSeparator) {
			pendingSeparator = separator
		}
	}

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			builder.WriteString(pendingSeparator)
			return nil
		} else if err != nil {
			return ErrDocumentTextInvalid
		}

		switch element := token.(type) {
		case xml.StartElement:
			name := element.Name.Local

			if rules.ignoredElements[name] {
				ignoredDepth++
			}

			if rules.textElements[name] {
				textDepth++
			}

			if rules.tabAfter[name] {
				cellDepth++
			}

			if ignoredDepth > 0 {
				continue
			}

			if rules.newLineElements[name] {
				setSeparator("\n")
			} else if rules.tabElements[name] {
				setSeparator("\t")
			} else if rules.spaceElements[name] {
				builder.WriteString(pendingSeparator)
				pendingSeparator = ""
				builder.WriteString(" ")
			}
		case xml.EndElement:
			name := element.Name.Local

			if rules.ignoredElements[name] && ignoredDepth > 0 {
				ignoredDepth--
				continue
			}

			if rules.textElements[name] && textDepth > 0 {
				textDepth--
			}

			if rules.tabAfter[name] && cellDepth > 0 {
				cellDepth--
			}

			if ignoredDepth > 0 {
				continue
			}

			if rules.newLineAfter[name] {
				// paragraphs inside table cells are joined by spaces, so that each table row stays on one line
				if cellDepth > 0 {
					setSeparator(" ")
				} else {
					setSeparator("\n")
				}
			} else if rules.tabAfter[name] {
				setSeparator("\t")
			}
		case xml.CharData:
			if ignoredDepth > 0 {
				continue
			}

			if len(rules.textElements) > 0 && textDepth < 1 {
				continue
			}

			// whitespace only character data containing line breaks is the indentation of the xml, not document text
			if len(rules.textElements) < 1 && len(bytes.TrimSpace(element)) < 1 && bytes.ContainsAny(element, "\r\n") {
				continue
			}

			if builder.Len() > 0 {
				builder.WriteString(pendingSeparator)
			}

			pendingSeparator = ""
			builder.Write(element)
		}
	}
}

// readZipEntry returns the decompressed content of the specified entry of the zip archive, returns false if the entry does not exist
func readZipEntry(reader *zip.Reader, entryName string) ([]byte, bool, error) {
	for _, file := range reader.File {
		if file.Name != entryName {
			continue
		}

		entryReader, err := file.Open()

		if err != nil {
			return nil, true, ErrDocumentTextInvalid
		}

		defer entryReader.Close()

		content, err := io.ReadAll(io.LimitReader(entryReader, documentTextMaxZipEntrySize+1))

		if err != nil {
			return nil, true, ErrDocumentTextInvalid
		}

		if len(content) > documentTextMaxZipEntrySize {
			return nil, true, ErrDocumentTextInvalid
		}

		return content, true, nil
	}

	return nil, false, nil
}
