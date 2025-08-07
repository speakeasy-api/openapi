package system

import (
	"io/fs"
	"os"
)

type VirtualFS interface {
	fs.FS
}

type FileSystem struct{}

var _ VirtualFS = (*FileSystem)(nil)

func (fs *FileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}
