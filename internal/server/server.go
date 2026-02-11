package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/catatsuy/utsuro/internal/cache"
)

type Config struct {
	ListenAddr            string
	MaxBytes              int64
	TargetBytes           int64
	MaxEvictPerOp         int
	IncrSlidingTTLSeconds int64
	Verbose               bool
	Logger                *slog.Logger
}

type Server struct {
	cfg   Config
	cache *cache.Cache

	mu        sync.RWMutex
	listener  net.Listener
	readyCh   chan struct{}
	readyOnce sync.Once
	closed    bool

	logger *slog.Logger
}

func NewServer(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Server{
		cfg:     cfg,
		cache:   cache.NewCache(cfg.MaxBytes, cfg.TargetBytes, 200, cfg.MaxEvictPerOp, cfg.IncrSlidingTTLSeconds),
		readyCh: make(chan struct{}),
		logger:  logger,
	}
}

func (s *Server) Ready() <-chan struct{} {
	return s.readyCh
}

func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	s.readyOnce.Do(func() { close(s.readyCh) })

	s.logf("listening on %s", ln.Addr().String())

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Temporary() {
				s.logf("temporary accept error: %v", err)
				continue
			}
			s.logf("accept error: %v", err)
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

func (s *Server) logf(format string, args ...any) {
	if !s.cfg.Verbose {
		return
	}
	s.logger.Info(fmt.Sprintf(format, args...))
}
