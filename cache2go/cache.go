package cache2go

import "sync"

var (
	cache = make(map[string]*CacheTable)
	mutex sync.RWMutex
)

// Cache 创建新的缓存表，如果存在就返回已存在的缓存表
func Cache(table string) *CacheTable {
	mutex.Lock()
	defer mutex.Unlock()
	t, ok := cache[table]

	if !ok {
		t = &CacheTable{
			name:  table,
			items: make(map[interface{}]*CacheItem),
		}
		cache[table] = t
	}
	return t
}
