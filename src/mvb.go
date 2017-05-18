package main

import (
	"./mvb"
	"io/ioutil"
	"log"
	"os"
	"time"
	"fmt"
	"strings"
	"path/filepath"
)

// mvb init [path]
func initialize(path string) {
	if err := ioutil.WriteFile("ref", []byte(path), 0644); err != nil {
		log.Fatalf("init: %v", err)
	}
	fmt.Printf("init: %s", path)
}

// mvb backup
func backup() {
	t := time.Now()
	fs := mvb.GetFiles()
	v := mvb.ToVersionText(fs)
	id := mvb.Sha1([]byte(v))

	if mvb.IsObjectExist(id) {
		fmt.Printf("backup & skip: %s\n", id)
		return
	}

	mvb.CopyFileObjects(fs)
	mvb.WriteVersionFile(id, v)
	mvb.AddVersionToIndex(id, t)
	fmt.Printf("backup: %s\n", id)
}

// mvb restore [version] [path]
func restore(version string, path string) {
	if path == "" {
		path = mvb.GetRef()
	}
}

// mvb link [version] [path]
func link(version string, path string) {
	fs := mvb.GetVersionFiles(version)
	for _, f := range fs {
		if strings.HasSuffix(f.Path, "/") {
			if err := os.Mkdir(filepath.Join(path, f.Path), os.ModeDir | 0755); err != nil {
				log.Fatalf("link: %v", err)
			}
		} else {
			l, err := filepath.Abs(mvb.GetObjectPath(f.DataDigest))
			if err != nil {
				log.Fatalf("link: %v", err)
			}
			if err := os.Symlink(l, filepath.Join(path, f.Path)); err != nil {
				log.Fatalf("link: %v", err)
			}
		}
	}
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
	if len(os.Args) < 2 {
		message()
		return
	}
	switch os.Args[1] {
	case "init":
		initialize(os.Args[2])
	case "backup":
		backup()
	case "restore":
		switch len(os.Args) {
		case 3:
			restore(os.Args[2], "")
		case 4:
			restore(os.Args[2], os.Args[3])
		default:
			message()
		}
	case "link":
		if len(os.Args) < 4 {
			message()
		} else {
			link(os.Args[2], os.Args[3])
		}
	case "list":
		list()
	case "delete":
		delete(os.Args[2])
	case "gc":
		gc()
	default:
		message()
	}
}
