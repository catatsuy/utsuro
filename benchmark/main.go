package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/catatsuy/utsuro/internal/server"
)

const defaultAddr = "127.0.0.1:11211"

func main() {
	if err := runDemo(defaultAddr); err != nil {
		panic(err)
	}
}

func runDemo(addr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := server.NewServer(server.Config{
		ListenAddr: addr,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	select {
	case <-srv.Ready():
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server failed before ready: %w", err)
		}
		return fmt.Errorf("server exited before ready")
	case <-time.After(3 * time.Second):
		return fmt.Errorf("server did not become ready")
	}

	addr = srv.Addr()
	if addr == "" {
		return fmt.Errorf("server address is empty")
	}

	mc := memcache.New(addr)

	if err := mc.Set(&memcache.Item{Key: "hello", Value: []byte("world")}); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}
	item, err := mc.Get("hello")
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}
	fmt.Printf("get hello => %s\n", string(item.Value))

	v, err := mc.Increment("counter", 1)
	if err != nil {
		return fmt.Errorf("incr missing failed: %w", err)
	}
	fmt.Printf("incr counter 1 (missing key) => %d\n", v)

	v, err = mc.Increment("counter", 2)
	if err != nil {
		return fmt.Errorf("incr existing failed: %w", err)
	}
	fmt.Printf("incr counter 2 => %d\n", v)

	v, err = mc.Decrement("counter", 10)
	if err != nil {
		return fmt.Errorf("decr failed: %w", err)
	}
	fmt.Printf("decr counter 10 => %d\n", v)

	fmt.Println("gomemcache client works with utsuro text protocol subset")

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server stop error: %w", err)
		}
	case <-time.After(3 * time.Second):
		return fmt.Errorf("server shutdown timeout")
	}

	return nil
}
