package main

import (
	"./common"
	"fmt"
	"flag"
	"path/filepath"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	flag.Parse()
	base, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	command := flag.Arg(0)
	switch command {
	case "init": {
		mvb.Init(base)
		break
	}
	case "backup": {
		root:= flag.Arg(1)
		mvb.CreateSnapshot(base, root)
		break
	}
	}
}
