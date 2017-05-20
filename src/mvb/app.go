package mvb

import (
	"fmt"
	"os"
)

var Verbose bool

func Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func Verbosef(format string, a ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}
