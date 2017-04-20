package main

import (
	"./common"
	"fmt"
	"flag"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	flag.Parse()

	command := flag.Arg(0)
	switch command {
	case "init":
		{
			root:= flag.Arg(1)
			mvb.Init(root)
			break
		}
	case "backup":
		{
			mvb.Backup()
			break
		}
	}
}
