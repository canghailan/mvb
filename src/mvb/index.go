package mvb

import (
	"os"
	"log"
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
	if _, err = f.ReadAt(buf, int64(i) * int64(VERSION_LINE_LEN)); err != nil {
		log.Fatalf("GetIndexVersionAt: %v", err)
	}
	return string(buf)
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