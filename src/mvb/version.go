package mvb

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"bytes"
)

func GetVersionFiles(version string) (files []FileObject) {
	data, err := ioutil.ReadFile(GetObjectPath(version))
	if err != nil {
		log.Fatalf("GetVersionFiles: %v", err)
	}
	for _, line := range strings.Split(string(data[:len(data) - 1]), "\n") {
		files = append(files, FileObject{Path: line[:82], DataDigest: line[:40], MetadataDigest: line[41:81]})
	}
	return files
}

func WriteVersionFile(id string, version string)  {
	path := GetObjectPath(id)
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir | 0774); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
	if err := ioutil.WriteFile(path, []byte(version), 0644); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
}

func getFileHashCache() map[string]*FileObject {
	cache := map[string]*FileObject{}

	n := GetIndexVersionCount()
	if n == 0 {
		return cache
	}

 	v := GetIndexVersionAt(n - 1)

	for _, f := range GetVersionFiles(v) {
		if f.MetadataDigest != EMPTY_DIGEST {
			cache[f.MetadataDigest] = &f
		}
	}
	return cache
}

func GetFiles() []FileObject {
	root := GetRef()
	cache := getFileHashCache()

	var files FileObjectSlice
	var ch = make(chan *FileObject, 1024)

	var wg1 sync.WaitGroup
	wg1.Add(1)
	go func() {
		for f := range ch {
			files = append(files, *f)
		}
		wg1.Done()
	}()

	var wg2 sync.WaitGroup
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("GetFiles: %v", err)
		}

		p, err := filepath.Rel(root, path)
		if err != nil {
			log.Fatalf("GetFiles: %v", err)
		}
		if p == "." {
			return nil
		}
		p = filepath.ToSlash(p)

		wg2.Add(1)
		go func(ch chan *FileObject) {
			if fi.IsDir() {
				ch <- &FileObject{Path: p + "/", DataDigest: EMPTY_DIGEST, MetadataDigest: EMPTY_DIGEST}
			} else {
				fmd := GetFileMetadataDigest(p, fi)
				f := cache[fmd]
				if f == nil || f.Path != p {
					fdd := GetFileDataDigest(path)
					f = &FileObject{Path: p, DataDigest: fdd, MetadataDigest: fmd}
				}
				ch <- f
			}
			wg2.Done()
		}(ch)

		return nil
	})

	wg2.Wait()
	close(ch)
	wg1.Wait()

	sort.Sort(files)
	return files
}

func ToVersionText(files []FileObject) string {
	var buf bytes.Buffer
	for _, f := range files {
		buf.WriteString(f.DataDigest)
		buf.WriteString(" ")
		buf.WriteString(f.MetadataDigest)
		buf.WriteString(" ")
		buf.WriteString(f.Path)
		buf.WriteString("\n")
	}
	return buf.String()
}