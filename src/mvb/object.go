package mvb

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sort"
)

const MAX_GOS = 4

func IsObjectExist(id string) bool {
	if _, err := os.Stat(GetObjectPath(id)); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			Errorf("IsObjectExist: %v", err)
		}
	}
	return true
}

func CopyObject(id string, w *os.File)  {
	r, err := os.Open(GetObjectPath(id))
	if err != nil {
		Errorf("CopyObject: %v", err)
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		Errorf("CopyObject: %v", err)
	}
}

func FastDigestFileObjects(fileObjects []FileObject, cachedFileObjects []FileObject)  {
	for i := range fileObjects {
		f := SearchFileObjects(cachedFileObjects, fileObjects[i].Path)
		if f != nil && f.MetadataDigest == fileObjects[i].MetadataDigest {
			fileObjects[i].DataDigest = f.DataDigest
		}
	}
}

func DigestFileObjects(fileObjects []FileObject) {
	var wg sync.WaitGroup
	sem := make(chan int, MAX_GOS)
	for i := range fileObjects {
		f := &fileObjects[i]
		if f.MetadataDigest == "" || f.DataDigest == "" {
			sem <- 1
			wg.Add(1)
			go func() {
				DigestFileObject(f)
				wg.Done()
				<-sem
			}()
		}
	}
	wg.Wait()
	close(sem)
}

func DigestFileObject(f *FileObject)  {
	if strings.HasSuffix(f.Path, "/") {
		f.MetadataDigest = EMPTY_DIGEST
		f.DataDigest = EMPTY_DIGEST
		return
	}

	path := filepath.Join(GetRef(), f.Path)
	if f.MetadataDigest == "" {
		fi, err := os.Stat(path)
		if err != nil {
			Errorf("DigestFileObject: %v", err)
		}
		f.MetadataDigest = GetFileMetadataDigest(f.Path, fi)
	}
	if f.DataDigest == "" {
		f.DataDigest = GetFileDataDigest(path)
	}
}

func SearchFileObjects(fileObjects []FileObject, path string) *FileObject {
	start := 0
	end := len(fileObjects)
	for start <= end {
		mid := start + (end - start) / 2
		if fileObjects[mid].Path < path {
			start = mid + 1
		} else if fileObjects[mid].Path > path {
			end = mid - 1
		} else {
			return &fileObjects[mid]
		}
	}
	return nil
}

func DiffFileObjects(from []FileObject, to []FileObject) []DiffFileObject {
	var diffFileObjects DiffFileObjectSlice
	for _, f := range to {
		pf := SearchFileObjects(from, f.Path)
		if pf == nil {
			diffFileObjects = append(diffFileObjects, DiffFileObject{Type: "+", FileObject: f})
		} else if pf.MetadataDigest != f.MetadataDigest {
			diffFileObjects = append(diffFileObjects, DiffFileObject{Type: "*", FileObject: f})
		}
	}
	for _, f := range from {
		pf := SearchFileObjects(to, f.Path)
		if pf == nil {
			diffFileObjects = append(diffFileObjects, DiffFileObject{Type: "-", FileObject: f})
		}
	}
	sort.Sort(diffFileObjects)
	return diffFileObjects
}

func CopyFileObjects(fileObjects []FileObject)  {
	var wg sync.WaitGroup
	sem := make(chan int, MAX_GOS)
	for _, f := range fileObjects {
		sem <- 1
		wg.Add(1)
		go func(f FileObject) {
			CopyFileObject(f)
			wg.Done()
			<-sem
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
		Verbosef("copy & skip %s\n", f.Path)
		return
	}

	src := filepath.Join(GetRef(), f.Path)
	dst := GetObjectPath(id)
	CopyFile(src, dst)

	Verbosef("copy %s\n", f.Path)
}

func CopyFile(src string, dst string)  {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModeDir|0774); err != nil {
		Errorf("Copy: %v", err)
	}

	w, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		Errorf("Copy: %v", err)
	}
	defer w.Close()

	r, err := os.Open(src)
	if err != nil {
		Errorf("Copy: %v", err)
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		Errorf("Copy: %v", err)
	}
}