// Package donut implements the built-in "donut" recognition provider, which reads receipts and invoices with Donut models
// running in a local worker process instead of calling a large language model API
package donut

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/donut"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/httpclient"
	"github.com/mayswind/ezbookkeeping/pkg/llm/data"
	"github.com/mayswind/ezbookkeeping/pkg/llm/provider"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// WorkerBinaryName is the file name of the worker binary, which is looked up next to the ezBookkeeping binary by default
const WorkerBinaryName = "ezbookkeeping-donut"

// pdfMinimumTextLength is the minimum length of the text layer of a PDF document for it to be read (shorter text layers are noise of scanned documents)
const pdfMinimumTextLength = 40

// pdfMaxTextPages is the number of pages of a PDF document whose text layer is read
const pdfMaxTextPages = 3

var documentTextPrefixPattern = regexp.MustCompile(`(?s)^The following is the text content of the document .*?:\n\n`)

// runtime is shared by all providers using the same model directory, so the models are downloaded and loaded only once
type runtime struct {
	store      *donut.ModelStore
	worker     *donut.Worker
	workerPath string
}

var (
	runtimesMutex sync.Mutex
	runtimes      = map[string]*runtime{}
)

// DonutLargeLanguageModelProvider reads documents with the built-in Donut models
type DonutLargeLanguageModelProvider struct {
	runtime   *runtime
	dateOrder donut.DateOrder
	account   string
	timeout   time.Duration
}

// NewDonutLargeLanguageModelProvider creates a new Donut provider instance
func NewDonutLargeLanguageModelProvider(llmConfig *settings.LLMConfig) provider.LargeLanguageModelProvider {
	return &DonutLargeLanguageModelProvider{
		runtime:   getRuntime(llmConfig),
		dateOrder: donut.DateOrder(llmConfig.DonutDateOrder),
		account:   llmConfig.DonutDefaultAccount,
		timeout:   time.Duration(llmConfig.LargeLanguageModelAPIRequestTimeout) * time.Millisecond,
	}
}

func getRuntime(llmConfig *settings.LLMConfig) *runtime {
	runtimesMutex.Lock()
	defer runtimesMutex.Unlock()

	if existing, exists := runtimes[llmConfig.DonutModelPath]; exists {
		return existing
	}

	logf := func(format string, args ...any) {
		log.Infof(nil, "[donut_large_language_model_provider] "+format, args...)
	}

	// the models are large, so the download is only bounded by the timeout of each file in the model store
	client := httpclient.NewHttpClient(0, llmConfig.LargeLanguageModelAPIProxy, llmConfig.LargeLanguageModelAPISkipTLSVerify, core.GetOutgoingUserAgent(), false, []uint16{})
	workerPath := ResolveWorkerPath(llmConfig)

	created := &runtime{
		workerPath: workerPath,
		store:      donut.NewModelStore(llmConfig.DonutModelPath, llmConfig.DonutModelDownloadURL, client, donut.DefaultModels, logf),
		worker:     donut.NewWorker(workerPath, llmConfig.DonutOnnxRuntimeLibraryPath, int(llmConfig.DonutThreads), llmConfig.DonutLowMemory, time.Duration(llmConfig.DonutWorkerIdleTimeout)*time.Second, logf),
	}

	runtimes[llmConfig.DonutModelPath] = created
	return created
}

// ResolveWorkerPath returns the path of the worker binary, "ezbookkeeping-donut" next to the ezBookkeeping binary unless configured
func ResolveWorkerPath(llmConfig *settings.LLMConfig) string {
	if llmConfig.DonutWorkerPath != "" {
		return llmConfig.DonutWorkerPath
	}

	if executable, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(executable), WorkerBinaryName)
	}

	return WorkerBinaryName
}

// IsWorkerAvailable returns whether the worker binary exists and is executable (it is included in the Docker image, but not in the release binaries)
func IsWorkerAvailable(llmConfig *settings.LLMConfig) (string, bool) {
	path := ResolveWorkerPath(llmConfig)
	info, err := os.Stat(path)

	if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return path, false
	}

	return path, true
}

// Preload starts downloading the models in the background if they are not downloaded yet, so they are ready when the first document is read
func (p *DonutLargeLanguageModelProvider) Preload() {
	if err := p.runtime.store.EnsureReady(); err != nil {
		log.Infof(nil, "[donut_large_language_model_provider.Preload] %s", err.Error())
	} else {
		log.Infof(nil, "[donut_large_language_model_provider.Preload] the donut models are ready")
	}
}

