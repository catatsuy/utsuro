package cache

import "github.com/catatsuy/utsuro/internal/model"

type lruEntry struct {
	key  string
	item *model.Item
}
