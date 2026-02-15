package cache

import "testing"

func FuzzCacheOperations(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	f.Add([]byte("set/get/delete/incr/decr"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewCache(256, 256, 0, 64, 0)
		keys := []string{"a", "b", "c", "d", "e"}

		for i := 0; i < len(data); i++ {
			op := data[i] % 5
			key := keys[int(data[i])%len(keys)]

			switch op {
			case 0:
				valLen := int(data[i] % 32)
				value := make([]byte, valLen)
				for j := 0; j < valLen; j++ {
					value[j] = data[(i+j)%len(data)]
				}
				_ = c.Set(key, uint32(data[i]), value)
			case 1:
				item, ok := c.Get(key)
				if ok {
					wantSize := c.entrySize(key, item.Value)
					if item.Size != wantSize {
						t.Fatalf("size mismatch for key %q: got=%d want=%d", key, item.Size, wantSize)
					}
				}
			case 2:
				_ = c.Delete(key)
			case 3:
				_, _ = c.Incr(key, uint64(data[i]))
			case 4:
				_, _ = c.Decr(key, uint64(data[i]))
			}

			assertCacheInvariant(t, c)
		}
	})
}

func assertCacheInvariant(t *testing.T, c *Cache) {
	t.Helper()

	if c.usedBytes < 0 {
		t.Fatalf("usedBytes must be non-negative: %d", c.usedBytes)
	}
	if c.usedBytes > c.maxBytes {
		t.Fatalf("usedBytes must not exceed maxBytes: used=%d max=%d", c.usedBytes, c.maxBytes)
	}

	var total int64
	for key, elem := range c.items {
		if elem == nil || elem.Value == nil || elem.Value.item == nil {
			t.Fatalf("invalid entry state for key %q", key)
		}
		size := c.entrySize(key, elem.Value.item.Value)
		if elem.Value.item.Size != size {
			t.Fatalf("item size mismatch for key %q: got=%d want=%d", key, elem.Value.item.Size, size)
		}
		total += elem.Value.item.Size
	}
	if total != c.usedBytes {
		t.Fatalf("usedBytes mismatch: got=%d want=%d", c.usedBytes, total)
	}
}
