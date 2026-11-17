package storage

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// LocalFileSystemObjectStorage represents local file system object storage
type LocalFileSystemObjectStorage struct {
	rootPath string
}

// NewLocalFileSystemObjectStorage returns a local file system object storage
func NewLocalFileSystemObjectStorage(config *settings.Config, pathPrefix string) (*LocalFileSystemObjectStorage, error) {
	storage := &LocalFileSystemObjectStorage{
		rootPath: filepath.Join(config.LocalFileSystemPath, pathPrefix),
	}

	if err := os.MkdirAll(storage.rootPath, os.ModePerm); err != nil {
		return nil, err
	}

	return storage, nil
}

// Exists returns whether the file exists
func (s *LocalFileSystemObjectStorage) Exists(ctx core.Context, path string) (bool, error) {
	return utils.IsExists(s.getFinalPath(path))
}

// Read returns the object instance according to specified the file path
func (s *LocalFileSystemObjectStorage) Read(ctx core.Context, path string) (ObjectInStorage, bool, error) {
	object, err := os.Open(s.getFinalPath(path))

	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return object, true, nil
}

// Save returns whether save the object instance successfully
func (s *LocalFileSystemObjectStorage) Save(ctx core.Context, path string, object ObjectInStorage) error {
	finalPath := s.getFinalPath(path)

	if err := os.MkdirAll(filepath.Dir(finalPath), os.ModePerm); err != nil {
		return err
	}

	targetFile, err := os.Create(finalPath)

	if err != nil {
		return err
	}

	defer targetFile.Close()

	_, err = io.Copy(targetFile, object)

	return err
}

// Delete returns whether delete the object according to specified the file path successfully
func (s *LocalFileSystemObjectStorage) Delete(ctx core.Context, path string) error {
	err := os.Remove(s.getFinalPath(path))

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// List returns the paths of all the objects in this storage
func (s *LocalFileSystemObjectStorage) List(ctx core.Context) ([]string, error) {
	paths := make([]string, 0)

	err := filepath.WalkDir(s.rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == s.rootPath {
				return filepath.SkipDir
			}

			return err
		}

		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}

		relativePath, err := filepath.Rel(s.rootPath, path)

		if err != nil {
			return err
		}

		paths = append(paths, filepath.ToSlash(relativePath))
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(paths)

	return paths, nil
}

func (s *LocalFileSystemObjectStorage) getFinalPath(path string) string {
	return filepath.Join(s.rootPath, path)
}
