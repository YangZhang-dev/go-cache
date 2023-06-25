package cache2go

import (
	"container/list"
	"sync"
	"time"
)

type CacheItem struct {
	list.List
	// 互斥读写锁保证并发安全
	sync.RWMutex
	// k,v 可以是任意类型
	key  interface{}
	data interface{}
	// 存活时间
	lifeSpan time.Duration
	// 创建时间
	createTime time.Time
	// 最后访问时间
	accessedTime time.Time
	// 访问次数
	accessCount int64
	// 在item将要被删除时触发的回调函数切片
	aboutToExpire []func(key interface{})
}

// NewCacheItem 创建一个CacheItem
func NewCacheItem(key, value interface{}, lifeSpan time.Duration) *CacheItem {
	now := time.Now()
	return &CacheItem{
		key:           key,
		data:          value,
		lifeSpan:      lifeSpan,
		createTime:    now,
		accessedTime:  now,
		accessCount:   0,
		aboutToExpire: nil,
	}
}

// KeepAlive 当访问该缓存项时需要调用
func (ci *CacheItem) KeepAlive() {
	ci.Lock()
	defer ci.Unlock()
	ci.accessCount++
	ci.accessedTime = time.Now()
}

// LifeSpan 获取缓存项的存活时间
func (ci *CacheItem) LifeSpan() time.Duration {
	return ci.lifeSpan
}

// AccessedTime 获取最近的访问时间
func (ci *CacheItem) AccessedTime() time.Time {
	ci.RLock()
	defer ci.RUnlock()
	return ci.accessedTime
}

// AccessedCount 获取访问次数
func (ci *CacheItem) AccessedCount() int64 {
	ci.RLock()
	defer ci.RUnlock()
	return ci.accessCount
}

// CreateTime 获取创建时间
func (ci *CacheItem) CreateTime() time.Time {
	return ci.createTime
}

// Key 获取键
func (ci *CacheItem) Key() interface{} {
	return ci.key
}

// Data 获取数据
func (ci *CacheItem) Data() interface{} {
	return ci.data
}

// RemoveAboutToExpireCallBack 将删除时触发的回调函数清空
func (ci *CacheItem) RemoveAboutToExpireCallBack() {
	ci.Lock()
	defer ci.Unlock()
	ci.aboutToExpire = nil
}

// SetAboutToExpireCallback 设置删除时触发的回调函数，如果切片不为空，那么就先清空再设置
func (ci *CacheItem) SetAboutToExpireCallback(f func(interface{})) {
	if len(ci.aboutToExpire) > 0 {
		ci.RemoveAboutToExpireCallBack()
	}
	ci.Lock()
	defer ci.Unlock()
	ci.aboutToExpire = append(ci.aboutToExpire, f)
}

// AddAboutToExpireCallback 向切片中增加删除时触发的回调函数
func (ci *CacheItem) AddAboutToExpireCallback(f func(interface{})) {
	ci.Lock()
	defer ci.Unlock()
	ci.aboutToExpire = append(ci.aboutToExpire, f)
}
