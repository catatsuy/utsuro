package cache

type lruEntry struct {
	key  string
	item *Item
}
