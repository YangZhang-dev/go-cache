package lru

import "container/list"

type Cache struct {
	// 最大内存，k+v，0表示无内存限制
	maxBytes int64
	// 当前使用的内存，k+v
	nbytes int64
	// 双向链表
	ll *list.List
	// 缓存项
	cache map[string]*list.Element
	// key被回收时触发的回调函数
	OnEvicted func(key string, value Value)
}

// Value 缓存值需要实现计算内存大小
type Value interface {
	Len() int
}

// 定义了list的存储数据类型 list.Element.Value.(entry)
type entry struct {
	key   string
	value Value
}

// NewCache 创建一个缓存，传入最大支持内存量以及被删除时的回调函数
func NewCache(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 获取缓存项
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 将最近访问的缓存项移到链表最前端
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 删除最远使用的缓存项
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中删除
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从map中删除
		delete(c.cache, kv.key)
		// 更新当前使用的内存量
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 增加缓存项
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 根据内存差值更新
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 维持最大内存限制
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len 获取缓存项条数
func (c *Cache) Len() int {
	return c.ll.Len()
}
