package donut

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/donut"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
)

const testSystemPrompt = `### Expense categories:
Groceries
Fuel

### Account names:
Current Account

### Current time:
2026-11-18 11:00:00
`

// fakeWorkerScript answers every request with the output of the two models for a fuel receipt, the "receipts" model misreads the total
const fakeWorkerScript = `#!/bin/sh
echo '{"id":0,"ready":true}'
while IFS= read -r line; do
  id=$(echo "$line" | sed 's/.*"id":\([0-9]*\).*/\1/')
  echo "{\"id\":$id,\"results\":[{\"model\":\"receipts\",\"sequence\":\"<s_store_name> SHELLWATFORD</s_store_name><s_date> 03/11/2016</s_date><s_time> 17:02:11</s_time><s_total> £2.83</s_total>\"},{\"model\":\"sroie\",\"sequence\":\"<s_total> 60.15</s_total><s_date> 03/11/2016</s_date><s_company> SHELL WATFORD HEMPSTEAD Rd</s_company>\"}]}"
done
`

func newTestProvider(t *testing.T, ready bool, serverURL string) *DonutLargeLanguageModelProvider {
	dir := t.TempDir()
	content := []byte("{}")
	checksum := sha256.Sum256(content)
	manifest := []donut.ModelManifest{{Name: "test", Files: []donut.ModelFile{{Path: "test/donut.json", Size: 2, SHA256: hex.EncodeToString(checksum[:])}}}}
	store := donut.NewModelStore(dir, serverURL, http.DefaultClient, manifest, nil)

	if ready {
		os.MkdirAll(filepath.Join(dir, "test"), 0o755)
		os.WriteFile(filepath.Join(dir, "test/donut.json"), content, 0o644)
		os.WriteFile(filepath.Join(dir, ".verified"), []byte(manifest[0].Files[0].SHA256+"  test/donut.json\n"), 0o644)
	}

	workerPath := filepath.Join(t.TempDir(), "fake-worker.sh")
	os.WriteFile(workerPath, []byte(fakeWorkerScript), 0o755)
	worker := donut.NewWorker(workerPath, "lib", 1, false, time.Minute, nil)
	t.Cleanup(worker.Stop)

	return &DonutLargeLanguageModelProvider{
		runtime:   &runtime{store: store, worker: worker},
		dateOrder: donut.DateOrderDMY,
		timeout:   10 * time.Second,
	}
}

func decodeResult(t *testing.T, response *data.LargeLanguageModelTextualResponse) map[string]any {
	result := map[string]any{}
	assert.Nil(t, json.Unmarshal([]byte(response.Content), &result))
	return result
}

func TestDonutProvider_Text(t *testing.T) {
	provider := newTestProvider(t, false, "http://127.0.0.1:1")
	response, err := provider.GetJsonResponse(nil, 1, nil, &data.LargeLanguageModelRequest{
		SystemPrompt:   testSystemPrompt,
		UserPrompt:     []byte("The following is the text content of the document \"a.docx\" (application/x):\n\nTesco Express\nDate: 01/10/2026\nTotal £20.00"),
		UserPromptType: data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_TEXT,
	})

	assert.Nil(t, err)
	assert.Equal(t, map[string]any{
		"type": "expense", "time": "2026-10-01 12:00:00", "amount": "20.00", "currency": "GBP", "supplier": "Tesco Express",
		"description": "Tesco Express", "category": "Groceries", "account": "Current Account",
	}, decodeResult(t, response))
}

func TestDonutProvider_ImageMergesModels(t *testing.T) {
	provider := newTestProvider(t, true, "http://127.0.0.1:1")
	response, err := provider.GetJsonResponse(nil, 1, nil, &data.LargeLanguageModelRequest{
		SystemPrompt:   testSystemPrompt,
		UserPrompt:     []byte("image bytes"),
		UserPromptType: data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
	})

	assert.Nil(t, err)
	assert.Equal(t, map[string]any{
		"type": "expense", "time": "2026-11-03 17:02:11", "amount": "60.15", "currency": "GBP", "supplier": "SHELL WATFORD",
		"description": "SHELL WATFORD", "category": "Fuel", "account": "Current Account",
	}, decodeResult(t, response))
}

func TestDonutProvider_ImageWhileModelsDownload(t *testing.T) {
	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	defer server.Close()
	defer close(blocked)

	provider := newTestProvider(t, false, server.URL)
	_, err := provider.GetJsonResponse(nil, 1, nil, &data.LargeLanguageModelRequest{
		SystemPrompt:   testSystemPrompt,
		UserPrompt:     []byte("image bytes"),
		UserPromptType: data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
	})

	customError, ok := err.(*errs.Error)
	assert.True(t, ok)
	assert.Equal(t, errs.ErrDonutModelDownloading.Code(), customError.Code())
	assert.Contains(t, customError.Message, "being downloaded")
}

func TestDonutProvider_RejectsNonPdfDocuments(t *testing.T) {
	provider := newTestProvider(t, false, "http://127.0.0.1:1")
	_, err := provider.GetJsonResponse(nil, 1, nil, &data.LargeLanguageModelRequest{
		UserPrompt:            []byte("PK\x03\x04"),
		UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_DOCUMENT,
		UserPromptContentType: "application/zip",
	})

	assert.Equal(t, errs.ErrLargeLanguageModelDocumentNotSupported, err)
}
