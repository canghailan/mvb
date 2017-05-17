package main

import (
	"./mvb"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// mvb init
func initialize(root string) {
	if err := ioutil.WriteFile("ref", []byte(root), 0644); err != nil {
		log.Fatalf("init: %v", err)
	}
}

// mvb backup
func backup() {
	t := time.Now()
	fs := mvb.GetFiles()
	v := mvb.ToVersionText(fs)
	id := mvb.Sha1([]byte(v))

	if mvb.IsObjectExist(id) {
		os.Stdout.WriteString(id)
		return
	}

	mvb.CopyFileObjects(fs)
	mvb.WriteVersionFile(id, v)
	mvb.AddVersionToIndex(id, t)
	os.Stdout.WriteString(id)
}

// mvb restore [version] [root]
func restore(version string, root string) {

}

// mvb link [version] [to]
func link(version string, root string) {

}

// mvb list
func list() {
	n := mvb.GetIndexVersionCount()
	if n == 0 {
		return
	}

	f, err := os.Open("index")
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	defer f.Close()

	buf := make([]byte, mvb.VERSION_LINE_LEN)
	for i := int64(n - 1); i >= 0; i-- {
		f.ReadAt(buf, i*int64(mvb.VERSION_LINE_LEN))
		os.Stdout.Write(buf)
	}
}

// mvb delete [version]
func delete(version string) {
	log.Fatal("delete: not supported")
}

// mvb gc
func gc() {
	log.Fatal("gc: not supported")
}

func message() {
	println(`usage:
mvb init [path]
mvb backup
mvb restore [version] [path]
mvb link [version] [path]
mvb list
mvb delete [versions]
mvb gc`)
}

func main() {
	backup()
}
