// Command donut-worker runs Donut models with ONNX Runtime for the built-in "donut" recognition provider of ezBookkeeping.
//
// It is started by the ezBookkeeping server on first use and talks to it over stdin / stdout (see pkg/donut/worker_protocol.go),
// so it needs no port. It is a separate binary because it loads the ONNX Runtime shared library, which a statically linked
// ezBookkeeping binary cannot do.
//
// Usage: donut-worker -library /usr/lib/libonnxruntime.so.1 -threads 4 [-low-memory] <model dir> [<model dir>...]
package main

import (
	"bufio"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/mayswind/ezbookkeeping/pkg/donut"
)

type model struct {
	config        *donut.ModelConfig
	tokenizer     *donut.Tokenizer
	encoder       *ort.DynamicAdvancedSession
	decoder       *ort.DynamicAdvancedSession
	decoderInputs []string
	decoderOutput []string
	taskIDs       []int64
}

func main() {
	library := flag.String("library", "libonnxruntime.so.1", "path of the ONNX Runtime shared library")
	threads := flag.Int("threads", 4, "number of CPU threads used by each model")
	lowMemory := flag.Bool("low-memory", false, "use about 40% less memory at about half the speed")
	flag.Parse()

	if err := run(*library, *threads, *lowMemory, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "donut-worker:", err)
		os.Exit(1)
	}
}

func run(library string, threads int, lowMemory bool, modelDirs []string) error {
	output := bufio.NewWriter(os.Stdout)
	respond := func(response *donut.WorkerResponse) {
		content, _ := json.Marshal(response)
		output.Write(content)
		output.WriteByte('\n')
		output.Flush()
	}

	if len(modelDirs) < 1 {
		return errors.New("no model directory specified")
	}

	ort.SetSharedLibraryPath(library)

	if err := ort.InitializeEnvironment(); err != nil {
		respond(&donut.WorkerResponse{Error: "cannot load ONNX Runtime: " + err.Error()})
		return err
	}

	defer ort.DestroyEnvironment()

	models := make([]*model, 0, len(modelDirs))

	for _, dir := range modelDirs {
		m, err := loadModel(dir, threads, lowMemory)

		if err != nil {
			respond(&donut.WorkerResponse{Error: "cannot load model " + dir + ": " + err.Error()})
			return err
		}

		models = append(models, m)
	}

	respond(&donut.WorkerResponse{Ready: true})
	input := bufio.NewScanner(os.Stdin)
	input.Buffer(make([]byte, 1024*1024), 128*1024*1024)

	for input.Scan() {
		var request donut.WorkerRequest

		if err := json.Unmarshal(input.Bytes(), &request); err != nil {
			respond(&donut.WorkerResponse{Error: "invalid request: " + err.Error()})
			continue
		}

		respond(readImage(models, &request))
	}

	return input.Err()
}

func loadModel(dir string, threads int, lowMemory bool) (*model, error) {
	config, err := donut.LoadModelConfig(dir)

	if err != nil {
		return nil, err
	}

	tokenizer, err := donut.LoadTokenizer(filepath.Join(dir, config.TokenizerFile))

	if err != nil {
		return nil, err
	}

	options, err := ort.NewSessionOptions()

	if err != nil {
		return nil, err
	}

	defer options.Destroy()

	if err := options.SetIntraOpNumThreads(threads); err != nil {
		return nil, err
	}

	if err := options.SetInterOpNumThreads(1); err != nil {
		return nil, err
	}

	// without memory patterns the peak memory of both models is about 2.2 GB instead of 2.6 GB at the same speed,
	// without the memory arena it is about 1.5 GB, but reading takes about twice as long
	if err := options.SetMemPattern(false); err != nil {
		return nil, err
	}

	if lowMemory {
		if err := options.SetCpuMemArena(false); err != nil {
			return nil, err
		}
	}

	encoderPath := filepath.Join(dir, config.EncoderFile)
	encoder, err := ort.NewDynamicAdvancedSession(encoderPath, []string{"pixel_values"}, []string{"last_hidden_state"}, options)

	if err != nil {
		return nil, err
	}

	decoderPath := filepath.Join(dir, config.DecoderFile)
	inputInfos, outputInfos, err := ort.GetInputOutputInfo(decoderPath)

	if err != nil {
		return nil, err
	}

	m := &model{config: config, tokenizer: tokenizer, encoder: encoder}

	for _, info := range inputInfos {
		m.decoderInputs = append(m.decoderInputs, info.Name)
	}

	for _, info := range outputInfos {
		m.decoderOutput = append(m.decoderOutput, info.Name)
	}

	if m.decoder, err = ort.NewDynamicAdvancedSession(decoderPath, m.decoderInputs, m.decoderOutput, options); err != nil {
		return nil, err
	}

	taskID, exists := tokenizer.TokenID(config.TaskPrompt)

	if !exists {
		return nil, fmt.Errorf("task prompt %q is not a token", config.TaskPrompt)
	}

	m.taskIDs = []int64{taskID}

	return m, nil
}

