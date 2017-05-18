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
	fs := mvb.GetFileObjects(mvb.GetRef())
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

// mvb check
func check()  {
	n := mvb.GetIndexVersionCount()
	v := mvb.GetIndexVersionAt(n - 1)
	fs := mvb.GetVersionFileObjects(v)
	for _, f := range fs {
		if strings.HasSuffix(f.Path, "/") {
			continue
		}

		src := filepath.Join(mvb.GetRef(), f.Path)
		dst := mvb.GetObjectPath(f.DataDigest)

		s, err := os.Stat(src)
		if err != nil {
			log.Fatalf("check: %v", err)
		}
		d, err := os.Stat(dst)
		if err != nil {
			log.Fatalf("check: %v", err)
		}

		if s.Size() != d.Size() {
			fmt.Printf("%s %s\n", dst, f.Path)
		}
	}
}

// mvb restore [version] [path]
func restore(version string, root string) {
	version = mvb.ResolveVersion(version)
	if root == "" {
		root = mvb.GetRef()
	}

	src := mvb.GetVersionFileObjects(version)
	dst := mvb.GetFileObjectsWithoutDataDigest(root)
	diff := mvb.DiffFileObjects(src, dst)
	for _, f := range diff {
		println(f.Path)
	}
}

// mvb link [version] [path]
func link(version string, path string) {
	fs := mvb.GetVersionFileObjects(mvb.ResolveVersion(version))
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

// mvb list [versions]
func find(versions string)  {
	for _, v := range mvb.FindIndexVersions(versions) {
		os.Stdout.WriteString(v)
	}
}

// mvb diff [version a] [version b]
func diff(a string, b string)  {

}

// mvb delete [version]
func del(version string) {
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
mvb check
mvb restore [version] [path]
mvb link [version] [path]
mvb list
mvb find [versions]
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
		if len(os.Args) < 3 {
			message()
			return
		}
		initialize(os.Args[2])
	case "backup":
		backup()
	case "check":
		check()
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
			return
		}
		link(os.Args[2], os.Args[3])
	case "list":
		list()
	case "find":
		if len(os.Args) < 3 {
			message()
			return
		}
		find(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			message()
			return
		}
		del(os.Args[2])
	case "gc":
		gc()
	default:
		message()
	}
}
