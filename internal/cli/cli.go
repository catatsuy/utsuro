package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/catatsuy/utsuro/internal/server"
)

var Version string

func version() string {
	if Version != "" {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}

	return info.Main.Version
}

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
	if opts.showVersion {
		fmt.Fprintf(c.stdout, "utsuro version %s; %s\n", version(), runtime.Version())
		return 0
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
