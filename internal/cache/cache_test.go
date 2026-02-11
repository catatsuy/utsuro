package cache

import "testing"

func TestIncrMissingCreatesKey(t *testing.T) {
	c := NewCache(1024, 1024, 0, 64, 0)
	v, err := c.Incr("k", 7)
	if err != nil {
		t.Fatalf("incr failed: %v", err)
	}
	if v != 7 {
		t.Fatalf("unexpected value: %d", v)
	}

	item, ok := c.Get("k")
	if !ok {
		t.Fatal("missing key after incr")
	}
	if string(item.Value) != "7" {
		t.Fatalf("unexpected stored value: %q", string(item.Value))
	}
}

func TestDecrMissingCreatesZero(t *testing.T) {
	c := NewCache(1024, 1024, 0, 64, 0)
	v, err := c.Decr("k", 7)
	if err != nil {
		t.Fatalf("decr failed: %v", err)
	}
	if v != 0 {
		t.Fatalf("unexpected value: %d", v)
	}

	item, ok := c.Get("k")
	if !ok {
		t.Fatal("missing key after decr")
	}
	if string(item.Value) != "0" {
		t.Fatalf("unexpected stored value: %q", string(item.Value))
	}
}

func TestIncrOverflowReturnsError(t *testing.T) {
	c := NewCache(1024, 1024, 0, 64, 0)
	if err := c.Set("k", 0, []byte("18446744073709551615")); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	_, err := c.Incr("k", 1)
	if err != ErrOverflow {
		t.Fatalf("expected ErrOverflow, got: %v", err)
	}
}

func TestSlidingTTLAndExpiredRecreate(t *testing.T) {
	c := NewCache(1024, 1024, 0, 64, 10)
	now := int64(100)
	restore := SetNowUnixForTest(func() int64 { return now })
	defer restore()

	if _, err := c.Incr("k", 5); err != nil {
		t.Fatalf("incr failed: %v", err)
	}
	item, ok := c.Get("k")
	if !ok {
		t.Fatal("missing key after incr")
	}
	if item.ExpUnix != 110 {
		t.Fatalf("unexpected exp after incr: %d", item.ExpUnix)
	}

	now = 120
	v, err := c.Incr("k", 2)
	if err != nil {
		t.Fatalf("incr on expired key failed: %v", err)
	}
	if v != 2 {
		t.Fatalf("expected recreate value 2, got: %d", v)
	}
	item, ok = c.Get("k")
	if !ok {
		t.Fatal("missing key after recreate")
	}
	if item.ExpUnix != 130 {
		t.Fatalf("unexpected exp after recreate: %d", item.ExpUnix)
	}
}

func TestEvictionPrefersExpired(t *testing.T) {
	c := NewCache(12, 12, 0, 64, 10)
	now := int64(100)
	restore := SetNowUnixForTest(func() int64 { return now })
	defer restore()

	if err := c.Set("live", 0, []byte("1111")); err != nil {
		t.Fatalf("set live failed: %v", err)
	}
	if _, err := c.Incr("exp", 1); err != nil {
		t.Fatalf("incr exp failed: %v", err)
	}

	now = 111 // "exp" is expired, "live" is not expired (no TTL)
	if err := c.Set("n", 0, []byte("123")); err != nil {
		t.Fatalf("set n failed: %v", err)
	}

	if _, ok := c.Get("live"); !ok {
		t.Fatal("live should remain when expired item exists")
	}
	if _, ok := c.Get("exp"); ok {
		t.Fatal("expired key should be evicted first")
	}
	if _, ok := c.Get("n"); !ok {
		t.Fatal("new key should be stored")
	}
}
