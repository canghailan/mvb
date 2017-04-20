package mvb

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
	"fmt"
	"strings"
	"sync"
	"bufio"
	"strconv"
	"sort"
)

const ObjectId0 = "0000000000000000000000000000000000000000" // hex, 20bits

type FileObject struct {
	ObjectId  string
	ObjectKey string
	ModTime time.Time
	Size int64
}

type FileObjectsDiff struct {
	Source []FileObject
	Target []FileObject
	Add    []FileObject
	Delete []FileObject
	Modify []FileObject
}

type FileObjectSlice []FileObject

func (s FileObjectSlice) Len() int           { return len(s) }
func (s FileObjectSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FileObjectSlice) Less(i, j int) bool { return s[i].ObjectKey < s[j].ObjectKey }

func GetFileObjectId(path string) string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func IsObjectExist(objectId string) bool {
	return IsFileExist(GetObjectPath(objectId))
}

func ParseFileObject(line string) *FileObject {
	objectId := line[:40]
	size, err := strconv.ParseInt(strings.TrimSpace(line[41:60]), 10, 64)
	if err != nil {
		panic(err)
	}
	modTime,err := time.Parse(ISO8601, line[61:80])
	if err != nil {
		panic(err)
	}
	objectKey := line[81:len(line) - 1]
	return &FileObject{ObjectId: objectId, ObjectKey: objectKey, ModTime:modTime,Size:size}
}

func CopyFileObjects() {
	src := GetLinkOfNowPath()
	r := NewFileObjectListReader()
	defer r.Close()
	for {
		f := r.ReadFileObject()
		if f == nil {
			break
		}
		if strings.HasSuffix(f.ObjectKey, "/") {
			continue
		}
		if IsObjectExist(f.ObjectId) {
			continue
		}
		CopyFile(filepath.Join(src, f.ObjectKey), GetObjectPath(f.ObjectId))
	}
}

func WriteObject(base string, objectId string, data string) {
	objectPath := GetObjectPath(objectId)
	if err := os.MkdirAll(filepath.Dir(objectPath), DefaultDirPerm); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(objectPath, []byte(data), DefaultObjectPerm); err != nil {
		panic(err)
	}
	log.Printf("%s (add)", objectId)
}

func DiffFileObjects(source []FileObject, target []FileObject) {

}

type FileObjectListReader struct {
	io.Closer
	f *os.File
	r *bufio.Reader
}

func NewFileObjectListReader() *FileObjectListReader {
	f, err := os.Open(GetListPath())
	if err != nil {
		panic(err)
	}
	return &FileObjectListReader{f: f, r: bufio.NewReader(f)}
}

func (ir *FileObjectListReader) ReadFileObject() *FileObject {
	line, err := ir.r.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil
		} else {
			panic(err)
		}
	}
	return ParseFileObject(line)
}

func (ir *FileObjectListReader) Close() error {
	return ir.f.Close()
}

type FileObjectListWriter struct {
	io.Closer
	f *os.File
}

func NewFileObjectListWriter() *FileObjectListWriter {
	f, err := os.OpenFile(GetListPath(), os.O_CREATE | os.O_TRUNC | os.O_WRONLY, DefaultPerm)
	if err != nil {
		panic(err)
	}
	return &FileObjectListWriter{f: f}
}

func (ir *FileObjectListWriter) WriteFileObject(fileObject *FileObject)  {
	_, err := fmt.Fprintf(ir.f, "%s %19d %s %s\n",
		fileObject.ObjectId,
		fileObject.Size,
		fileObject.ModTime.Format(ISO8601),
		fileObject.ObjectKey)
	if err != nil {
		panic(err)
	}
}

func (ir *FileObjectListWriter) Close() error {
	return ir.f.Close()
}

func MakeFileObjectList()  {
	root, err := filepath.EvalSymlinks(GetLinkOfNowPath())
	if err != nil {
		panic(err)
	}

	w := NewFileObjectListWriter()
	defer w.Close()

	var wg sync.WaitGroup
	filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		wg.Add(1)
		go func() {
			if f.IsDir() {
				w.WriteFileObject(&FileObject{
					ObjectId:ObjectId0,
					Size:0,
					ModTime:f.ModTime(),
					ObjectKey:rel + "/"})
			} else {
				w.WriteFileObject(&FileObject{
					ObjectId:GetFileObjectId(path),
					Size:f.Size(),
					ModTime:f.ModTime(),
					ObjectKey:rel})
			}
			wg.Done()
		}()
		return nil
	})
	wg.Wait()
	SortFileObjectList()
}

func SortFileObjectList()  {
	var fileObjects FileObjectSlice
	func() {
		r := NewFileObjectListReader()
		defer r.Close()
		for {
			f := r.ReadFileObject()
			if f == nil {
				break
			}
			fileObjects = append(fileObjects, *f)
		}
	}()
	sort.Sort(fileObjects)

	w := NewFileObjectListWriter()
	defer w.Close()

	for _, f := range fileObjects {
		w.WriteFileObject(&f)
	}
}