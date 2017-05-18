package mvb

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"fmt"
	"strings"
	"sync"
)

func IsObjectExist(id string) bool {
	if _, err := os.Stat(GetObjectPath(id)); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			log.Fatalf("IsObjectExist: %v", err)
		}
	}
	return true
}

func CopyFileObjects(fs []FileObject)  {
	var wg sync.WaitGroup
	sem := make(chan int, 4)
	for _, f := range fs {
		wg.Add(1)
		sem <- 1
		go func(f FileObject) {
			CopyFileObject(f)
			<-sem
			wg.Done()
		}(f)
	}
	wg.Wait()
	close(sem)
}

func CopyFileObject(f FileObject) {
	id := f.DataDigest
	if strings.HasSuffix(f.Path, "/") {
		return
	}
	if IsObjectExist(id) {
		fmt.Printf("copy & skip %s\n", f.Path)
		return
	}

	src := filepath.Join(GetRef(), f.Path)
	dst := GetObjectPath(id)

	if err := os.MkdirAll(filepath.Dir(dst), os.ModeDir|0774); err != nil {
		log.Fatalf("CopyFileObject: %v", err)
	}

	w, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("CopyFileObject: %v", err)
	}
	defer w.Close()

	r, err := os.Open(src)
	if err != nil {
		log.Fatalf("CopyFileObject: %v", err)
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		log.Fatalf("CopyFileObject: %v", err)
	}

	fmt.Printf("copy %s\n", f.Path)
}
