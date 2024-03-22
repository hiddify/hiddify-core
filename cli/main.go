package main

import (
	"os"

	"github.com/hiddify/hiddify-core/cmd"
)

func main() {
	cmd.ParseCli(os.Args[1:])
}
