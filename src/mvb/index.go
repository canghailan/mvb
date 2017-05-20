package mvb

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
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

func (ri *ReverseIndex) NextVersion() string {
	ri.offset -= int64(VERSION_LEN)
	if ri.offset >= 0 {
		buffer := make([]byte, VERSION_LEN)
		_, err := ri.index.ReadAt(buffer, ri.offset)
		if err != nil {
			Errorf("NextVersion：%v", err)
		}
		return string(buffer[:VERSION_LEN-1])
	}
	return ""
}

func ParseIndexedVersion(a string) int {
	i, err := strconv.Atoi(a[1:])
	if err != nil {
		Errorf("ParseIndexedVersion：%s", a)
	}
	if i > 0 {
		return i - 1
	} else {
		return GetIndexVersionCount() + i
	}
}

func WriteReverseIndexTo(w *os.File)  {
	i, err := NewReverseIndex()
	if err != nil {
		Errorf("WriteReverseIndexTo: %v", err)
	}
	defer i.Close()

	for {
		r := i.NextVersion()
		if r == "" {
			break
		}
		w.WriteString(r)
		w.WriteString("\n")
	}
}

func AddVersionToIndex(version Version) {
	f, err := os.OpenFile("index", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		Errorf("AddVersionToIndex: %v", err)
	}
	defer f.Close()

	f.WriteString(StringifyVersion(version))
}

func DeleteIndexVersionAt(i int)  {
	f, err := os.OpenFile("index", os.O_RDWR, 0644)
	if err != nil {
		Errorf("DeleteIndexVersionAt: %v", err)
	}
	defer f.Close()

	w := int64(i)*int64(VERSION_LEN)
	if _, err := f.Seek(w + int64(VERSION_LEN), os.SEEK_SET); err!= nil {
		Errorf("DeleteIndexVersionAt: %v", err)
	}

	buffer := make([]byte, 4 * 1024)
	for {
		n, err := f.Read(buffer)
		if err != nil {
			if err == io.EOF {
				if err := f.Truncate(w); err != nil {
					Errorf("DeleteIndexVersionAt: %v", err)
				}
				break
			}
		}
		f.WriteAt(buffer[:n], w)
		w += int64(n)
	}
}

func DeleteIndexVersion(pattern string)  {
	f, err := os.OpenFile("index", os.O_RDWR, 0644)
	if err != nil {
		Errorf("DeleteIndexVersion: %v", err)
	}
	defer f.Close()

	r := int64(0)
	w := int64(0)

	buffer := make([]byte, VERSION_LEN)
	for {
		if _, err = f.ReadAt(buffer, r); err != nil {
			if err == io.EOF {
				if err = f.Truncate(w); err != nil {
					Errorf("DeleteIndexVersion: %v", err)
				}
				break
			} else {
				Errorf("DeleteIndexVersion: %v", err)
			}
		}

		r += int64(VERSION_LEN)
		v := string(buffer)
		if !MatchVersion(pattern, ParseVersion(v[:len(v) -1])) {
			if r != w {
				if _, err := f.WriteAt(buffer, w); err != nil {
					Errorf("DeleteIndexVersion: %v", err)
				}
			}
			w += int64(VERSION_LEN)
		}
	}
}

func GetIndexVersionCount() int {
	fi, err := os.Stat("index")
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		} else {
			Errorf("GetIndexVersionCount: %v", err)
		}
	}
	return int(fi.Size() / int64(VERSION_LEN))
}

func GetIndexVersions() []string {
	data, err := ioutil.ReadFile("index")
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}
		}
		Errorf("GetIndexVersions: %v", err)
	}
	if len(data) == 0 {
		return []string{}
	}
	return strings.Split(string(data[:len(data) - 1]), "\n")
}

func GetIndexVersionAt(i int) string {
	f, err := os.Open("index")
	if err != nil {
		Errorf("GetIndexVersionAt: %v", err)
	}

	buf := make([]byte, VERSION_LEN-1)
	if _, err = f.ReadAt(buf, int64(i)*int64(VERSION_LEN)); err != nil {
		Errorf("GetIndexVersionAt: %v", err)
	}
	return string(buf)
}

func GetLatestVersionSha1() string {
	n := GetIndexVersionCount()
	if n == 0 {
		return ""
	}
	return ParseVersion(GetIndexVersionAt(n - 1)).Sha1
}

func MatchVersion(pattern string, version Version) bool {
	return strings.HasPrefix(version.Sha1, pattern) || strings.HasPrefix(version.Timestamp, pattern)
}

func FindIndexVersions(pattern string) (r []string) {
	f, err := os.Open("index")
	if err != nil {
		if os.IsNotExist(err) {
			return r
		} else {
			Errorf("FindIndexVersion: %v", err)
		}
	}
	defer f.Close()

	buffer := make([]byte, VERSION_LEN)
	for {
		_, err := io.ReadFull(f, buffer)
		if err != nil {
			if err == io.EOF {
				return r
			} else {
				Errorf("FindIndexVersion: %v", err)
			}
		}

		v := string(buffer[:VERSION_LEN- 1])
		if MatchVersion(pattern, ParseVersion(v)) {
			r = append(r, v)
		}
	}
}

func ResolveVersions(pattern string) []string {
	if strings.HasPrefix(pattern, "v") {
		return []string{GetIndexVersionAt(ParseIndexedVersion(pattern))}
	} else {
		return FindIndexVersions(pattern)
	}
}

func ResolveVersionSha1(pattern string) string {
	versions := ResolveVersions(pattern)
	if len(versions) == 0 {
		Errorf("未找到对应的版本：%s", pattern)
	}
	if len(versions) > 1 {
		Errorf("找到多个版本，请输入更精确的版本号：%s", pattern)
	}
	return ParseVersion(versions[0]).Sha1
}