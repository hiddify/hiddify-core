package main

import (
	"os"

	"github.com/hiddify/libcore/cmd"
)

func main() {
	cmd.ParseCli(os.Args[1:])
}
