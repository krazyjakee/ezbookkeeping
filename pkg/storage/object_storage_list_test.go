package storage

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/webdav"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func testObjectStorageList(t *testing.T, storage ObjectStorage) {
	ctx := core.NewNullContext()

	paths, err := storage.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, paths)

	objects := map[string]string{
		"1.png":                  "avatar",
		"123/456.pdf":            "attachment",
		"123/789 with space.jpg": "picture",
		"999/nested/deep.bin":    "deep",
	}

	for path, content := range objects {
		require.NoError(t, storage.Save(ctx, path, newByteSliceObject([]byte(content))))
	}

	paths, err = storage.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.png", "123/456.pdf", "123/789 with space.jpg", "999/nested/deep.bin"}, paths)

	for _, path := range paths {
		object, exists, err := storage.Read(ctx, path)
		require.NoError(t, err)
		require.True(t, exists)

		data, err := io.ReadAll(object)
		require.NoError(t, err)
		_ = object.Close()
		assert.Equal(t, objects[path], string(data))
	}
}

func TestLocalFileSystemObjectStorage_List(t *testing.T) {
	storage, err := NewLocalFileSystemObjectStorage(&settings.Config{LocalFileSystemPath: t.TempDir()}, "avatar")
	require.NoError(t, err)

	testObjectStorageList(t, storage)
}

func TestWebDAVObjectStorage_List(t *testing.T) {
	server := httptest.NewServer(&webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
	})
	defer server.Close()

	storage, err := NewWebDAVObjectStorage(&settings.Config{
		WebDAVConfig: &settings.WebDAVConfig{
			Url:            server.URL + "/",
			RootPath:       "ezbookkeeping data",
			RequestTimeout: 10000,
			Proxy:          "none",
		},
	}, "transaction")
	require.NoError(t, err)

	testObjectStorageList(t, storage)
}
