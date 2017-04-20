package mvb

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"
)

func Init(base string) {
	if err := os.MkdirAll(GetObjectsPath(base), DefaultDirPerm); err != nil {
		panic(err)
	}
	log.Println("init")
}

func ListSnapshot(base string, prefix string, offset int, limit int) []Snapshot {
	r, err := NewIndexReverseReader(base)
	if CheckError(err, os.IsNotExist) {
		return make([]Snapshot, 0)
	}
	var snapshots []Snapshot
	n := 0
	for {
		s, err := r.ReadSnapshot()
		if CheckError(err, IsEOF) {
			return snapshots
		}
		if prefix == "" ||
			strings.HasPrefix(s.ObjectId, prefix) ||
			strings.HasPrefix(s.Timestamp.Format(ISO8601), prefix) {
			n++
			if n >= offset {
				snapshots = append(snapshots, *s)
			}
		}
		if len(snapshots) >= limit {
			return snapshots
		}
	}
	return snapshots
}

func GetSnapshot(base string, objectId string) *Snapshot {
	r, err := NewIndexReader(base)
	if CheckError(err, os.IsNotExist) {
		return nil
	}
	for {
		s, err := r.ReadSnapshot()
		if CheckError(err, IsEOF) {
			return nil
		}
		if s.ObjectId == objectId {
			return s
		}
	}
}

func WriteSnapshots(snapshots []Snapshot, w *bufio.Writer) {
	for _, s := range snapshots {
		w.WriteString(StringifySnapshot(&s))
	}
	w.Flush()
}

func CreateSnapshot(base string, root string) *Snapshot {
	now := time.Now()
	fileObjects := ListFileObjects(root)
	fileObjectsAsString := StringifyFileObjects(fileObjects)
	objectId := GetObjectId([]byte(fileObjectsAsString))
	if IsObjectExist(base, objectId) {
		return nil
	}
	snapshot := &Snapshot{ObjectId: objectId, Timestamp: now}

	log.Println("copy files")
	CopyFileObjects(base, root, fileObjects)
	log.Println("create snapshot")
	WriteFileObject(base, objectId, fileObjectsAsString)
	AppendSnapshotToIndex(base, snapshot)
	return snapshot
}

func GetSnapshotFileObject(base string, snapshotObjectId string, fileObjectKey string) *FileObject {
	r, err := NewSnapshotReader(base, snapshotObjectId)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	for {
		f, err := r.ReadFileObject()
		if CheckError(err, IsEOF) {
			return nil
		}
		if f.ObjectKey == fileObjectKey {
			return f
		}
	}
}