func readImage(models []*model, request *donut.WorkerRequest) *donut.WorkerResponse {
	img, err := donut.DecodeImage(request.Image)

	if err != nil {
		return &donut.WorkerResponse{ID: request.ID, Error: "cannot decode image: " + err.Error()}
	}

	response := &donut.WorkerResponse{ID: request.ID}

	for _, m := range models {
		started := time.Now()
		ids, err := m.generate(donut.Preprocess(img, m.config.ImageHeight, m.config.ImageWidth))

		if err != nil {
			return &donut.WorkerResponse{ID: request.ID, Error: m.config.Name + ": " + err.Error()}
		}

		response.Results = append(response.Results, &donut.WorkerModelResult{
			Model:    m.config.Name,
			Sequence: m.tokenizer.Decode(ids),
			Tokens:   len(ids),
			Millis:   time.Since(started).Milliseconds(),
		})
	}

	return response
}

// generate runs the encoder once and the decoder greedily (with key / value cache) until the end of sequence token
func (m *model) generate(pixels []float32) ([]int64, error) {
	pixelTensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(m.config.ImageHeight), int64(m.config.ImageWidth)), pixels)

	if err != nil {
		return nil, err
	}

	defer pixelTensor.Destroy()

	encoderOutputs := []ort.Value{nil}

	if err := m.encoder.Run([]ort.Value{pixelTensor}, encoderOutputs); err != nil {
		return nil, err
	}

	hidden := encoderOutputs[0]
	defer hidden.Destroy()

	cache := make(map[string]ort.Value)

	defer func() {
		for _, value := range cache {
			value.Destroy()
		}
	}()

	emptyShape := ort.NewShape(1, int64(m.config.DecoderAttentionHeads), 0, int64(m.config.DecoderHeadDim))

	for _, name := range m.decoderInputs {
		if strings.HasPrefix(name, "past_key_values.") {
			empty, err := ort.NewTensor(emptyShape, []float32{})

			if err != nil {
				return nil, err
			}

			cache[name] = empty
		}
	}

	generated := make([]int64, 0, 128)
	nextInput := m.taskIDs
	useCache := false

	for step := 0; step < m.config.MaxLength; step++ {
		idsTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(nextInput))), nextInput)

		if err != nil {
			return nil, err
		}

		useCacheTensor, err := ort.NewTensor(ort.NewShape(1), []bool{useCache})

		if err != nil {
			idsTensor.Destroy()
			return nil, err
		}

		inputs := make([]ort.Value, len(m.decoderInputs))

		for i, name := range m.decoderInputs {
			switch {
			case name == "input_ids":
				inputs[i] = idsTensor
			case name == "encoder_hidden_states":
				inputs[i] = hidden
			case name == "use_cache_branch":
				inputs[i] = useCacheTensor
			default:
				inputs[i] = cache[name]
			}
		}

		outputs := make([]ort.Value, len(m.decoderOutput))
		err = m.decoder.Run(inputs, outputs)
		idsTensor.Destroy()
		useCacheTensor.Destroy()

		if err != nil {
			return nil, err
		}

		next := int64(-1)

		for i, name := range m.decoderOutput {
			value := outputs[i]

			if name == "logits" {
				next = argmaxLastPosition(value.(*ort.Tensor[float32]), m.tokenizer.UnkID)
				value.Destroy()
				continue
			}

			pastName := strings.Replace(name, "present.", "past_key_values.", 1)

			// the cross attention cache only depends on the image, keep the one from the first step
			if useCache && strings.Contains(name, ".encoder.") {
				value.Destroy()
				continue
			}

			if old, exists := cache[pastName]; exists {
				old.Destroy()
			}

			cache[pastName] = value
		}

		useCache = true

		if next < 0 || next == m.tokenizer.EosID {
			break
		}

		generated = append(generated, next)
		nextInput = []int64{next}
	}

	return generated, nil
}

// argmaxLastPosition returns the most likely next token (excluding the unknown token) from logits of shape [1, sequence, vocabulary]
func argmaxLastPosition(logits *ort.Tensor[float32], excluded int64) int64 {
	shape := logits.GetShape()
	vocabulary := int(shape[len(shape)-1])
	data := logits.GetData()
	last := data[len(data)-vocabulary:]
	best := int64(-1)
	bestValue := float32(math.Inf(-1))

	for i, value := range last {
		if int64(i) != excluded && value > bestValue {
			best, bestValue = int64(i), value
		}
	}

	return best
}
