package mvb

import (
	"os"
	"log"
	"time"
	"io"
	"strings"
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
	if _, err = f.ReadAt(buf, int64(i) * int64(VERSION_LINE_LEN)); err != nil {
		log.Fatalf("GetIndexVersionAt: %v", err)
	}
	return string(buf)
}

func FindIndexVersions(version string) (r []string)  {
	f, err := os.Open("index")
	if err != nil {
		if os.IsNotExist(err) {
			return r
		} else {
			log.Fatalf("FindIndexVersion: %v", err)
		}
	}
	defer f.Close()

	buf := make([]byte, VERSION_LINE_LEN)
	for {
		_, err := io.ReadFull(f, buf)
		if err != nil {
			if err == io.EOF {
				return r
			} else {
				log.Fatalf("FindIndexVersion: %v", err)
			}
		}

		line := string(buf)
		v := line[:40]
		t := line[41:]
		if strings.HasPrefix(v, version) || strings.HasPrefix(t, version) {
			r = append(r, line)
		}
	}
}

func ResolveVersion(version string) string {
	vs := FindIndexVersions(version)
	if len(vs) == 0 {
		log.Fatal("ResolveVersion: version not found")
	}
	if len(vs) > 1 {
		log.Fatal("ResolveVersion: too many versions")
	}
	return vs[1]
}

func AddVersionToIndex(id string, t time.Time) string {
	f, err := os.OpenFile("index", os.O_CREATE | os.O_APPEND | os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("AddVersionToIndex: %v", err)
	}
	defer f.Close()

	line := id + " " + t.Format(ISO8601) + "\n"
	f.WriteString(line)

	return id
}