// GetJsonResponse returns the transaction recognized from the image, PDF document or document text in the json schema of the recognition prompts
func (p *DonutLargeLanguageModelProvider) GetJsonResponse(c core.Context, uid int64, currentLLMConfig *settings.LLMConfig, request *data.LargeLanguageModelRequest) (*data.LargeLanguageModelTextualResponse, error) {
	options := donut.ParsePromptOptions(request.SystemPrompt)
	today := options.Now

	if today.IsZero() {
		today = time.Now()
	}

	parses := make([]*donut.Recognized, 0, 3)
	sources := donut.FieldSources{}

	switch request.UserPromptType {
	case data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_TEXT:
		text := documentTextPrefixPattern.ReplaceAllString(string(request.UserPrompt), "")
		parses = append(parses, donut.FromText(text, p.dateOrder, today))
	case data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL:
		recognized, modelSources, err := p.readImage(c, uid, request.UserPrompt, today, 0)

		if err != nil {
			return nil, err
		}

		parses, sources = recognized, modelSources
	case data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_DOCUMENT:
		if request.UserPromptContentType != "application/pdf" && !bytes.HasPrefix(request.UserPrompt, []byte("%PDF-")) {
			return nil, errs.ErrLargeLanguageModelDocumentNotSupported
		}

		page, text, err := donut.ReadPDF(request.UserPrompt, pdfMaxTextPages)

		if err != nil {
			log.Warnf(c, "[donut_large_language_model_provider.GetJsonResponse] cannot read pdf document for user \"uid:%d\", because %s", uid, err.Error())

			if errors.Is(err, donut.ErrPdfToolsNotFound) {
				return nil, errs.ErrLargeLanguageModelDocumentNotSupported
			}

			return nil, errs.ErrOperationFailed
		}

		offset := 0

		// the text layer of a digital PDF is exact, so it is preferred for the amount and the date
		if len(strings.TrimSpace(text)) >= pdfMinimumTextLength {
			parses = append(parses, donut.FromText(text, p.dateOrder, today))
			offset = 1
		}

		recognized, modelSources, err := p.readImage(c, uid, page, today, offset)

		if err != nil {
			return nil, err
		}

		parses = append(parses, recognized...)
		sources = modelSources

		if offset == 1 {
			sources.Amount = append([]int{0}, sources.Amount...)
			sources.DateTime = append([]int{0}, sources.DateTime...)
		}
	default:
		return nil, errs.ErrLargeLanguageModelDocumentNotSupported
	}

	merged := donut.Merge(parses, sources)
	result := donut.ToRecognitionResult(merged, options, p.account)
	content, err := json.Marshal(result)

	if err != nil {
		return nil, errs.ErrOperationFailed
	}

	log.Debugf(c, "[donut_large_language_model_provider.GetJsonResponse] recognized %s for user \"uid:%d\"", content, uid)

	return &data.LargeLanguageModelTextualResponse{Content: string(content)}, nil
}

// readImage reads the image with all models, and returns the parses and the preferred source of each field (indexes are shifted by offset)
func (p *DonutLargeLanguageModelProvider) readImage(c core.Context, uid int64, image []byte, today time.Time, offset int) ([]*donut.Recognized, donut.FieldSources, error) {
	sources := donut.FieldSources{}

	if err := p.runtime.store.EnsureReady(); err != nil {
		log.Warnf(c, "[donut_large_language_model_provider.readImage] the models are not ready for user \"uid:%d\", because %s", uid, err.Error())

		if errors.Is(err, donut.ErrModelDownloading) {
			return nil, sources, errs.NewNormalError(errs.NormalSubcategoryLargeLanguageModel, 12, http.StatusServiceUnavailable, err.Error())
		}

		return nil, sources, errs.ErrDonutWorkerUnavailable
	}

	results, err := p.runtime.worker.Read(p.runtime.store.ModelDirs(), image, p.timeout+time.Minute)

	if err != nil {
		log.Errorf(c, "[donut_large_language_model_provider.readImage] cannot read the image for user \"uid:%d\", because %s", uid, err.Error())

		if errors.Is(err, donut.ErrWorkerUnavailable) {
			return nil, sources, errs.ErrDonutWorkerUnavailable
		}

		return nil, sources, errs.ErrOperationFailed
	}

	parses := make([]*donut.Recognized, 0, len(results))

	for i, result := range results {
		log.Debugf(c, "[donut_large_language_model_provider.readImage] model \"%s\" read %d tokens in %dms: %s", result.Model, result.Tokens, result.Millis, result.Sequence)
		parses = append(parses, donut.FromDonut(donut.TokensToJSON(result.Sequence), p.dateOrder, today))

		// "sroie" reads totals best, "receipts" reads merchant names, dates and times best
		switch result.Model {
		case "sroie":
			sources.Amount = append([]int{offset + i}, sources.Amount...)
		case "receipts":
			sources.Supplier = append([]int{offset + i}, sources.Supplier...)
			sources.DateTime = append([]int{offset + i}, sources.DateTime...)
		}
	}

	return parses, sources, nil
}
