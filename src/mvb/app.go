package mvb

import (
	"fmt"
	"os"
)

var Verbose bool

func Print(a ...interface{})  {
	fmt.Fprint(os.Stdout, a...)
}

func Println(a ...interface{})  {
	fmt.Fprintln(os.Stdout, a...)
}

func Printf(format string, a ...interface{})  {
	fmt.Fprintf(os.Stdout, format, a...)
}

func Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func Verbosef(format string, a ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}