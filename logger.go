package minfs

import (
	"io/ioutil"
	"log"
)

var Logger StdLogger = log.New(ioutil.Discard, "[MinFS] ", log.LstdFlags)

type StdLogger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}
