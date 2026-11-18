package donut

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testManifest(files map[string]string) []ModelManifest {
	manifest := ModelManifest{Name: "test"}
	paths := make([]string, 0, len(files))

	for path := range files {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	for _, path := range paths {
		content := files[path]
		checksum := sha256.Sum256([]byte(content))
		manifest.Files = append(manifest.Files, ModelFile{Path: path, Size: int64(len(content)), SHA256: hex.EncodeToString(checksum[:])})
	}

	return []ModelManifest{manifest}
}

func waitUntilReady(t *testing.T, store *ModelStore) error {
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		err := store.EnsureReady()

		if err == nil || !errors.Is(err, ErrModelDownloading) {
			return err
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("download did not finish")
	return nil
}

func TestModelStore_DownloadsVerifiesAndReuses(t *testing.T) {
	files := map[string]string{"test/donut.json": `{"name":"test"}`, "test/encoder.onnx": "encoder-bytes"}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		content, exists := files[r.URL.Path[len("/models/"):]]

		if !exists {
			http.NotFound(w, r)
			return
		}

		w.Write([]byte(content))
	}))
	defer server.Close()

	dir := t.TempDir()
	store := NewModelStore(dir, server.URL+"/models", server.Client(), testManifest(files), nil)

	err := store.EnsureReady()
	assert.ErrorIs(t, err, ErrModelDownloading)
	assert.Contains(t, err.Error(), "try again in a few minutes")
	assert.Nil(t, waitUntilReady(t, store))

	content, _ := os.ReadFile(filepath.Join(dir, "test/encoder.onnx"))
	assert.Equal(t, "encoder-bytes", string(content))
	assert.Equal(t, []string{filepath.Join(dir, "test")}, store.ModelDirs())
	assert.Equal(t, int32(2), requests.Load())

	// a new store (e.g. after a restart) uses the verified files without downloading them again
	restarted := NewModelStore(dir, server.URL+"/models", server.Client(), testManifest(files), nil)
	assert.Nil(t, restarted.EnsureReady())
	assert.Equal(t, int32(2), requests.Load())
}

func TestModelStore_RejectsChecksumMismatchAndRetries(t *testing.T) {
	files := map[string]string{"test/encoder.onnx": "expected"}
	var serveWrongContent atomic.Bool
	serveWrongContent.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveWrongContent.Load() {
			w.Write([]byte("tampered"))
		} else {
			w.Write([]byte("expected"))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	store := NewModelStore(dir, server.URL, server.Client(), testManifest(files), nil)

	assert.ErrorIs(t, store.EnsureReady(), ErrModelDownloading)
	deadline := time.Now().Add(5 * time.Second)

	for {
		store.mutex.Lock()
		running, lastError := store.running, store.lastError
		store.mutex.Unlock()

		if !running && lastError != nil {
			assert.Contains(t, lastError.Error(), "checksum mismatch")
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("download did not fail")
		}

		time.Sleep(10 * time.Millisecond)
	}

	_, err := os.Stat(filepath.Join(dir, "test/encoder.onnx"))
	assert.True(t, os.IsNotExist(err))

	serveWrongContent.Store(false)
	err = store.EnsureReady()
	assert.ErrorIs(t, err, ErrModelDownloading)
	assert.Contains(t, err.Error(), "checksum mismatch")
	assert.Nil(t, waitUntilReady(t, store))
}

func TestModelStore_DefaultManifestIsComplete(t *testing.T) {
	names := map[string]bool{}

	for _, model := range DefaultModels {
		names[model.Name] = true
		paths := map[string]bool{}

		for _, file := range model.Files {
			paths[filepath.Base(file.Path)] = true
			assert.Len(t, file.SHA256, 64)
			assert.Greater(t, file.Size, int64(0))
		}

		assert.Equal(t, map[string]bool{"donut.json": true, "tokenizer.json": true, "encoder.onnx": true, "decoder.onnx": true}, paths, model.Name)
	}

	assert.Equal(t, map[string]bool{"receipts": true, "sroie": true}, names)
}
