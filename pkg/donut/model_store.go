package donut

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ModelFile is a file of a converted Donut model
type ModelFile struct {
	Path   string
	Size   int64
	SHA256 string
}

// ModelManifest lists the files of a converted Donut model
type ModelManifest struct {
	Name  string
	Files []ModelFile
}

// DefaultModels are the Donut checkpoints used by the built-in provider, converted to ONNX (8-bit weights). The order matters:
// "receipts" (mychen76/invoice-and-receipts_donut_v1, Apache 2.0) reads merchant names, dates and times best, and
// "sroie" (unstructuredio/donut-base-sroie, MIT) reads totals best
var DefaultModels = []ModelManifest{
	{
		Name: "receipts",
		Files: []ModelFile{
			{Path: "receipts/donut.json", Size: 365, SHA256: "f67f550cc0128b12dc908f4cc6072264acc58273f3b5ba0a50e7947c620048d0"},
			{Path: "receipts/tokenizer.json", Size: 4021351, SHA256: "3f4937e2ef3dd64fa40ca963f8af6e595e8e103b813a2a8d6a7dddf1ac002836"},
			{Path: "receipts/encoder.onnx", Size: 90623959, SHA256: "a43d6c84a9a6cf1a924645ba7d0dea4697bf6b2b59be5f2997d0af868e38b6ca"},
			{Path: "receipts/decoder.onnx", Size: 191296244, SHA256: "d314d0bc64ced9703adb75e304bacb37aafb4cd2d7814355b53ca3da0de1ebfa"},
		},
	},
	{
		Name: "sroie",
		Files: []ModelFile{
			{Path: "sroie/donut.json", Size: 346, SHA256: "02565d74a2c4f3cbfe28b7c91cdc558ef8f87fb6a01540af2d02ef9d3a1558fb"},
			{Path: "sroie/tokenizer.json", Size: 4009401, SHA256: "ba311372cc837e225b5dc1d3fd32d1d120e77bec626254fea07bd52fde99539f"},
			{Path: "sroie/encoder.onnx", Size: 90681066, SHA256: "90a4667a405a182fde05b4b71c9115ee0c2dda0918e2d883bc4dc93a2407042f"},
			{Path: "sroie/decoder.onnx", Size: 191165204, SHA256: "ca0435adcea1431a59b62473aa4cb86da1507049b79459fbf25db01a909accd8"},
		},
	},
}

// verifiedMarkerFileName is the name of the file which records the checksums of the verified model files, so they are not hashed on every start
const verifiedMarkerFileName = ".verified"

// ErrModelDownloading is returned while the models are being downloaded
var ErrModelDownloading = errors.New("the receipt recognition models are being downloaded")

// ModelStore downloads the model files on first use and verifies their checksums
type ModelStore struct {
	dir        string
	baseURL    string
	client     *http.Client
	models     []ModelManifest
	mutex      sync.Mutex
	ready      bool
	running    bool
	lastError  error
	total      int64
	downloaded atomic.Int64
	logf       func(format string, args ...any)
}

// NewModelStore returns a model store which keeps the models in dir and downloads them from baseURL (the file paths are appended to it)
func NewModelStore(dir string, baseURL string, client *http.Client, models []ModelManifest, logf func(format string, args ...any)) *ModelStore {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	if logf == nil {
		logf = func(string, ...any) {}
	}

	store := &ModelStore{dir: dir, baseURL: baseURL, client: client, models: models, logf: logf}

	for _, model := range models {
		for _, file := range model.Files {
			store.total += file.Size
		}
	}

	return store
}

// ModelDirs returns the directories of the models (in manifest order)
func (s *ModelStore) ModelDirs() []string {
	dirs := make([]string, len(s.models))

	for i, model := range s.models {
		dirs[i] = filepath.Join(s.dir, model.Name)
	}

	return dirs
}

// ModelNames returns the names of the models (in manifest order)
func (s *ModelStore) ModelNames() []string {
	names := make([]string, len(s.models))

	for i, model := range s.models {
		names[i] = model.Name
	}

	return names
}

