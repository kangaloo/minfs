package minfs

import (
	"io"
	"io/fs"
	"sync"

	"github.com/minio/minio-go/v7"
)

type File interface {
	fs.File
	io.Seeker
	io.ReaderAt
}

type MinFile struct {
	wg           *sync.WaitGroup
	obj          *minio.Object
	uploadReader *io.PipeReader
	uploadWriter *io.PipeWriter
	uploadName   string
}

func (mf *MinFile) Stat() (fs.FileInfo, error) {
	objInfo, err := mf.obj.Stat()
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		objectInfo: &objInfo,
	}, nil
}

func (mf *MinFile) Read(b []byte) (int, error) {
	return mf.obj.Read(b)
}

func (mf *MinFile) Close() error {
	Logger.Println("[MINFS] closing file")
	defer func() { Logger.Printf("[MINFS] file closed, name: %s", mf.uploadName) }()

	if mf.uploadWriter != nil {
		Logger.Println("[MINFS] find upload writer, will close it")
		err := mf.uploadWriter.Close()

		if err != nil {
			// uploadWriter.Close 返回错误，尝试执行 uploadReader.Close 使上传任务出错
			Logger.Println("[MINFS] close upload writer error: " + err.Error())
			Logger.Println("[MINFS] try to close upload reader")

			rErr := mf.uploadReader.Close()
			if rErr != nil {
				Logger.Println("[MINFS] close upload reader error: " + err.Error())
			}

			mf.wg.Wait()
			return err
		}

		Logger.Println("[MINFS] upload writer closed, wait for the upload process to complete")
		mf.wg.Wait()
		Logger.Println("[MINFS] upload process completed")
	}

	// 使用 fs.Open 打开的只读文件
	if mf.obj != nil {
		err := mf.obj.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (mf *MinFile) Seek(offset int64, whence int) (int64, error) {
	return mf.obj.Seek(offset, whence)
}

func (mf *MinFile) ReadAt(b []byte, offset int64) (n int, err error) {
	return mf.obj.ReadAt(b, offset)
}

func (mf *MinFile) Write(b []byte) (int, error) {
	return mf.uploadWriter.Write(b)
}
