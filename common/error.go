package mvb

import "io"

func NoExcept(err error) bool {
	return true
}

func IsEOF(err error) bool {
	return err == io.EOF
}

func CheckError(err error, ignore func(error) bool) bool {
	if err == nil {
		return false
	}
	if ignore(err) {
		return true
	} else {
		panic(err)
	}
}
