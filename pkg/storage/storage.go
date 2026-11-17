package storage

import "github.com/mayswind/ezbookkeeping/pkg/core"

// ObjectStorage represents an object storage to store file object
type ObjectStorage interface {
	Exists(ctx core.Context, path string) (bool, error)
	Read(ctx core.Context, path string) (object ObjectInStorage, exists bool, err error)
	Save(ctx core.Context, path string, object ObjectInStorage) error
	Delete(ctx core.Context, path string) error

	// List returns the paths (relative to the root path of this storage, separated by "/") of all the objects in this storage
	List(ctx core.Context) ([]string, error)
}
