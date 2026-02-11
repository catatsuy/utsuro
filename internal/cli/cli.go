package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/catatsuy/utsuro/internal/server"
)

type CLI struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func NewCLI(stdout, stderr io.Writer, stdin io.Reader) *CLI {
	return &CLI{
		stdout: stdout,
		stderr: stderr,
		stdin:  stdin,
	}
}

func (c *CLI) Run(args []string) int {
	opts, err := parseFlags(args[1:])
	if err != nil {
		fmt.Fprintf(c.stderr, "failed to parse flags: %v\n", err)
		return 2
	}

	logger := slog.New(slog.NewTextHandler(c.stderr, nil))
	srv := server.NewServer(server.Config{
		ListenAddr:            opts.listenAddr,
		MaxBytes:              opts.maxBytes,
		TargetBytes:           opts.targetBytes,
		MaxEvictPerOp:         opts.maxEvictPerOp,
		IncrSlidingTTLSeconds: opts.incrSlidingTTLSeconds,
		Verbose:               opts.verbose,
		Logger:                logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil {
		fmt.Fprintf(c.stderr, "server failed: %v\n", err)
		return 1
	}
	return 0
}
