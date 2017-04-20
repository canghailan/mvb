package mvb

import (
	"io"
	"os"
	"path/filepath"
)

const DefaultPerm = 0664
const DefaultObjectPerm = 0444
const DefaultDirPerm = os.ModeDir | 0774

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			panic(err)
		}
	}
	return true
}

func CopyFile(from string, to string) {
	err := os.MkdirAll(filepath.Dir(to), DefaultDirPerm)
	if err != nil {
		panic(err)
	}

	writer, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY, DefaultObjectPerm)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	reader, err := os.Open(from)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		panic(err)
	}
}
