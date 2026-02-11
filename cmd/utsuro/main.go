package main

import (
	"os"

	"github.com/catatsuy/utsuro/internal/cli"
	"github.com/catatsuy/utsuro/internal/term"
)

func main() {
	cl := cli.NewCLI(os.Stdout, os.Stderr, os.Stdin, nil, term.IsTerminal(int(os.Stdin.Fd())))
	os.Exit(cl.Run(os.Args))
}
