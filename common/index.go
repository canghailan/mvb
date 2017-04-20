package mvb

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const ISO8601 = "20060102150405-0700"
const lineLen = len(ObjectId0 + " " + ISO8601 + "\n")

func GetIndexPath(base string) string {
	return filepath.Join(base, "index")
}

func ParseSnapshot(line string) *Snapshot {
	objectId := line[:40]
	timestamp, err := time.Parse(ISO8601, line[41:len(line)-1])
	if err != nil {
		panic(err)
	}
	return &Snapshot{ObjectId: objectId, Timestamp: timestamp}
}

func StringifySnapshot(snapshot *Snapshot) string {
	return snapshot.ObjectId + " " + snapshot.Timestamp.Format(ISO8601) + "\n"
}

type IndexReader interface {
	ReadSnapshot() (*Snapshot, error)
	ReadSnapshots(n int) ([]Snapshot, error)
}

type AbstractIndexReader struct {
	IndexReader
}

func (ir *AbstractIndexReader) ReadSnapshots(limit int) ([]Snapshot, error) {
	if limit < 0 {
		limit = int(^uint(0) >> 1) // max int
	}
	var snapshots []Snapshot
	for {
		s, err := ir.ReadSnapshot()
		if err != nil {
			if err == io.EOF {
				return snapshots, nil
			} else {
				return nil, err
			}
		}
		snapshots = append(snapshots, *s)
		if len(snapshots) >= limit {
			return snapshots, nil
		}
	}
	return snapshots, nil
}

type IndexDefaultReader struct {
	AbstractIndexReader
	io.Closer
	f *os.File
	r *bufio.Reader
}

func NewIndexReader(base string) (*IndexDefaultReader, error) {
	f, err := os.Open(GetIndexPath(base))
	if err != nil {
		return nil, err
	}
	return &IndexDefaultReader{f: f, r: bufio.NewReader(f)}, nil
}

func (ir *IndexDefaultReader) SkipSnapshot(n int) error {
	bytes := n * lineLen
	for {
		discarded, err := ir.r.Discard(bytes)
		if err != nil {
			return err
		}
		bytes -= discarded
		if bytes == 0 {
			return nil
		}
	}
}

func (ir *IndexDefaultReader) ReadSnapshot() (*Snapshot, error) {
	line, err := ir.r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return ParseSnapshot(line), nil
}

func (ir *IndexDefaultReader) Close() error {
	return ir.f.Close()
}

type IndexReverseReader struct {
	AbstractIndexReader
	io.Closer
	f      *os.File
	offset int64
	buf    []byte
}

func NewIndexReverseReader(base string) (*IndexReverseReader, error) {
	f, err := os.Open(GetIndexPath(base))
	if err != nil {
		return nil, err
	}
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	return &IndexReverseReader{f: f, offset: offset, buf: make([]byte, lineLen)}, nil
}

func (ir *IndexReverseReader) SkipSnapshot(n int) error {
	ir.offset -= int64(n * lineLen)
	if ir.offset < 0 {
		return io.EOF
	}
	return nil
}

func (ir *IndexReverseReader) ReadSnapshot() (*Snapshot, error) {
	ir.offset -= int64(lineLen)
	if ir.offset < 0 {
		return nil, io.EOF
	}
	n, err := ir.f.ReadAt(ir.buf, ir.offset)
	if err != nil {
		return nil, err
	}
	if n != lineLen {
		panic("")
	}
	return ParseSnapshot(string(ir.buf)), nil
}

func (ir *IndexReverseReader) Close() error {
	return ir.f.Close()
}

type IndexWriter struct {
	io.Closer
	f *os.File
	w *bufio.Writer
}

func NewIndexWriter(base string) (*IndexWriter, error) {
	f, err := os.OpenFile(GetIndexPath(base), os.O_CREATE|os.O_APPEND, DefaultPerm)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	return &IndexWriter{f: f, w: bufio.NewWriter(f)}, nil
}

func (iw *IndexWriter) WriteSnapshot(snapshot *Snapshot) error {
	_, err := iw.w.WriteString(StringifySnapshot(snapshot))
	return err
}

func (iw *IndexWriter) Flush() error {
	return iw.w.Flush()
}

func (iw *IndexWriter) Close() error {
	if err := iw.w.Flush(); err != nil {
		return err
	}
	return iw.f.Close()
}

func AppendSnapshotToIndex(base string, snapshot *Snapshot) {
	w, err := NewIndexWriter(base)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if err := w.WriteSnapshot(snapshot); err != nil {
		panic(err)
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}
	log.Println("update index")
}
