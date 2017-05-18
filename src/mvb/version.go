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
	"fmt"
)

func WriteVersionFile(id string, version string)  {
	path := GetObjectPath(id)
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir | 0774); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
	if err := ioutil.WriteFile(path, []byte(version), 0644); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
}

func GetVersionFileObjects(version string) (files []FileObject) {
	data, err := ioutil.ReadFile(GetObjectPath(version))
	if err != nil {
		log.Fatalf("GetVersionFileObjects: %v", err)
	}
	for _, line := range strings.Split(string(data[:len(data) - 1]), "\n") {
		files = append(files, FileObject{Path: line[82:], DataDigest: line[:40], MetadataDigest: line[41:81]})
	}
	return files
}

func GetFileObjects(root string) []FileObject {
	cache := getFastCache()

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
			log.Fatalf("GetFileObjects: %v", err)
		}

		p, err := filepath.Rel(root, path)
		if err != nil {
			log.Fatalf("GetFileObjects: %v", err)
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
				fdd := cache[fmd]
				if fdd == "" {
					fmt.Printf("scan %s\n", p)
					fdd = GetFileDataDigest(path)
				} else {
					fmt.Printf("scan & skip %s\n", p)
				}
				ch <- &FileObject{Path: p, DataDigest: fdd, MetadataDigest: fmd}
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

func GetFileObjectsWithoutDataDigest(root string) []FileObject {
	var files FileObjectSlice
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("GetFileObjectsWithoutDataDigest: %v", err)
		}

		p, err := filepath.Rel(root, path)
		if err != nil {
			log.Fatalf("GetFileObjectsWithoutDataDigest: %v", err)
		}
		if p == "." {
			return nil
		}
		p = filepath.ToSlash(p)

		if fi.IsDir() {
			files = append(files, FileObject{Path: p + "/", MetadataDigest: EMPTY_DIGEST})
		} else {
			fmd := GetFileMetadataDigest(p, fi)
			files = append(files, FileObject{Path: p + "/", MetadataDigest: fmd})
		}

		return nil
	})
	sort.Sort(files)
	return files
}

func getFastCache() map[string]string {
	cache := map[string]string{}

	n := GetIndexVersionCount()
	if n == 0 {
		return cache
	}

	v := GetIndexVersionAt(n - 1)

	for _, f := range GetVersionFileObjects(v) {
		if f.MetadataDigest != EMPTY_DIGEST {
			cache[f.MetadataDigest] = f.DataDigest
		}
	}
	return cache
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

func DiffFileObjects(src []FileObject, dst []FileObject) []FileDiff {
	return nil
}