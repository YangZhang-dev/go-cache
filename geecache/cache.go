package geecache

import (
	"geecache/lru"
	"sync"
)

type cache struct {
	sync.RWMutex
	lru        *lru.Cache
	cacheBytes int64
}

// 新增缓存项，传入string和ByteView
func (c *cache) add(key string, value ByteView) {
	c.Lock()
	defer c.Unlock()
	if c.lru == nil {
		// 延迟初始化
		c.lru = lru.NewCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// 获取缓存项，传入key，返回ByteView和是否存在
func (c *cache) get(key string) (ByteView, bool) {
	c.RLock()
	defer c.RUnlock()
	if c.lru == nil {
		return ByteView{}, false
	}
	if value, ok := c.lru.Get(key); ok {
		return value.(ByteView), true
	}
	return ByteView{}, false
}
