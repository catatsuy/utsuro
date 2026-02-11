package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/catatsuy/utsuro/internal/server"
)

type CLI struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	deps   any
	isTTY  bool
}

func NewCLI(stdout, stderr io.Writer, stdin io.Reader, deps any, isTTY bool) *CLI {
	return &CLI{
		stdout: stdout,
		stderr: stderr,
		stdin:  stdin,
		deps:   deps,
		isTTY:  isTTY,
	}
}

func (c *CLI) Run(args []string) int {
	opts, err := parseFlags(args[1:])
	if err != nil {
		fmt.Fprintf(c.stderr, "failed to parse flags: %v\n", err)
		return 2
	}

	logger := log.New(c.stderr, "utsuro: ", log.LstdFlags)
	srv := server.NewServer(server.Config{
		ListenAddr:    opts.listenAddr,
		MaxBytes:      opts.maxBytes,
		TargetBytes:   opts.targetBytes,
		MaxEvictPerOp: opts.maxEvictPerOp,
		Verbose:       opts.verbose,
		Logger:        logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil {
		fmt.Fprintf(c.stderr, "server failed: %v\n", err)
		return 1
	}
	return 0
}
