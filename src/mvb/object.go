package mvb

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func IsObjectExist(objectSha1 string) bool {
	if _, err := os.Stat(GetObjectPath(objectSha1)); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			Errorf("IsObjectExist: %v", err)
		}
	}
	return true
}

func CopyObjects(files []FileMetadata) {
	var wg sync.WaitGroup
	sem := make(chan int, MAX_GOS)
	for i := range files {
		sem <- 1
		wg.Add(1)
		go func(f *FileMetadata) {
			CopyObject(f)
			wg.Done()
			<-sem
		}(&files[i])
	}
	wg.Wait()
	close(sem)
}

func CopyObject(file *FileMetadata) {
	if strings.HasSuffix(file.Path, "/") {
		return
	}
	if IsObjectExist(file.Sha1) {
		Verbosef("文件已存在： %s %s\n", file.Sha1, file.Path)
		return
	}

	src := filepath.Join(GetRef(), file.Path)
	dst := GetObjectPath(file.Sha1)
	CopyFile(src, dst)

	Verbosef("保存成功： %s\n", file.Path)
}

func WriteObjectTo(objectSha1 string, w *os.File) {
	r, err := os.Open(GetObjectPath(objectSha1))
	if err != nil {
		Errorf("WriteObjectTo: %v", err)
	}
	defer r.Close()

	if _, err = io.Copy(w, r); err != nil {
		Errorf("WriteObjectTo: %v", err)
	}
}

func FastGetFilesSha1(files []FileMetadata, sha1Files []FileMetadata) {
	for i := range files {
		f := SearchFile(sha1Files, files[i].Path)
		if f != nil && f.ModTime == files[i].ModTime && f.Size == files[i].Size {
			files[i].Sha1 = f.Sha1
		}
	}
}

func GetFilesSha1(root string, files []FileMetadata) {
	var wg sync.WaitGroup
	sem := make(chan int, MAX_GOS)
	for i := range files {
		f := &files[i]
		if f.Sha1 != "" {
			continue
		}
		if strings.HasSuffix(f.Path, "/") {
			f.Sha1 = EMPTY_SHA1
			continue
		}
		sem <- 1
		wg.Add(1)
		go func(root string, f *FileMetadata) {
			Verbosef("计算SHA1：%s\n", f.Path)
			f.Sha1 = GetFileSha1(filepath.Join(root, f.Path))
			wg.Done()
			<-sem
		}(root, f)
	}
	wg.Wait()
	close(sem)
}

func SearchFile(files []FileMetadata, path string) *FileMetadata {
	if len(files) == 0 {
		return nil
	}

	start := 0
	end := len(files)
	for start <= end {
		mid := start + (end-start)/2
		if files[mid].Path < path {
			start = mid + 1
		} else if files[mid].Path > path {
			end = mid - 1
		} else {
			return &files[mid]
		}
	}
	return nil
}

func DiffFiles(from []FileMetadata, to []FileMetadata) []DiffFileMetadata {
	var diffFileObjects DiffFileMetadataSlice
	for _, f := range to {
		file := SearchFile(from, f.Path)
		if file == nil {
			diffFileObjects = append(diffFileObjects, DiffFileMetadata{Type: "+", FileMetadata: f})
		} else if file.Sha1 != f.Sha1 {
			diffFileObjects = append(diffFileObjects, DiffFileMetadata{Type: "*", FileMetadata: f})
		}
	}
	for _, f := range from {
		file := SearchFile(to, f.Path)
		if file == nil {
			diffFileObjects = append(diffFileObjects, DiffFileMetadata{Type: "-", FileMetadata: f})
		}
	}
	sort.Sort(diffFileObjects)
	return diffFileObjects
}

func CopyFile(src string, dst string) {
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

	// ignore error
	if fi, err := r.Stat(); err == nil {
		os.Chtimes(dst, time.Now(), fi.ModTime())
	}
}
