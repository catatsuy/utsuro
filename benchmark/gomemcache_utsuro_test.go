package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/catatsuy/utsuro/internal/server"
)

func startUtsuroServer(t *testing.T) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	srv := server.NewServer(server.Config{
		ListenAddr: "127.0.0.1:0",
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	select {
	case <-srv.Ready():
	case err := <-errCh:
		t.Fatalf("server failed before ready: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not become ready")
	}

	addr := srv.Addr()
	if addr == "" {
		t.Fatalf("server address is empty")
	}

	stop := func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(3 * time.Second):
			t.Fatalf("server shutdown timeout")
		}
	}
	return addr, stop
}

func mustSet(t *testing.T, c *memcache.Client, it *memcache.Item) {
	t.Helper()
	if err := c.Set(it); err != nil {
		t.Fatalf("failed to Set %#v: %v", *it, err)
	}
}

func TestGomemcacheWithUtsuro(t *testing.T) {
	addr, stop := startUtsuroServer(t)
	defer stop()

	c := memcache.New(addr)

	foo := &memcache.Item{Key: "foo", Value: []byte("fooval-fromset"), Flags: 123}
	if err := c.Set(foo); err != nil {
		t.Fatalf("set(foo): %v", err)
	}

	it, err := c.Get("foo")
	if err != nil {
		t.Fatalf("get(foo): %v", err)
	}
	if string(it.Value) != "fooval-fromset" {
		t.Fatalf("get(foo) value: want=%q got=%q", "fooval-fromset", string(it.Value))
	}
	if it.Flags != 123 {
		t.Fatalf("get(foo) flags: want=%d got=%d", 123, it.Flags)
	}

	qux := &memcache.Item{Key: "Hello_世界", Value: []byte("hello world")}
	if err := c.Set(qux); err != nil {
		t.Fatalf("set(Hello_世界): %v", err)
	}
	it, err = c.Get(qux.Key)
	if err != nil {
		t.Fatalf("get(Hello_世界): %v", err)
	}
	if string(it.Value) != "hello world" {
		t.Fatalf("get(Hello_世界) value: want=%q got=%q", "hello world", string(it.Value))
	}

	if err := c.Set(&memcache.Item{Key: "foo bar", Value: []byte("x")}); err != memcache.ErrMalformedKey {
		t.Fatalf("set malformed key: want=%v got=%v", memcache.ErrMalformedKey, err)
	}

	mustSet(t, c, &memcache.Item{Key: "bar", Value: []byte("barval")})
	m, err := c.GetMulti([]string{"foo", "bar"})
	if err != nil {
		t.Fatalf("GetMulti: %v", err)
	}
	if _, ok := m["foo"]; !ok {
		t.Fatalf("GetMulti missing key foo")
	}
	if _, ok := m["bar"]; !ok {
		t.Fatalf("GetMulti missing key bar")
	}

	if err := c.Delete("foo"); err != nil {
		t.Fatalf("Delete(foo): %v", err)
	}
	if _, err := c.Get("foo"); err != memcache.ErrCacheMiss {
		t.Fatalf("get after delete: want=%v got=%v", memcache.ErrCacheMiss, err)
	}

	mustSet(t, c, &memcache.Item{Key: "num", Value: []byte("42")})
	n, err := c.Increment("num", 8)
	if err != nil || n != 50 {
		t.Fatalf("increment: want=(50,nil) got=(%d,%v)", n, err)
	}
	n, err = c.Decrement("num", 49)
	if err != nil || n != 1 {
		t.Fatalf("decrement: want=(1,nil) got=(%d,%v)", n, err)
	}

	if err := c.Delete("num"); err != nil {
		t.Fatalf("Delete(num): %v", err)
	}
	n, err = c.Increment("num", 1)
	if err != nil || n != 1 {
		t.Fatalf("increment missing key: want=(1,nil) got=(%d,%v)", n, err)
	}
	n, err = c.Decrement("missing", 9)
	if err != nil || n != 0 {
		t.Fatalf("decrement missing key: want=(0,nil) got=(%d,%v)", n, err)
	}

	mustSet(t, c, &memcache.Item{Key: "nonnum", Value: []byte("abc")})
	if _, err := c.Increment("nonnum", 1); err == nil || !strings.Contains(err.Error(), "client error") {
		t.Fatalf("increment non numeric: got=%v", err)
	}

	mustSet(t, c, &memcache.Item{Key: "max", Value: []byte("18446744073709551615")})
	if _, err := c.Increment("max", 1); err == nil || !strings.Contains(err.Error(), "client error") {
		t.Fatalf("increment overflow: got=%v", err)
	}
}

func TestGomemcacheWithUtsuroUnsupportedCommands(t *testing.T) {
	// The following upstream checks are intentionally commented out for utsuro.
	// utsuro's MVP protocol subset does not implement:
	// - CompareAndSwap (cas)
	// - Add / Replace / Append / Prepend
	// - Touch / GetAndTouch
	// - DeleteAll (flush_all)
	// - Ping (version)
	//
	// Example (upstream):
	//   err := c.CompareAndSwap(item)
	//   if err != memcache.ErrCASConflict { ... }
	//
	//   err := c.DeleteAll()
	//   if err != nil { ... }
	//
	//   err := c.Ping()
	//   if err != nil { ... }
	t.Skip("unsupported memcached commands are commented out for utsuro")
}

