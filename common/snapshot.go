package mvb

import (
	"bufio"
	"io"
	"os"
	"time"
	"path/filepath"
)

func ParseSnapshotFileObject(line string) *FileObject {
	objectId := line[:40]
	objectKey := line[41:len(line) - 1]
	return &FileObject{ObjectId: objectId, ObjectKey: objectKey}
}

func StringifySnapshotFileObject(fileObject *FileObject) string {
	return fileObject.ObjectId + " " + fileObject.ObjectKey + "\n"
}

type SnapshotReader struct {
	io.Closer
	f *os.File
	r *bufio.Reader
}

func NewSnapshotReader(path string) *SnapshotReader {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	return &SnapshotReader{f: f, r: bufio.NewReader(f)}
}

func (ir *SnapshotReader) ReadFileObject() *FileObject {
	line, err := ir.r.ReadString('\n')
	if err == io.EOF {
		return nil
	} else {
		panic(err)
	}
	return ParseSnapshotFileObject(line)
}

func (ir *SnapshotReader) Close() error {
	return ir.f.Close()
}

type SnapshotWriter struct {
	io.Closer
	f *os.File
}

func NewSnapshotWriter(path string) *SnapshotWriter {
	f, err := os.OpenFile(path, os.O_CREATE | os.O_TRUNC | os.O_WRONLY, DefaultPerm)
	if err != nil {
		panic(err)
	}
	return &SnapshotWriter{f: f}
}

func (iw *SnapshotWriter) WriteFileObject(fileObject *FileObject) error {
	_, err := iw.f.WriteString(StringifySnapshotFileObject(fileObject))
	return err
}

func (iw *SnapshotWriter) Close() error {
	return iw.f.Close()
}

type Snapshot struct {
	ObjectId  string
	Timestamp time.Time
}

func CreateSnapshot(at time.Time) *Snapshot {
	temp := GetPath(".snapshot")
	func() {
		r := NewFileObjectListReader()
		defer r.Close()
		w := NewSnapshotWriter(temp)
		defer w.Close()
		for {
			f := r.ReadFileObject()
			if f == nil {
				break
			}
			w.WriteFileObject(f)
		}
	}()

	objectId := GetFileObjectId(temp)
	if IsObjectExist(objectId) {
		return &Snapshot{ObjectId:objectId}
	}

	objectPath := GetObjectPath(objectId)
	if err := os.Mkdir(filepath.Dir(objectPath), DefaultDirPerm); err != nil {
		panic(err)
	}
	if err := os.Rename(temp, objectPath); err != nil {
		panic(err)
	}

	return &Snapshot{ObjectId: objectId, Timestamp: at}
}