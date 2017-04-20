package mvb

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"io/ioutil"
	"log"
)

const ObjectId0 = "0000000000000000000000000000000000000000" // hex, 20bits

type FileObject struct {
	ObjectId  string
	ObjectKey string
}

type FileObjectsDiff struct {
	Source []FileObject
	Target []FileObject
	Add []FileObject
	Delete []FileObject
	Modify []FileObject
}

type FileObjectSlice []FileObject

func (s FileObjectSlice) Len() int           { return len(s) }
func (s FileObjectSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FileObjectSlice) Less(i, j int) bool { return s[i].ObjectKey < s[j].ObjectKey }

func GetObjectId(object []byte) string {
	h := sha1.New()
	_, err := h.Write(object)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

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

func GetObjectsPath(base string) string {
	return filepath.Join(base, "objects")
}

func GetObjectPath(base string, objectId string) string {
	return filepath.Join(base, "objects", objectId[0:2], objectId[2:])
}

func IsObjectExist(base string, objectId string) bool {
	return IsFileExist(GetObjectPath(base, objectId))
}

func ListFileObjects(root string) []FileObject {
	var fileObjects FileObjectSlice
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if f.IsDir() {
			fileObjects = append(fileObjects, FileObject{ObjectId: ObjectId0, ObjectKey: rel + "/"})
		} else {
			fileObjects = append(fileObjects, FileObject{ObjectId: GetFileObjectId(path), ObjectKey: rel})
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Sort(fileObjects)
	return fileObjects
}

func CopyFileObjects(base string, root string, fileObjects []FileObject) {
	for _, f := range fileObjects {
		if strings.HasSuffix(f.ObjectKey, "/") {
			continue
		}
		if IsObjectExist(base, f.ObjectId) {
			continue
		}
		from := filepath.Join(root, f.ObjectKey)
		to := GetObjectPath(base, f.ObjectId)
		log.Printf("%s (add)", f.ObjectId)
		CopyFile(from, to)
	}
}

func WriteFileObject(base string, objectId string, data string) {
	objectPath := GetObjectPath(base, objectId)
	if err := os.MkdirAll(filepath.Dir(objectPath), os.ModeDir); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(objectPath, []byte(data), DefaultPerm); err != nil {
		panic(err)
	}
	log.Printf("%s (add)", objectId)
}

func DiffFileObjects(source []FileObject, target []FileObject)  {

}