// EnsureReady returns nil if all model files are present and verified. Otherwise it starts downloading them in the background
// (unless a download is already running) and returns an error which describes the progress, so callers can ask the user to retry later
func (s *ModelStore) EnsureReady() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ready {
		return nil
	}

	if !s.running && s.isVerified() {
		s.ready = true
		return nil
	}

	if !s.running {
		previousError := s.lastError
		s.running = true
		s.lastError = nil
		s.downloaded.Store(0)
		go s.download()

		if previousError != nil {
			return fmt.Errorf("%w (retrying after the previous attempt failed: %s)", ErrModelDownloading, previousError.Error())
		}
	}

	progress := float64(s.downloaded.Load()) * 100 / float64(max(s.total, 1))
	return fmt.Errorf("%w (%.0f%% of %d MB), try again in a few minutes", ErrModelDownloading, progress, s.total/1_000_000)
}

func (s *ModelStore) markerContent() string {
	var builder strings.Builder

	for _, model := range s.models {
		for _, file := range model.Files {
			builder.WriteString(file.SHA256 + "  " + file.Path + "\n")
		}
	}

	return builder.String()
}

// isVerified returns whether the verified marker matches the manifest and all files have the expected size
func (s *ModelStore) isVerified() bool {
	marker, err := os.ReadFile(filepath.Join(s.dir, verifiedMarkerFileName))

	if err != nil || string(marker) != s.markerContent() {
		return false
	}

	for _, model := range s.models {
		for _, file := range model.Files {
			info, err := os.Stat(filepath.Join(s.dir, file.Path))

			if err != nil || info.Size() != file.Size {
				return false
			}
		}
	}

	return true
}

func (s *ModelStore) download() {
	err := s.downloadAll()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.running = false

	if err != nil {
		s.lastError = err
		s.logf("failed to download the donut models, because %s", err.Error())
		return
	}

	s.ready = true
	s.logf("the donut models have been downloaded to %s", s.dir)
}

func (s *ModelStore) downloadAll() error {
	for _, model := range s.models {
		for _, file := range model.Files {
			path := filepath.Join(s.dir, file.Path)

			if hasChecksum(path, file) {
				s.downloaded.Add(file.Size)
				continue
			}

			if err := s.downloadFile(file, path); err != nil {
				return fmt.Errorf("%s: %w", file.Path, err)
			}
		}
	}

	return os.WriteFile(filepath.Join(s.dir, verifiedMarkerFileName), []byte(s.markerContent()), 0o644)
}

func (s *ModelStore) downloadFile(file ModelFile, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+file.Path, nil)

	if err != nil {
		return err
	}

	s.logf("downloading %s", s.baseURL+file.Path)
	response, err := s.client.Do(request)

	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http status %s", response.Status)
	}

	partPath := path + ".part"
	output, err := os.Create(partPath)

	if err != nil {
		return err
	}

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(output, hash, progressWriter{&s.downloaded}), io.LimitReader(response.Body, file.Size+1))
	closeErr := output.Close()

	if err == nil {
		err = closeErr
	}

	if err == nil && written != file.Size {
		err = fmt.Errorf("expected %d bytes, but got %d", file.Size, written)
	}

	if err == nil && hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
		err = errors.New("checksum mismatch")
	}

	if err != nil {
		os.Remove(partPath)
		return err
	}

	return os.Rename(partPath, path)
}

func hasChecksum(path string, file ModelFile) bool {
	info, err := os.Stat(path)

	if err != nil || info.Size() != file.Size {
		return false
	}

	input, err := os.Open(path)

	if err != nil {
		return false
	}

	defer input.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, input); err != nil {
		return false
	}

	return hex.EncodeToString(hash.Sum(nil)) == file.SHA256
}

type progressWriter struct {
	counter *atomic.Int64
}

func (w progressWriter) Write(data []byte) (int, error) {
	w.counter.Add(int64(len(data)))
	return len(data), nil
}
