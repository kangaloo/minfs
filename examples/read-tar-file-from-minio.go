// +build ignore

package main

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/kangaloo/minfs"
)

func main() {

	minfs.Logger = log.StandardLogger()
	log.SetReportCaller(true)

	fs, err := minfs.NewMinFS("registry:9000", "minioadmin", "minioadmin", false, "package")
	if err != nil {
		panic(err)
	}

	tarFile, err := fs.OpenTar("minfs/")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := tarFile.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	f, err := os.Create("test.tar.gz")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	_, err = io.Copy(f, tarFile)
	if err != nil {
		panic(err)
	}
}
