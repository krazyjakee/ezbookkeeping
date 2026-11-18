package llm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// modelPath returns a model directory outside the test temp dir, because a background download may still be writing to it when the test ends
func modelPath(t *testing.T) string {
	dir, err := os.MkdirTemp("", "ezbookkeeping-donut-test-")
	assert.Nil(t, err)
	t.Cleanup(func() { go func() { os.RemoveAll(dir) }() })
	return filepath.Join(dir, "donut")
}

func newDonutTestConfig(t *testing.T, workerPath string) *settings.Config {
	return &settings.Config{
		TransactionFromAIImageRecognition: true,
		ReceiptImageRecognitionLLMConfig: &settings.LLMConfig{
			LLMProvider:           settings.DonutLLMProvider,
			DonutWorkerPath:       workerPath,
			DonutModelPath:        modelPath(t),
			DonutModelDownloadURL: "http://127.0.0.1:1/",
			DonutThreads:          1,
			DonutDateOrder:        "DMY",
		},
	}
}

func TestInitializeLargeLanguageModelProvider_DisablesDonutWithoutWorker(t *testing.T) {
	config := newDonutTestConfig(t, filepath.Join(t.TempDir(), "missing-worker"))

	assert.Nil(t, InitializeLargeLanguageModelProvider(config))
	assert.False(t, config.TransactionFromAIImageRecognition)

	// nothing is preloaded when image recognition is disabled
	PreloadReceiptImageRecognitionModel(config)
	_, err := os.Stat(config.ReceiptImageRecognitionLLMConfig.DonutModelPath)
	assert.True(t, os.IsNotExist(err))
}

func TestInitializeLargeLanguageModelProvider_KeepsDonutWithWorker(t *testing.T) {
	workerPath := filepath.Join(t.TempDir(), "ezbookkeeping-donut")
	assert.Nil(t, os.WriteFile(workerPath, []byte("#!/bin/sh\n"), 0o755))
	config := newDonutTestConfig(t, workerPath)

	assert.Nil(t, InitializeLargeLanguageModelProvider(config))
	assert.True(t, config.TransactionFromAIImageRecognition)
	assert.NotNil(t, Container.receiptImageRecognitionCurrentProvider)

	// the preload starts the download in the background (which fails here, the server is not reachable) without blocking
	PreloadReceiptImageRecognitionModel(config)
}
