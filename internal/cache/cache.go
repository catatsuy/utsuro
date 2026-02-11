package cache

import (
	"container/list"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/catatsuy/utsuro/internal/model"
)

var (
	ErrObjectTooLarge = errors.New("object too large")
	ErrNoSpace        = errors.New("out of memory")
	ErrNonNumeric     = errors.New("cannot increment or decrement non-numeric value")
	ErrOverflow       = errors.New("increment or decrement overflow")
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

	incrSlidingTTLSeconds int64
	nextCAS               uint64
}

var nowUnix = func() int64 { return time.Now().Unix() }

func NewCache(maxBytes, targetBytes, entryOverhead int64, maxEvictPerOp int, incrSlidingTTLSeconds int64) *Cache {
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
	if incrSlidingTTLSeconds < 0 {
		incrSlidingTTLSeconds = 0
	}

	return &Cache{
		maxBytes:              maxBytes,
		targetBytes:           targetBytes,
		items:                 make(map[string]*list.Element),
		lru:                   list.New(),
		entryOverhead:         entryOverhead,
		maxEvictPerOp:         maxEvictPerOp,
		incrSlidingTTLSeconds: incrSlidingTTLSeconds,
		nextCAS:               1,
	}
}

func (c *Cache) Get(key string) (*model.Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	now := nowUnix()
	entry := elem.Value.(*lruEntry)
	if isExpired(entry.item, now) {
		c.removeElementLocked(elem)
		return nil, false
	}
	c.lru.MoveToFront(elem)

	return cloneItem(entry.item), true
}

func (c *Cache) Set(key string, flags uint32, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.setLocked(key, flags, value, 0)
}

func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return false
	}
	now := nowUnix()
	entry := elem.Value.(*lruEntry)
	if isExpired(entry.item, now) {
		c.removeElementLocked(elem)
		return false
	}
	c.removeElementLocked(elem)
	return true
}

func (c *Cache) Incr(key string, delta uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := nowUnix()
	expUnix := c.expirationForIncrDecr(now)

	elem, ok := c.items[key]
	if !ok {
		if err := c.setLocked(key, 0, []byte(strconv.FormatUint(delta, 10)), expUnix); err != nil {
			return 0, err
		}
		return delta, nil
	}

	entry := elem.Value.(*lruEntry)
	if isExpired(entry.item, now) {
		c.removeElementLocked(elem)
		if err := c.setLocked(key, 0, []byte(strconv.FormatUint(delta, 10)), expUnix); err != nil {
			return 0, err
		}
		return delta, nil
	}

	cur, err := parseUint(entry.item.Value)
	if err != nil {
		return 0, ErrNonNumeric
	}
	if cur > math.MaxUint64-delta {
		return 0, ErrOverflow
	}
	next := cur + delta
	if err := c.setLocked(key, entry.item.Flags, []byte(strconv.FormatUint(next, 10)), expUnix); err != nil {
		return 0, err
	}
	return next, nil
}

func (c *Cache) Decr(key string, delta uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := nowUnix()
	expUnix := c.expirationForIncrDecr(now)

	elem, ok := c.items[key]
	if !ok {
		if err := c.setLocked(key, 0, []byte("0"), expUnix); err != nil {
			return 0, err
		}
		return 0, nil
	}

	entry := elem.Value.(*lruEntry)
	if isExpired(entry.item, now) {
		c.removeElementLocked(elem)
		if err := c.setLocked(key, 0, []byte("0"), expUnix); err != nil {
			return 0, err
		}
		return 0, nil
	}

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
	if err := c.setLocked(key, entry.item.Flags, []byte(strconv.FormatUint(next, 10)), expUnix); err != nil {
		return 0, err
	}
	return next, nil
}

func (c *Cache) setLocked(key string, flags uint32, value []byte, expUnix int64) error {
	need := c.entrySize(key, value)
	if need > c.maxBytes {
		return ErrObjectTooLarge
	}
	now := nowUnix()

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*lruEntry)
		if isExpired(entry.item, now) {
			c.removeElementLocked(elem)
		} else {
			delta := need - entry.item.Size
			if delta > 0 {
				c.evictLocked(delta, key, now)
			}
			if c.usedBytes+delta > c.maxBytes {
				return ErrNoSpace
			}

			entry.item.Value = cloneBytes(value)
			entry.item.Flags = flags
			entry.item.Size = need
			entry.item.CAS = c.nextCASLocked()
			entry.item.ExpUnix = expUnix
			c.usedBytes += delta
			c.lru.MoveToFront(elem)
			c.evictBestEffortLocked("", now)
			return nil
		}
	}

	c.evictLocked(need, "", now)
	if c.usedBytes+need > c.maxBytes {
		return ErrNoSpace
	}

	item := &model.Item{
		Key:     key,
		Value:   cloneBytes(value),
		Flags:   flags,
		Size:    need,
		CAS:     c.nextCASLocked(),
		ExpUnix: expUnix,
	}
	elem := c.lru.PushFront(&lruEntry{key: key, item: item})
	c.items[key] = elem
	c.usedBytes += need
	c.evictBestEffortLocked("", now)
	return nil
}

func (c *Cache) evictLocked(incomingDelta int64, protectKey string, now int64) {
	evicted := 0
	for c.usedBytes+incomingDelta > c.maxBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey, now)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}

	for c.usedBytes+incomingDelta > c.targetBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey, now)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}
}

func (c *Cache) evictBestEffortLocked(protectKey string, now int64) {
	evicted := 0
	for c.usedBytes > c.targetBytes && evicted < c.maxEvictPerOp {
		victim := c.selectVictimLocked(protectKey, now)
		if victim == nil {
			return
		}
		c.removeElementLocked(victim)
		evicted++
	}
}

func (c *Cache) selectVictimLocked(protectKey string, now int64) *list.Element {
	var fallback *list.Element
	for elem := c.lru.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*lruEntry)
		if entry.key == protectKey {
			continue
		}
		if isExpired(entry.item, now) {
			return elem
		}
		if fallback == nil {
			fallback = elem
		}
	}
	return fallback
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
		Key:     item.Key,
		Value:   cloneBytes(item.Value),
		Flags:   item.Flags,
		Size:    item.Size,
		CAS:     item.CAS,
		ExpUnix: item.ExpUnix,
	}
}

func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func (c *Cache) expirationForIncrDecr(now int64) int64 {
	if c.incrSlidingTTLSeconds <= 0 {
		return 0
	}
	return now + c.incrSlidingTTLSeconds
}

func isExpired(item *model.Item, now int64) bool {
	return item.ExpUnix > 0 && item.ExpUnix <= now
}

func (c *Cache) nextCASLocked() uint64 {
	v := c.nextCAS
	c.nextCAS++
	if c.nextCAS == 0 {
		c.nextCAS = 1
	}
	return v
}
