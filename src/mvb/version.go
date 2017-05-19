package mvb

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func GetIndexVersionCount() int {
	fi, err := os.Stat("index")
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		} else {
			log.Fatalf("GetIndexVersionCount: %v", err)
		}
	}
	return int(fi.Size() / int64(VERSION_LINE_LEN))
}

func GetIndexVersionAt(i int) string {
	f, err := os.Open("index")
	if err != nil {
		log.Fatalf("GetIndexVersionAt: %v", err)
	}

	buf := make([]byte, DIGEST_LEN)
	if _, err = f.ReadAt(buf, int64(i)*int64(VERSION_LINE_LEN)); err != nil {
		log.Fatalf("GetIndexVersionAt: %v", err)
	}
	return string(buf)
}

func AddVersionToIndex(id string, t time.Time) string {
	f, err := os.OpenFile("index", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
	defer f.Close()

	line := id + " " + t.Format(ISO8601) + "\n"
	f.WriteString(line)

	return id
}

func FindIndexVersions(versionPattern string) (r []string) {
	f, err := os.Open("index")
	if err != nil {
		if os.IsNotExist(err) {
			return r
		} else {
			log.Fatalf("FindIndexVersion: %v", err)
		}
	}
	defer f.Close()

	buffer := make([]byte, VERSION_LINE_LEN)
	for {
		_, err := io.ReadFull(f, buffer)
		if err != nil {
			if err == io.EOF {
				return r
			} else {
				log.Fatalf("FindIndexVersion: %v", err)
			}
		}

		line := string(buffer)
		version := line[:40]
		timestamp := line[41:]
		if strings.HasPrefix(version, versionPattern) || strings.HasPrefix(timestamp, versionPattern) {
			r = append(r, version)
		}
	}
}

func ResolveVersion(version string) string {
	versions := FindIndexVersions(version)
	if len(versions) == 0 {
		log.Fatal("ResolveVersion: version not found")
	}
	if len(versions) > 1 {
		log.Fatal("ResolveVersion: too many versions")
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
		log.Fatalf("GetVersionFileObjects: %v", err)
	}
	for _, line := range strings.Split(string(data[:len(data)-1]), "\n") {
		files = append(files, FileObject{Path: line[82:], DataDigest: line[:40], MetadataDigest: line[41:81]})
	}
	return files
}

func WriteVersionFile(id string, snapshot string) {
	path := GetObjectPath(id)
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0774); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
	if err := ioutil.WriteFile(path, []byte(snapshot), 0644); err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
}

func GetFileObjects(root string) []FileObject {
	var files FileObjectSlice
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

	n := GetIndexVersionCount()
	if n > 0 {
		lastVersion := GetIndexVersionAt(n - 1)
		lastVersionFileObjects := GetVersionFileObjects(lastVersion)
		FastDigestFileObjects(fileObjects, lastVersionFileObjects)
	}

	DigestFileObjects(fileObjects)

	return fileObjects
}