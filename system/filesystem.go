package system

import (
	"io/fs"
	"os"
	"path/filepath"
)

type VirtualFS interface {
	fs.FS
}

// WritableVirtualFS extends VirtualFS with write operations needed for localization
type WritableVirtualFS interface {
	VirtualFS
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

type FileSystem struct{}

var _ VirtualFS = (*FileSystem)(nil)
var _ WritableVirtualFS = (*FileSystem)(nil)

func (fs *FileSystem) Open(name string) (fs.File, error) {
	return os.Open(name) //nolint:gosec
}

func (fs *FileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(name, data, perm)
}

func (fs *FileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
