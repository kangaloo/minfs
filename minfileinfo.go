package minfs

import (
	"io/fs"
	"time"

	"github.com/minio/minio-go/v7"
)

type FileInfo struct {
	objectInfo *minio.ObjectInfo
}

func (fi *FileInfo) Name() string {
	return ""
}

func (fi *FileInfo) Size() int64 {
	return fi.objectInfo.Size
}

func (fi *FileInfo) Mode() fs.FileMode {
	return 0
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.objectInfo.LastModified
}

func (fi *FileInfo) IsDir() bool {
	return false
}

func (fi *FileInfo) Sys() interface{} {
	return nil
}
