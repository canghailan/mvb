package mvb

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"time"
)

func ParseFileObject(line string) *FileObject {
	objectId := line[:40]
	objectKey := line[41:]
	return &FileObject{ObjectId: objectId, ObjectKey: objectKey}
}

func StringifyFileObject(fileObject *FileObject) string {
	return fileObject.ObjectId + " " + fileObject.ObjectKey + "\n"
}

func StringifyFileObjects(fileObjects []FileObject) string {
	var buffer bytes.Buffer
	for _, fileObject := range fileObjects {
		buffer.WriteString(fileObject.ObjectId)
		buffer.WriteString(" ")
		buffer.WriteString(fileObject.ObjectKey)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

type SnapshotReader struct {
	io.Closer
	f *os.File
	r *bufio.Reader
}

func NewSnapshotReader(base string, objectId string) (*SnapshotReader, error) {
	f, err := os.Open(base)
	if err != nil {
		return nil, err
	}
	return &SnapshotReader{f: f, r: bufio.NewReader(f)}, nil
}

func (ir *SnapshotReader) ReadFileObject() (*FileObject, error) {
	line, err := ir.r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return ParseFileObject(line), nil
}

func (ir *SnapshotReader) Close() error {
	return ir.f.Close()
}

type SnapshotWriter struct {
	io.Closer
	f *os.File
	w *bufio.Writer
}

func NewSnapshotWriter(f *os.File) (*SnapshotWriter, error) {
	return &SnapshotWriter{f: f, w: bufio.NewWriter(f)}, nil
}

func (iw *SnapshotWriter) WriteSnapshot(snapshot *Snapshot) error {
	_, err := iw.w.WriteString(StringifySnapshot(snapshot))
	return err
}

func (iw *SnapshotWriter) Flush() error {
	return iw.w.Flush()
}

func (iw *SnapshotWriter) Close() error {
	if err := iw.w.Flush(); err != nil {
		return err
	}
	return iw.f.Close()
}

type Snapshot struct {
	ObjectId    string
	Timestamp   time.Time
}