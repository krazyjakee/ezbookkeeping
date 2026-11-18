package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var imageFileExtensionContentTypeMap = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
}

// GetImageContentType returns the content type of specified image file extension or returns empty when the file extension is not image or not supported
func GetImageContentType(fileExtension string) string {
	contentType, exists := imageFileExtensionContentTypeMap[fileExtension]

	if !exists {
		return ""
	}

	return contentType
}

// PdfFileExtension represents the file extension of pdf document
const PdfFileExtension = "pdf"

// PdfFileMagicBytes represents the magic bytes at the beginning of a pdf document
var PdfFileMagicBytes = []byte("%PDF-")

// zipFileMagicBytes represents the magic bytes at the beginning of a zip archive (office open xml and open document files are zip archives)
var zipFileMagicBytes = []byte("PK\x03\x04")

// textFileCheckedHeaderLength represents the length of the beginning of a text document which is checked for binary content
const textFileCheckedHeaderLength = 4096

// isoBaseMediaFileHeaderLength represents the length of the beginning of a heic / heif image which contains the file type box and its major brand
const isoBaseMediaFileHeaderLength = 12

// heifBrands represents the major brands of the file type box of heic / heif images
var heifBrands = map[string]bool{
	"heic": true,
	"heix": true,
	"hevc": true,
	"hevx": true,
	"heim": true,
	"heis": true,
	"hevm": true,
	"hevs": true,
	"mif1": true,
	"msf1": true,
}

// transactionAttachmentContentKind represents how the content of a transaction attachment file type is validated
type transactionAttachmentContentKind byte

const (
	transactionAttachmentContentKindNone transactionAttachmentContentKind = iota
	transactionAttachmentContentKindPdf
	transactionAttachmentContentKindZip
	transactionAttachmentContentKindText
	transactionAttachmentContentKindHeif
)

type documentFileType struct {
	contentType string
	contentKind transactionAttachmentContentKind
}

var documentFileExtensionTypeMap = map[string]documentFileType{
	PdfFileExtension: {contentType: "application/pdf", contentKind: transactionAttachmentContentKindPdf},
	"docx":           {contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", contentKind: transactionAttachmentContentKindZip},
	"xlsx":           {contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contentKind: transactionAttachmentContentKindZip},
	"odt":            {contentType: "application/vnd.oasis.opendocument.text", contentKind: transactionAttachmentContentKindZip},
	"ods":            {contentType: "application/vnd.oasis.opendocument.spreadsheet", contentKind: transactionAttachmentContentKindZip},
	"txt":            {contentType: "text/plain", contentKind: transactionAttachmentContentKindText},
	"csv":            {contentType: "text/csv", contentKind: transactionAttachmentContentKindText},
	"heic":           {contentType: "image/heic", contentKind: transactionAttachmentContentKindHeif},
	"heif":           {contentType: "image/heif", contentKind: transactionAttachmentContentKindHeif},
}

// GetTransactionAttachmentContentType returns the content type of specified transaction attachment file extension (images and documents) or returns empty when the file extension is not supported
func GetTransactionAttachmentContentType(fileExtension string) string {
	contentType := GetImageContentType(fileExtension)

	if contentType != "" {
		return contentType
	}

	fileType, exists := documentFileExtensionTypeMap[fileExtension]

	if !exists {
		return ""
	}

	return fileType.contentType
}

// IsPdfFileExtension returns whether the specified file extension is pdf document
func IsPdfFileExtension(fileExtension string) bool {
	return fileExtension == PdfFileExtension
}

// IsTextDocumentFileExtension returns whether the specified file extension is a plain text document (e.g. txt or csv)
func IsTextDocumentFileExtension(fileExtension string) bool {
	fileType, exists := documentFileExtensionTypeMap[fileExtension]
	return exists && fileType.contentKind == transactionAttachmentContentKindText
}

// IsTransactionAttachmentPreviewable returns whether the specified transaction attachment file extension can be displayed inline by browsers (images and pdf documents), other documents are always downloaded
func IsTransactionAttachmentPreviewable(fileExtension string) bool {
	return GetImageContentType(fileExtension) != "" || IsPdfFileExtension(fileExtension)
}

// GetTransactionAttachmentMagicBytesLength returns the length of the file header which needs to be read for checking the content of specified transaction attachment file extension
func GetTransactionAttachmentMagicBytesLength(fileExtension string) int {
	fileType, exists := documentFileExtensionTypeMap[fileExtension]

	if !exists {
		return 0
	}

	switch fileType.contentKind {
	case transactionAttachmentContentKindPdf:
		return len(PdfFileMagicBytes)
	case transactionAttachmentContentKindZip:
		return len(zipFileMagicBytes)
	case transactionAttachmentContentKindText:
		return textFileCheckedHeaderLength
	case transactionAttachmentContentKindHeif:
		return isoBaseMediaFileHeaderLength
	default:
		return 0
	}
}

// IsTransactionAttachmentContentValid returns whether the file header matches the magic bytes required by specified transaction attachment file extension
func IsTransactionAttachmentContentValid(fileExtension string, fileHeader []byte) bool {
	if GetTransactionAttachmentContentType(fileExtension) == "" {
		return false
	}

	fileType, exists := documentFileExtensionTypeMap[fileExtension]

	if !exists {
		return true
	}

	switch fileType.contentKind {
	case transactionAttachmentContentKindPdf:
		return bytes.HasPrefix(fileHeader, PdfFileMagicBytes)
	case transactionAttachmentContentKindZip:
		return bytes.HasPrefix(fileHeader, zipFileMagicBytes)
	case transactionAttachmentContentKindText:
		return len(fileHeader) > 0 && bytes.IndexByte(fileHeader, 0) < 0
	case transactionAttachmentContentKindHeif:
		return len(fileHeader) >= isoBaseMediaFileHeaderLength && string(fileHeader[4:8]) == "ftyp" && heifBrands[string(fileHeader[8:12])]
	default:
		return true
	}
}

// ListFileNamesWithPrefixAndSuffix returns file name list which has specified prefix and suffix
func ListFileNamesWithPrefixAndSuffix(path string, prefix string, suffix string) []string {
	dir, err := os.Open(path)

	if err != nil {
		return nil
	}

	fileInfos, err := dir.Readdir(0)

	if err != nil {
		return nil
	}

	var fileNames []string

	for i := 0; i < len(fileInfos); i++ {
		fileInfo := fileInfos[i]

		if !fileInfo.IsDir() &&
			strings.HasPrefix(fileInfo.Name(), prefix) &&
			strings.HasSuffix(fileInfo.Name(), suffix) {
			fileNames = append(fileNames, fileInfo.Name())
		}
	}

	return fileNames
}

// IsExists returns whether specified file or directory path exits
func IsExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// WriteFile would write file according to specified content
func WriteFile(path string, data []byte) error {
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	n, err := file.Write(data)

	if err == nil && n < len(data) {
		return io.ErrShortWrite
	}

	return err
}

// GetFileNameWithoutExtension returns the file name without extension
func GetFileNameWithoutExtension(path string) string {
	if path == "" {
		return ""
	}

	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			path = path[i+1:]
			break
		}
	}

	if path == "" {
		return ""
	}

	extension := filepath.Ext(path)

	if len(extension) < 1 {
		return path
	}

	return path[0 : len(path)-len(extension)]
}

// GetFileNameExtension returns the file extension without dot
func GetFileNameExtension(path string) string {
	extension := filepath.Ext(path)

	if len(extension) < 1 || extension[0] != '.' {
		return extension
	}

	return extension[1:]
}
