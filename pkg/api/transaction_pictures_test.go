package api

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func createTransactionPictureTestMultipartFile(t *testing.T, fileName string, content []byte) multipart.File {
	body, contentType := createTransactionPictureTestMultipartBody(t, fileName, content)

	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Type", contentType)

	err := request.ParseMultipartForm(1 << 20)
	assert.Nil(t, err)

	file, _, err := request.FormFile("picture")
	assert.Nil(t, err)

	return file
}

func createTransactionPictureTestMultipartBody(t *testing.T, fileName string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("picture", fileName)
	assert.Nil(t, err)

	_, err = part.Write(content)
	assert.Nil(t, err)

	err = writer.Close()
	assert.Nil(t, err)

	return body, writer.FormDataContentType()
}

func createTransactionPictureTestWebContext(request *http.Request) (*core.WebContext, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request

	return core.WrapWebContext(ginContext, nil), recorder
}

func TestCheckTransactionPictureFileContent_ValidPdf(t *testing.T) {
	content := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
	file := createTransactionPictureTestMultipartFile(t, "invoice.pdf", content)
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "pdf")
	assert.Nil(t, err)

	// the file must be rewound so that the whole content is saved
	actualContent, err := io.ReadAll(file)
	assert.Nil(t, err)
	assert.Equal(t, content, actualContent)
}

func TestCheckTransactionPictureFileContent_RenamedNonPdf(t *testing.T) {
	file := createTransactionPictureTestMultipartFile(t, "invoice.pdf", []byte("<html><script>alert(1)</script></html>"))
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "pdf")
	assert.Equal(t, errs.ErrTransactionPictureContentInvalid, err)
}

func TestCheckTransactionPictureFileContent_RenamedImageAsPdf(t *testing.T) {
	file := createTransactionPictureTestMultipartFile(t, "image.pdf", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "pdf")
	assert.Equal(t, errs.ErrTransactionPictureContentInvalid, err)
}

func TestCheckTransactionPictureFileContent_TooShortPdf(t *testing.T) {
	file := createTransactionPictureTestMultipartFile(t, "short.pdf", []byte("%PD"))
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "pdf")
	assert.Equal(t, errs.ErrTransactionPictureContentInvalid, err)
}

func TestCheckTransactionPictureFileContent_Image(t *testing.T) {
	content := []byte{0xff, 0xd8, 0xff, 0xe0}
	file := createTransactionPictureTestMultipartFile(t, "image.jpg", content)
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "jpg")
	assert.Nil(t, err)

	actualContent, err := io.ReadAll(file)
	assert.Nil(t, err)
	assert.Equal(t, content, actualContent)
}

func TestCheckTransactionPictureFileContent_UnsupportedExtension(t *testing.T) {
	file := createTransactionPictureTestMultipartFile(t, "page.html", []byte("%PDF-1.7"))
	defer file.Close()

	err := checkTransactionPictureFileContent(file, "html")
	assert.Equal(t, errs.ErrTransactionPictureTypeNotSupported, err)
}

func TestSetTransactionPictureResponseHeaders_Image(t *testing.T) {
	c, recorder := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123.png", nil))

	setTransactionPictureResponseHeaders(c, 123, "png", false)

	assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "", recorder.Header().Get("Content-Disposition"))
	assert.Equal(t, "", recorder.Header().Get("Content-Security-Policy"))
}

func TestSetTransactionPictureResponseHeaders_PdfInline(t *testing.T) {
	c, recorder := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123.pdf", nil))

	setTransactionPictureResponseHeaders(c, 123, "pdf", false)

	assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "inline; filename=\"123.pdf\"", recorder.Header().Get("Content-Disposition"))
	assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'")
}

func TestSetTransactionPictureResponseHeaders_PdfDownload(t *testing.T) {
	c, recorder := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123.pdf?download=true", nil))

	setTransactionPictureResponseHeaders(c, 123, "pdf", true)

	assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "attachment; filename=\"123.pdf\"", recorder.Header().Get("Content-Disposition"))
	assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "default-src 'none'")
}

func TestSetTransactionPictureResponseHeaders_PdfIsNotSandboxed(t *testing.T) {
	c, recorder := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123.pdf", nil))

	setTransactionPictureResponseHeaders(c, 123, "pdf", false)

	assert.NotContains(t, recorder.Header().Get("Content-Security-Policy"), "sandbox")
}

