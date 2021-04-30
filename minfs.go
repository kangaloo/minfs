package minfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinFS(endpoint, accessKey, secretKey string, useSSL bool, bucket string) (*FS, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, err
	}

	return &FS{
		client: client,
		bucket: bucket,
	}, nil
}

type FS struct {
	client *minio.Client
	bucket string
}

// Open 用于返回只读文件
func (fs *FS) Open(name string) (File, error) {
	if !strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return nil, errors.New(fmt.Sprintf("invalid file name: %s", name))
	}

	name = strings.TrimPrefix(name, "/")

	// 验证对象是否存在
	statOpts := minio.StatObjectOptions{}
	_, err := fs.client.StatObject(context.Background(), fs.bucket, name, statOpts)
	if err != nil {
		return nil, err
	}

	getOpts := minio.GetObjectOptions{}
	obj, err := fs.client.GetObject(context.Background(), fs.bucket, name, getOpts)
	if err != nil {
		return nil, err
	}

	return &MinFile{obj: obj}, nil
}

func (fs *FS) Create(name string) (*MinFile, error) {
	if !strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return nil, errors.New(fmt.Sprintf("invalid file name: %s", name))
	}

	name = strings.TrimPrefix(name, "/")

	reader, writer := io.Pipe()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		Logger.Println("[MINFS] upload start")
		defer func() {
			Logger.Println("[MINFS] upload done")
			wg.Done()
		}()

		defer func() {
			Logger.Println("[MINFS] close reader")
			if err := reader.Close(); err != nil {
				Logger.Println("[MINFS] " + err.Error())
			}
			Logger.Println("[MINFS] close reader end")
		}()

		opts := minio.PutObjectOptions{}
		opts.PartSize = 1024 * 1024 * 1024 * 5
		stat, err := fs.client.PutObject(context.Background(), fs.bucket, name, reader, -1, opts)
		if err != nil {
			Logger.Println("[MINFS] upload error")
			Logger.Println("[MINFS] " + err.Error())
			return
		}
		Logger.Println("[MINFS] upload end")
		Logger.Printf("[MINFS] bucket: %s, key: %s, size: %d, lastModified: %s, location: %s\n", stat.Bucket, stat.Key, stat.Size, stat.LastModified.String(), stat.Location)
	}()

	return &MinFile{
		wg:           wg,
		uploadReader: reader,
		uploadWriter: writer,
		uploadName:   name,
	}, nil
}
