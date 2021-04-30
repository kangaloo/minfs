// +build ignore

package main

import (
	"io"
	"os"

	"github.com/kangaloo/minfs"

	log "github.com/sirupsen/logrus"
)

func main() {
	minfs.Logger = log.StandardLogger()
	log.SetReportCaller(true)

	// create file system
	fs, err := minfs.NewMinFS("minio-server-address:9000", "minioadmin", "minioadmin", false, "package")
	if err != nil {
		panic(err)
	}

	// create a remote file
	mf, err := fs.Create("/test.tar")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := mf.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	// open a local file
	localFile, err := os.Open("test.tar")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := localFile.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	// write to remote file
	_, err := io.Copy(mf, localFile)
	if err != nil {
		panic(err)
	}
}
