package mvb

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"strconv"
)

type ReverseIndex struct {
	io.Closer
	index *os.File
	offset int64
}

func NewReverseIndex() (*ReverseIndex, error) {
	fi, err := os.Stat("index")
	if err != nil {
		if os.IsNotExist(err) {
			return &ReverseIndex{index:nil, offset:0}, nil
		}
	}
	f, err := os.Open("index")
	if err != nil {
		return nil, err
	}
	return &ReverseIndex{index:f, offset:fi.Size()}, nil
}

func (ri *ReverseIndex) Close() error {
	if ri.index != nil {
		return ri.index.Close()
	}
	return nil
}

func (ri *ReverseIndex) NextVersionRecord() string {
	ri.offset -= int64(VERSION_RECORD_LEN)
	if ri.offset >= 0 {
		buffer := make([]byte, VERSION_RECORD_LEN)
		_, err := ri.index.ReadAt(buffer, ri.offset)
		if err != nil {
			Errorf("读取索引文件错误：%v", err)
		}
		return string(buffer[:VERSION_RECORD_LEN-1])
	}
	return ""
}

func GetIndexVersionRecordCount() int {
	fi, err := os.Stat("index")
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		} else {
			Errorf("GetIndexVersionRecordCount: %v", err)
		}
	}
	return int(fi.Size() / int64(VERSION_RECORD_LEN))
}


func GetIndexVersionRecordAt(i int) string {
	f, err := os.Open("index")
	if err != nil {
		Errorf("GetIndexVersionAt: %v", err)
	}

	buf := make([]byte, VERSION_RECORD_LEN-1)
	if _, err = f.ReadAt(buf, int64(i)*int64(VERSION_RECORD_LEN)); err != nil {
		Errorf("GetIndexVersionAt: %v", err)
	}
	return string(buf)
}

func GetIndexVersionAt(i int) string {
	return GetIndexVersionRecordAt(i)[:DIGEST_LEN]
}

func AddVersionRecordToIndex(id string, t time.Time) string {
	f, err := os.OpenFile("index", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		Errorf("AddVersionRecordToIndex: %v", err)
	}
	defer f.Close()

	line := id + " " + t.Format(ISO8601) + "\n"
	f.WriteString(line)

	return id
}

func GetLatestVersion() string {
	n := GetIndexVersionRecordCount()
	if n == 0 {
		Errorf("无最新版本")
	}
	return GetIndexVersionAt(n - 1)
}

func FindIndexVersionRecordAt(a string) string {
	i, err := strconv.Atoi(a)
	if err != nil {
		Errorf("版本格式错误：%s", a)
	}
	i += 1
	if i > 0 {
		return GetIndexVersionRecordAt(i - 1)
	} else {
		n := GetIndexVersionRecordCount()
		return GetIndexVersionRecordAt(n  + i- 1)
	}
}

func FindIndexVersionRecords(pattern string) (r []string) {
	f, err := os.Open("index")
	if err != nil {
		if os.IsNotExist(err) {
			return r
		} else {
			Errorf("FindIndexVersion: %v", err)
		}
	}
	defer f.Close()

	buffer := make([]byte, VERSION_RECORD_LEN)
	for {
		_, err := io.ReadFull(f, buffer)
		if err != nil {
			if err == io.EOF {
				return r
			} else {
				Errorf("FindIndexVersion: %v", err)
			}
		}

		v := string(buffer[:VERSION_RECORD_LEN- 1])
		if MatchVersionRecord(pattern, v) {
			r = append(r, v)
		}
	}
}

func ResolveVersions(pattern string) []string {
	if strings.HasPrefix(pattern, "v") {
		return []string{FindIndexVersionRecordAt(pattern[1:])[:DIGEST_LEN]}
	} else {
		var versions []string
		for _, v := range FindIndexVersionRecords(pattern) {
			versions = append(versions, v[:DIGEST_LEN])
		}
		return versions
	}
}

func ResolveVersion(pattern string) string {
	versions := ResolveVersions(pattern)
	if len(versions) == 0 {
		Errorf("未找到对应的版本：%s", pattern)
	}
	if len(versions) > 1 {
		Errorf("找到多个版本，请输入更精确的版本号：%s", pattern)
	}
	return versions[0]
}

func ToVersionSnapshot(fileObjects []FileObject) string {
	var buffer bytes.Buffer
	for _, f := range fileObjects {
		buffer.WriteString(f.DataDigest)
		buffer.WriteString(" ")
		buffer.WriteString(f.MetadataDigest)
		buffer.WriteString(" ")
		buffer.WriteString(f.Path)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func GetVersionFileObjects(version string) (files []FileObject) {
	data, err := ioutil.ReadFile(GetObjectPath(version))
	if err != nil {
		Errorf("GetVersionFileObjects: %v", err)
	}
	for _, line := range strings.Split(string(data[:len(data)-1]), "\n") {
		files = append(files, FileObject{Path: line[82:], DataDigest: line[:40], MetadataDigest: line[41:81]})
	}
	return files
}

func WriteVersionFile(id string, snapshot string) {
	path := GetObjectPath(id)
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0774); err != nil {
		Errorf("AddVersionRecordToIndex: %v", err)
	}
	if err := ioutil.WriteFile(path, []byte(snapshot), 0644); err != nil {
		Errorf("AddVersionRecordToIndex: %v", err)
	}
}

func GetFileObjects(root string) []FileObject {
	var files FileObjectSlice
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			Errorf("GetFileObjects: %v", err)
		}

		p, err := filepath.Rel(root, path)
		if err != nil {
			Errorf("GetFileObjects: %v", err)
		}
		if p == "." {
			return nil
		}
		p = filepath.ToSlash(p)

		if fi.IsDir() {
			p = p + "/"
			files = append(files, FileObject{Path: p, MetadataDigest: EMPTY_DIGEST, DataDigest: EMPTY_DIGEST})
		} else {
			fmd := GetFileMetadataDigest(p, fi)
			files = append(files, FileObject{Path: p, MetadataDigest: fmd})
		}

		return nil
	})
	sort.Sort(files)
	return files
}

func GetRefFileObjects() []FileObject {
	fileObjects := GetFileObjects(GetRef())

	n := GetIndexVersionRecordCount()
	if n > 0 {
		lastVersion := GetIndexVersionAt(n - 1)
		lastVersionFileObjects := GetVersionFileObjects(lastVersion)
		FastDigestFileObjects(fileObjects, lastVersionFileObjects)
	}

	DigestFileObjects(fileObjects)

	return fileObjects
}

func MatchVersion(pattern string, version string) bool  {
	return strings.HasPrefix(version, pattern)
}

func MatchVersionRecord(pattern string, record string) bool {
	return MatchVersion(pattern, record[:40]) || MatchVersion(pattern, record[41:])
}