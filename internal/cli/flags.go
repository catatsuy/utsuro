package cli

import "flag"

type options struct {
	listenAddr            string
	maxBytes              int64
	targetBytes           int64
	maxEvictPerOp         int
	incrSlidingTTLSeconds int64
	verbose               bool
	showVersion           bool
}

func parseFlags(args []string) (options, error) {
	opt := options{}
	fs := flag.NewFlagSet("utsuro", flag.ContinueOnError)
	fs.StringVar(&opt.listenAddr, "listen", "127.0.0.1:11211", "TCP address to listen on")
	fs.Int64Var(&opt.maxBytes, "max-bytes", 256*1024*1024, "max logical bytes")
	fs.Int64Var(&opt.targetBytes, "target-bytes", 0, "eviction target bytes")
	fs.IntVar(&opt.maxEvictPerOp, "evict-max", 64, "max evictions per operation")
	fs.Int64Var(&opt.incrSlidingTTLSeconds, "incr-sliding-ttl-seconds", 0, "sliding TTL in seconds for successful incr/decr; 0 disables")
	fs.BoolVar(&opt.verbose, "verbose", false, "verbose logging")
	fs.BoolVar(&opt.showVersion, "version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	if opt.targetBytes <= 0 {
		opt.targetBytes = opt.maxBytes * 95 / 100
	}

	return opt, nil
}
