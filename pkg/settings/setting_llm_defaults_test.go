package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setTestWorkingPath uses a temporary working path, the loader requires the static root path (public) and the storage path to exist
func setTestWorkingPath(t *testing.T) {
	root := t.TempDir()

	for _, dir := range []string{"public", "storage", "data", "log"} {
		assert.Nil(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
	}

	t.Setenv("EBK_WORK_DIR", root)
}

// loadTestConfiguration loads the shipped configuration file after removing the lines in removedLines and appending the extra content
func loadTestConfiguration(t *testing.T, removedLines []string, extra string) *Config {
	setTestWorkingPath(t)
	shipped, err := os.ReadFile("../../conf/ezbookkeeping.ini")
	assert.Nil(t, err)
	content := string(shipped)

	for _, line := range removedLines {
		assert.Contains(t, content, line+"\n")
		content = strings.Replace(content, line+"\n", "", 1)
	}

	path := filepath.Join(t.TempDir(), "ezbookkeeping.ini")
	assert.Nil(t, os.WriteFile(path, []byte(content+"\n"+extra), 0o644))
	config, err := LoadConfiguration(path)
	assert.Nil(t, err)
	return config
}

func TestLoadConfiguration_ReceiptRecognitionDefaultsToDonut(t *testing.T) {
	config := loadTestConfiguration(t, []string{"transaction_from_ai_image_recognition = true", "llm_provider = donut", "donut_model_download_url = " + defaultDonutModelDownloadURL}, "")

	assert.True(t, config.TransactionFromAIImageRecognition)
	assert.False(t, config.TransactionFromAITextRecognition)
	assert.Equal(t, DonutLLMProvider, config.ReceiptImageRecognitionLLMConfig.LLMProvider)
	assert.Equal(t, "", config.TextRecognitionLLMConfig.LLMProvider)
	assert.Equal(t, "", config.CodingAssistantLLMConfig.LLMProvider)
	assert.Equal(t, defaultDonutModelDownloadURL, config.ReceiptImageRecognitionLLMConfig.DonutModelDownloadURL)
	assert.Contains(t, defaultDonutModelDownloadURL, "https://huggingface.co/krazyjakee/ezbookkeeping-donut-onnx/resolve/")
	assert.Equal(t, filepath.Join(config.WorkingPath, "data/donut"), config.ReceiptImageRecognitionLLMConfig.DonutModelPath)
	assert.Equal(t, "DMY", config.ReceiptImageRecognitionLLMConfig.DonutDateOrder)
	assert.Equal(t, uint32(4), config.ReceiptImageRecognitionLLMConfig.DonutThreads)
}

func TestLoadConfiguration_BlankProviderUsesDonutAndCanBeDisabled(t *testing.T) {
	config := loadTestConfiguration(t, []string{"transaction_from_ai_image_recognition = true", "llm_provider = donut"}, "")
	assert.True(t, config.TransactionFromAIImageRecognition)

	t.Setenv("EBK_AI_TRANSACTION_FROM_AI_IMAGE_RECOGNITION", "false")
	config = loadTestConfiguration(t, []string{"llm_provider = donut"}, "")

	assert.False(t, config.TransactionFromAIImageRecognition)
	assert.Equal(t, DonutLLMProvider, config.ReceiptImageRecognitionLLMConfig.LLMProvider)
}

func TestLoadConfiguration_OtherProviderIsKept(t *testing.T) {
	t.Setenv("EBK_LLM_IMAGE_RECOGNITION_LLM_PROVIDER", "openai")
	t.Setenv("EBK_LLM_IMAGE_RECOGNITION_DONUT_DATE_ORDER", "mdy")
	config := loadTestConfiguration(t, nil, "")

	assert.Equal(t, OpenAILLMProvider, config.ReceiptImageRecognitionLLMConfig.LLMProvider)
	assert.Equal(t, "MDY", config.ReceiptImageRecognitionLLMConfig.DonutDateOrder)
}

func TestLoadConfiguration_ShippedConfigUsesDonut(t *testing.T) {
	setTestWorkingPath(t)
	config, err := LoadConfiguration("../../conf/ezbookkeeping.ini")
	assert.Nil(t, err)
	assert.True(t, config.TransactionFromAIImageRecognition)
	assert.Equal(t, DonutLLMProvider, config.ReceiptImageRecognitionLLMConfig.LLMProvider)
	assert.Equal(t, defaultDonutModelDownloadURL, config.ReceiptImageRecognitionLLMConfig.DonutModelDownloadURL)
}
