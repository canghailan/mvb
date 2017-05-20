package mvb

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const ISO8601 = "20060102150405-0700"
const EMPTY_SIZE = "                   "
const EMPTY_SHA1 = "                                        "
const VERSION = "da39a3ee5e6b4b0d3255bfef95601890afd80709 20060102150405-0700\n"
const VERSION_LEN = len(VERSION)

type Version struct {
	Sha1      string
	Timestamp string
}

type FileMetadata struct {
	Path    string
	ModTime string
	Size    string
	Sha1    string
}

type DiffFileMetadata struct {
	FileMetadata
	Type string
}

type FileMetadataSlice []FileMetadata

func (s FileMetadataSlice) Len() int           { return len(s) }
func (s FileMetadataSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FileMetadataSlice) Less(i, j int) bool { return s[i].Path < s[j].Path }

type DiffFileMetadataSlice []DiffFileMetadata

func (s DiffFileMetadataSlice) Len() int           { return len(s) }
func (s DiffFileMetadataSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s DiffFileMetadataSlice) Less(i, j int) bool { return s[i].Path < s[j].Path }

var ref string

func Sha1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func GetRef() string {
	if ref == "" {
		data, err := ioutil.ReadFile("ref")
		if err != nil {
			Errorf("GetRef: %v", err)
		}
		ref = string(data)
	}
	return ref
}

func GetObjectPath(objectSha1 string) string {
	if len(objectSha1) == 40 {
		return filepath.Join("objects", objectSha1[0:2], objectSha1[2:])
	}
	Errorf("GetObjectPath: %s", objectSha1)
	return ""
}

func GetFileSha1(path string) string {
	f, err := os.Open(path)
	if err != nil {
		Errorf("GetFileSha1: %v", err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		Errorf("GetFileSha1: %v", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func StringifyVersion(version Version) string {
	return fmt.Sprintf("%40s %19s\n", version.Sha1, version.Timestamp)
}

func ParseVersion(text string) Version {
	return Version{Sha1: text[:40], Timestamp: text[41:]}
}

func StringifyVersionObject(files []FileMetadata) string {
	var buffer bytes.Buffer
	for _, f := range files {
		buffer.WriteString(StringifyFileMetadata(f))
	}
	return buffer.String()
}

func ParseVersionObject(o string) (files []FileMetadata) {
	if len(o) == 0 {
		return files
	}
	for _, f := range strings.Split(string(o[:len(o)-1]), "\n") {
		files = append(files, ParseFileMetadata(f))
	}
	return files
}

func StringifyFileMetadata(file FileMetadata) string {
	return fmt.Sprintf("%40s %19s %19s %s\n", file.Sha1, file.ModTime, file.Size, file.Path)
}

func ParseFileMetadata(text string) FileMetadata {
	return FileMetadata{Sha1: text[:40], ModTime: text[41:60], Size: text[61:80], Path: text[81:]}
}
