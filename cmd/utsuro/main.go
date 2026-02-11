package main

import (
	"os"

	"github.com/catatsuy/utsuro/internal/cli"
)

func main() {
	cl := cli.NewCLI(os.Stdout, os.Stderr, os.Stdin)
	os.Exit(cl.Run(os.Args))
}
