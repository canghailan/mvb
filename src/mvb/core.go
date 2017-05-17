package mvb

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ISO8601 = "20060102150405-0700"
const EMPTY_DIGEST = "                                        "
const DIGEST_LEN  = len(EMPTY_DIGEST)
const VERSION_LINE = "da39a3ee5e6b4b0d3255bfef95601890afd80709 20060102150405-0700\n"
const VERSION_LINE_LEN  = len(VERSION_LINE)

type FileObject struct {
	Path string
	DataDigest string
	MetadataDigest string
}

type FileObjectSlice []FileObject

func (s FileObjectSlice) Len() int           { return len(s) }
func (s FileObjectSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FileObjectSlice) Less(i, j int) bool { return s[i].Path < s[j].Path }

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
			log.Fatalf("GetRef: %v", err)
		}
		ref = string(data)
	}
	return ref
}

func GetObjectPath(id string) string {
	return filepath.Join("objects", id[0:2], id[2:])
}

func GetFileDataDigest(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("GetFileHash: %v", err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatalf("GetFileHash: %v", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func GetFileMetadataDigest(key string, fileInfo os.FileInfo) string {
	modTime := fileInfo.ModTime().Format(ISO8601)
	size := strconv.FormatInt(fileInfo.Size(), 10)
	return Sha1([]byte(strings.Join([]string{key, modTime, size}, "\n")))
}
