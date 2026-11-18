// Package donut reads receipts and invoices with Donut document understanding models (https://arxiv.org/abs/2111.15664).
//
// The models run in a separate worker process (tools/donut-worker), which needs the ONNX Runtime library. This package
// contains everything else: the model manifest and downloader, the worker process manager, the image preprocessing and
// tokenizer shared with the worker, the conversion of Donut output to JSON, and the translation of the recognized fields
// into the transaction JSON which ezBookkeeping expects from a recognition model.
package donut

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
)

// ModelConfigFileName is the name of the file which describes a converted Donut model
const ModelConfigFileName = "donut.json"

// ModelConfig describes a Donut model converted to ONNX
type ModelConfig struct {
	Name                  string `json:"name"`
	TaskPrompt            string `json:"task_prompt"`
	ImageHeight           int    `json:"image_height"`
	ImageWidth            int    `json:"image_width"`
	MaxLength             int    `json:"max_length"`
	DecoderLayers         int    `json:"decoder_layers"`
	DecoderAttentionHeads int    `json:"decoder_attention_heads"`
	DecoderHeadDim        int    `json:"decoder_head_dim"`
	EncoderFile           string `json:"encoder_file"`
	DecoderFile           string `json:"decoder_file"`
	TokenizerFile         string `json:"tokenizer_file"`
}

// LoadModelConfig reads the model config from the model directory
func LoadModelConfig(modelDir string) (*ModelConfig, error) {
	content, err := os.ReadFile(filepath.Join(modelDir, ModelConfigFileName))

	if err != nil {
		return nil, err
	}

	config := &ModelConfig{}

	if err := json.Unmarshal(content, config); err != nil {
		return nil, err
	}

	return config, nil
}