func TestSetTransactionPictureResponseHeaders_DocumentsAlwaysDownloaded(t *testing.T) {
	for _, extension := range []string{"docx", "xlsx", "odt", "ods", "txt", "csv", "heic", "heif"} {
		c, recorder := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123."+extension, nil))

		setTransactionPictureResponseHeaders(c, 123, extension, false)

		assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"), extension)
		assert.Equal(t, "attachment; filename=\"123."+extension+"\"", recorder.Header().Get("Content-Disposition"), extension)
		assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "default-src 'none'", extension)
		assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "sandbox", extension)
	}
}

func TestCheckTransactionPictureFileContent_Documents(t *testing.T) {
	validFiles := map[string][]byte{
		"docx": []byte("PK\x03\x04\x14\x00\x06\x00"),
		"ods":  []byte("PK\x03\x04\x14\x00\x00\x08"),
		"csv":  []byte("date,amount\n2026-10-01,20.00\n"),
		"txt":  []byte("x"),
		"heic": []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic"),
	}

	for extension, content := range validFiles {
		file := createTransactionPictureTestMultipartFile(t, "file."+extension, content)
		assert.Nil(t, checkTransactionPictureFileContent(file, extension), extension)
		file.Close()
	}

	invalidFiles := map[string][]byte{
		"docx": []byte("%PDF-1.7"),
		"csv":  []byte("a,b\x00\x00"),
		"heic": []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01},
	}

	for extension, content := range invalidFiles {
		file := createTransactionPictureTestMultipartFile(t, "file."+extension, content)
		assert.Equal(t, errs.ErrTransactionPictureContentInvalid, checkTransactionPictureFileContent(file, extension), extension)
		file.Close()
	}
}

func TestTransactionPicturesApi_TransactionPictureGetHandler_UnsupportedType(t *testing.T) {
	api := &TransactionPicturesApi{}
	c, _ := createTransactionPictureTestWebContext(httptest.NewRequest(http.MethodGet, "/pictures/123.html", nil))
	c.Params = gin.Params{{Key: "fileName", Value: "123.html"}}

	data, contentType, err := api.TransactionPictureGetHandler(c)

	assert.Nil(t, data)
	assert.Equal(t, "", contentType)
	assert.Equal(t, errs.ErrTransactionPictureTypeNotSupported, err)
}

func TestTransactionPicturesApi_TransactionPictureUploadHandler_UnsupportedType(t *testing.T) {
	api := &TransactionPicturesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.NewConfigContainer(&settings.Config{
				MaxTransactionPictureFileSize: 1 << 20,
			}),
		},
	}

	body, contentType := createTransactionPictureTestMultipartBody(t, "setup.exe", []byte("MZ\x90\x00"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/transaction/pictures/upload.json", body)
	request.Header.Set("Content-Type", contentType)
	c, _ := createTransactionPictureTestWebContext(request)

	result, err := api.TransactionPictureUploadHandler(c)

	assert.Nil(t, result)
	assert.Equal(t, errs.ErrTransactionPictureTypeNotSupported, err)
}

func TestTransactionPicturesApi_TransactionPictureUploadHandler_RenamedNonPdf(t *testing.T) {
	api := &TransactionPicturesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.NewConfigContainer(&settings.Config{
				MaxTransactionPictureFileSize: 1 << 20,
			}),
		},
	}

	body, contentType := createTransactionPictureTestMultipartBody(t, "invoice.pdf", []byte("<html><body>not a pdf</body></html>"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/transaction/pictures/upload.json", body)
	request.Header.Set("Content-Type", contentType)
	c, _ := createTransactionPictureTestWebContext(request)

	result, err := api.TransactionPictureUploadHandler(c)

	assert.Nil(t, result)
	assert.Equal(t, errs.ErrTransactionPictureContentInvalid, err)
}

func TestTransactionPicturesApi_TransactionPictureUploadHandler_ExceedMaxSize(t *testing.T) {
	api := &TransactionPicturesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.NewConfigContainer(&settings.Config{
				MaxTransactionPictureFileSize: 8,
			}),
		},
	}

	body, contentType := createTransactionPictureTestMultipartBody(t, "invoice.pdf", []byte("%PDF-1.7\n%%EOF\n"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/transaction/pictures/upload.json", body)
	request.Header.Set("Content-Type", contentType)
	c, _ := createTransactionPictureTestWebContext(request)

	result, err := api.TransactionPictureUploadHandler(c)

	assert.Nil(t, result)
	assert.Equal(t, errs.ErrExceedMaxTransactionPictureFileSize, err)
}
