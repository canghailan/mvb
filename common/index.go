package mvb

import (
	"bufio"
	"io"
	"os"
	"time"
)

const lineLen = len(ObjectId0 + " " + ISO8601 + "\n")

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

type IndexReader struct {
	io.Closer
	file      *os.File
	direction int
	offset    int64
	buffer    []byte
}

func (ir *IndexReader) forward(n int) {
	ir.offset += int64(ir.direction * n * lineLen)
}

func (ir *IndexReader) SkipSnapshot(n int) {
	ir.forward(n)
}

func (ir *IndexReader) ReadSnapshot() *Snapshot {
	ir.forward(1)
	if ir.offset < 0 {
		return nil
	}
	n, err := ir.file.ReadAt(ir.buffer, ir.offset)
	if err != nil {
		panic(err)
	}
	if n != lineLen {
		panic("")
	}
	return ParseSnapshot(string(ir.buffer))
}

func (ir *IndexReader) ReadSnapshots(limit int) []Snapshot {
	if limit <= 0 {
		limit = int(^uint(0) >> 1) // max int
	}
	var snapshots []Snapshot
	for {
		s := ir.ReadSnapshot()
		if s == nil {
			return snapshots
		}
		snapshots = append(snapshots, *s)
		if len(snapshots) >= limit {
			return snapshots
		}
	}
}

func NewIndexReader(direction int) *IndexReader {
	f, err := os.Open(GetIndexPath())
	if err != nil {
		panic(err)
	}
	offset := int64(0)
	if direction < 0 {
		offset, err = f.Seek(0, os.SEEK_END)
		if err != nil {
			panic(err)
		}
	}
	return &IndexReader{file: f, direction: direction, offset: offset, buffer: make([]byte, lineLen)}
}

func (ir *IndexReader) Close() error {
	return ir.file.Close()
}

type IndexWriter struct {
	io.Closer
	file   *os.File
	writer *bufio.Writer
}

func NewIndexWriter() *IndexWriter {
	f, err := os.OpenFile(GetIndexPath(), os.O_WRONLY, DefaultPerm)
	if err != nil {
		panic(err)
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		panic(err)
	}
	return &IndexWriter{file: f, writer: bufio.NewWriter(f)}
}

func (iw *IndexWriter) WriteSnapshot(snapshot *Snapshot) {
	if _, err := iw.writer.WriteString(StringifySnapshot(snapshot)); err != nil {
		panic(err)
	}
}

func (iw *IndexWriter) Flush() {
	if err :=  iw.writer.Flush(); err != nil {
		panic(err)
	}
}

func (iw *IndexWriter) Close() error {
	if err :=  iw.writer.Flush(); err != nil {
		panic(err)
	}
	return iw.file.Close()
}
