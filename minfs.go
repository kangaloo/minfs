package minfs

import (
	"archive/tar"
	"compress/gzip"
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

// Create 创建可写的文件，同名对象会被覆盖
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
			wg.Done()
			Logger.Println("[MINFS] upload process completed")
		}()

		defer func() {
			if err := reader.Close(); err != nil {
				Logger.Println("[MINFS] reader close failed " + err.Error())
			}
			Logger.Println("[MINFS] reader closed successfully")
		}()

		opts := minio.PutObjectOptions{}
		opts.PartSize = 1024 * 1024 * 1024 * 5
		stat, err := fs.client.PutObject(context.Background(), fs.bucket, name, reader, -1, opts)
		if err != nil {
			Logger.Println("[MINFS] upload error " + err.Error())
			return
		}
		Logger.Printf(
			"[MINFS] upload completed, bucket: %s, key: %s, size: %d, lastModified: %s, location: %s\n",
			stat.Bucket,
			stat.Key,
			stat.Size,
			stat.LastModified.String(),
			stat.Location,
		)
	}()

	return &MinFile{
		wg:           wg,
		uploadReader: reader,
		uploadWriter: writer,
		uploadName:   name,
	}, nil
}

// OpenTar
func (fs *FS) OpenTar(path string) (io.ReadCloser, error) {

	if !strings.HasSuffix(path, "/") || strings.Contains(path, `\`) {
		return nil, errors.New(fmt.Sprintf("invalid path: %s", path))
	}

	opts := minio.ListObjectsOptions{}
	opts.Prefix = path
	opts.Recursive = true
	infoCh := fs.client.ListObjects(context.Background(), fs.bucket, opts)

	var files []*TarFile
	for info := range infoCh {
		info := info
		name := strings.TrimPrefix(info.Key, path)
		name = strings.TrimPrefix(name, "/")
		file := &TarFile{}
		file.Key = info.Key
		file.Name = name
		files = append(files, file)
	}

	pipeReader, pipeWriter := io.Pipe()
	compressWriter := gzip.NewWriter(pipeWriter)
	tarWriter := tar.NewWriter(compressWriter)

	go func() {
		defer func() {
			if err := tarWriter.Close(); err != nil {
				Logger.Println("[MINFS] " + err.Error())
			}
			if err := compressWriter.Close(); err != nil {
				Logger.Println("[MINFS] " + err.Error())
			}
			if err := pipeWriter.Close(); err != nil {
				Logger.Println("[MINFS] " + err.Error())
			}
		}()

		for _, file := range files {
			obj, err := fs.client.GetObject(context.Background(), fs.bucket, file.Key, minio.GetObjectOptions{})
			if err != nil {
				Logger.Println("[MINFS] " + err.Error())
				return
			}

			info, err := obj.Stat()
			if err != nil {
				Logger.Println("[MINFS] " + err.Error())
				return
			}

			h := &tar.Header{}
			h.Name = file.Name
			h.Size = info.Size
			h.ModTime = info.LastModified
			h.Mode = 0644

			if err := tarWriter.WriteHeader(h); err != nil {
				Logger.Println("[MINFS] " + err.Error())
				return
			}

			_, err = io.Copy(tarWriter, obj)
			if err != nil {
				Logger.Println("[MINFS] " + err.Error())
				return
			}

			if err := obj.Close(); err != nil {
				Logger.Println("[MINFS] " + err.Error())
				return
			}
		}
	}()

	return pipeReader, nil
}

type TarFile struct {
	Name string // 用于压缩文件的 header
	Key  string // 用于从 minio 读取数据
}
