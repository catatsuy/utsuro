package cache

import (
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/catatsuy/utsuro/internal/model"
)

var (
	ErrObjectTooLarge = errors.New("object too large")
	ErrNoSpace        = errors.New("no space available")
	ErrNonNumeric     = errors.New("cannot increment or decrement non-numeric value")
)

type Cache struct {
	mu sync.Mutex

	maxBytes    int64
	targetBytes int64
	usedBytes   int64

	items map[string]*list.Element
	lru   *list.List

	entryOverhead int64
	maxEvictPerOp int
}

func NewCache(maxBytes, targetBytes, entryOverhead int64, maxEvictPerOp int) *Cache {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024 * 1024
	}
	if targetBytes <= 0 || targetBytes > maxBytes {
		targetBytes = maxBytes * 95 / 100
	}
	if entryOverhead < 0 {
		entryOverhead = 0
	}
	if maxEvictPerOp <= 0 {
		maxEvictPerOp = 64
	}

	return &Cache{
		maxBytes:      maxBytes,
		targetBytes:   targetBytes,
		items:         make(map[string]*list.Element),
		lru:           list.New(),
		entryOverhead: entryOverhead,
		maxEvictPerOp: maxEvictPerOp,
	}
}

func (c *Cache) Get(key string) (*model.Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.lru.MoveToFront(elem)
	entry := elem.Value.(*lruEntry)

	return cloneItem(entry.item), true
}

func (c *Cache) Set(key string, flags uint32, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.setLocked(key, flags, value)
}

func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return false
	}
	c.removeElementLocked(elem)
	return true
}

func (c *Cache) Incr(key string, delta uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		if err := c.setLocked(key, 0, []byte(strconv.FormatUint(delta, 10))); err != nil {
			return 0, err
		}
		return delta, nil
	}

	entry := elem.Value.(*lruEntry)
	cur, err := parseUint(entry.item.Value)
	if err != nil {
		return 0, ErrNonNumeric
	}
	next := cur + delta
	if err := c.setLocked(key, entry.item.Flags, []byte(strconv.FormatUint(next, 10))); err != nil {
		return 0, err
	}
	return next, nil
}

func (c *Cache) Decr(key string, delta uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		if err := c.setLocked(key, 0, []byte("0")); err != nil {
			return 0, err
		}
		return 0, nil
	}

	entry := elem.Value.(*lruEntry)
	cur, err := parseUint(entry.item.Value)
	if err != nil {
		return 0, ErrNonNumeric
	}

	var next uint64
	if delta >= cur {
		next = 0
	} else {
		next = cur - delta
	}
	if err := c.setLocked(key, entry.item.Flags, []byte(strconv.FormatUint(next, 10))); err != nil {
		return 0, err
	}
	return next, nil
}

func (c *Cache) setLocked(key string, flags uint32, value []byte) error {
	need := c.entrySize(key, value)
	if need > c.maxBytes {
		return ErrObjectTooLarge
	}

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*lruEntry)
		delta := need - entry.item.Size
		if delta > 0 {
			c.evictLocked(delta, key)
		}
		if c.usedBytes+delta > c.maxBytes {
			return ErrNoSpace
		}

		entry.item.Value = cloneBytes(value)
		entry.item.Flags = flags
		entry.item.Size = need
		c.usedBytes += delta
		c.lru.MoveToFront(elem)
		c.evictBestEffortLocked("")
		return nil
	}

	c.evictLocked(need, "")
	if c.usedBytes+need > c.maxBytes {
		return ErrNoSpace
	}

	item := &model.Item{
		Key:   key,
		Value: cloneBytes(value),
		Flags: flags,
		Size:  need,
	}
	elem := c.lru.PushFront(&lruEntry{key: key, item: item})
	c.items[key] = elem
	c.usedBytes += need
	c.evictBestEffortLocked("")
	return nil
}

func (c *Cache) evictLocked(incomingDelta int64, protectKey string) {
	evicted := 0
	for c.usedBytes+incomingDelta > c.maxBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}

	for c.usedBytes+incomingDelta > c.targetBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}
}

func (c *Cache) evictBestEffortLocked(protectKey string) {
	evicted := 0
	for c.usedBytes > c.targetBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}
}

func (c *Cache) selectVictimLocked(protectKey string) *list.Element {
	for elem := c.lru.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*lruEntry)
		if entry.key == protectKey {
			continue
		}
		return elem
	}
	return nil
}

func (c *Cache) removeElementLocked(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	delete(c.items, entry.key)
	c.lru.Remove(elem)
	c.usedBytes -= entry.item.Size
	if c.usedBytes < 0 {
		c.usedBytes = 0
	}
}

func (c *Cache) entrySize(key string, value []byte) int64 {
	return int64(len(key)+len(value)) + c.entryOverhead
}

func parseUint(value []byte) (uint64, error) {
	if len(value) == 0 {
		return 0, fmt.Errorf("empty value")
	}
	return strconv.ParseUint(string(value), 10, 64)
}

func cloneItem(item *model.Item) *model.Item {
	return &model.Item{
		Key:   item.Key,
		Value: cloneBytes(item.Value),
		Flags: item.Flags,
		Size:  item.Size,
	}
}

func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
