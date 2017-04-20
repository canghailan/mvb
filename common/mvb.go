package mvb

import (
	"log"
	"os"
	"path/filepath"
	"io/ioutil"
	"time"
)

const ISO8601 = "20060102150405-0700"

func GetPath(rel string) string  {
	p, err := filepath.Abs(rel)
	if err != nil {
		panic(err)
	}
	return p
}

func GetBasePath() string {
	return GetPath(".")
}

func GetIndexPath() string {
	return GetPath("index")
}

func GetObjectsPath() string {
	return GetPath("objects")
}

func GetObjectPath(objectId string) string {
	return  GetPath(filepath.Join("objects", objectId[0:2], objectId[2:]))
}

func GetLinksPath() string {
	return GetPath("links")
}

func GetLinkPath(snapshot Snapshot) string {
	return  GetPath(filepath.Join("objects", ""))
}

func GetLinkOfNowPath() string {
	return GetPath(filepath.Join("links", "now"))
}

func GetListPath() string {
	return GetPath("list")
}

func Init(root string) {
	println(1)
	if err := os.MkdirAll(GetObjectsPath(), DefaultDirPerm); err != nil {
		panic(err)
	}
	println(1)
	if err := os.MkdirAll(GetLinksPath(), DefaultDirPerm); err != nil {
		panic(err)
	}
	println(1)
	if err := os.Symlink(root, GetLinkOfNowPath()); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(GetIndexPath(), []byte(""), DefaultPerm); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(GetListPath(), []byte(""), DefaultPerm); err != nil {
		panic(err)
	}
	log.Println("init")
}

func List(prefix string, offset int, limit int)  {
	r := NewIndexReader(-1)
	n := 0
	for {
		s := r.ReadSnapshot()
		if s == nil {
			break
		}
		n++
		println(StringifySnapshot(s))
		if n >= offset + limit {
			break
		}
	}
}

func Backup() {
	timestamp := time.Now()

	MakeFileObjectList()
	CopyFileObjects()
	s := CreateSnapshot(timestamp)
	if s.Timestamp == timestamp {
		iw := NewIndexWriter()
		defer iw.Close()
		iw.WriteSnapshot(s)
	}
}

func Restore() {
	
}

func Link() {
	
}

func Delete()  {
	
}

func Clean()  {
	
}