// +build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"time"

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

	// open a remote file
	f, err := fs.Open("/test.txt")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	// get file info
	info, err := f.Stat()
	if err != nil {
		panic(err)
	}

	location, err := time.LoadLocation("Local")
	if err != nil {
		panic(err)
	}

	// print file info
	fmt.Printf("name: %s, size: %d, lastModified: %s\n", info.Name(), info.Size(), info.ModTime().In(location).String())

	// create a local file
	localFile, err := os.Create("test.txt")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := localFile.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	// read remote file and write to local file
	_, err = io.Copy(localFile, f)
	if err != nil {
		panic(err)
	}

}
