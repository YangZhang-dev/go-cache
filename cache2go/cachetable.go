package cache2go

import (
	"log"
	"sort"
	"sync"
	"time"
)

type CacheTable struct {
	sync.RWMutex

	// 缓存表的名字
	name string
	// 使用map存储每一个缓存项
	items map[interface{}]*CacheItem
	// 负责触发清理过期缓存项的定时器
	cleanupTimer *time.Timer
	// 当前定时器的持续时间
	cleanupDuration time.Duration
	// 当尝试获取缓存表中不存在的缓存项时触发的回调函数
	loadData func(key interface{}, args ...interface{}) *CacheItem
	// 当增加一个缓存项时触发的回调函数
	addedItem []func(item *CacheItem)
	// 当删除一个缓存项时触发的回调函数
	deletedItem []func(item *CacheItem)
	// 日志
	logger *log.Logger
}

// SetDataLoader 设置当尝试获取缓存表中不存在的缓存项时触发的回调函数
// TODO 搞清楚args的作用
func (ct *CacheTable) SetDataLoader(f func(interface{}, ...interface{}) *CacheItem) {
	ct.Lock()
	defer ct.Unlock()
	ct.loadData = f
}

// SetLogger 设置内部日志系统
func (ct *CacheTable) SetLogger(logger *log.Logger) {
	ct.Lock()
	defer ct.Unlock()
	ct.logger = logger
}

// RemoveAddedItemCallBack 清空增加缓存项时触发的回调函数
func (ct *CacheTable) RemoveAddedItemCallBack() {
	ct.Lock()
	defer ct.Unlock()
	ct.addedItem = nil
}

// SetAddedItemCallback 设置增加缓存项时触发的回调函数
func (ct *CacheTable) SetAddedItemCallback(f func(*CacheItem)) {
	if len(ct.addedItem) > 0 {
		ct.RemoveAddedItemCallBack()
	}
	ct.Lock()
	defer ct.Unlock()
	ct.addedItem = append(ct.addedItem, f)
}

// AddAddedItemCallback 新增增加缓存项时触发的回调函数
func (ct *CacheTable) AddAddedItemCallback(f func(*CacheItem)) {
	ct.Lock()
	defer ct.Unlock()
	ct.addedItem = append(ct.addedItem, f)
}

// SetDeleteItemCallback 设置删除缓存项时触发的回调函数
func (ct *CacheTable) SetDeleteItemCallback(f func(*CacheItem)) {
	if len(ct.deletedItem) > 0 {
		ct.RemoveDeleteItemCallback()
	}
	ct.Lock()
	defer ct.Unlock()
	ct.deletedItem = append(ct.deletedItem, f)
}

// RemoveDeleteItemCallback 清空删除缓存项时触发的回调函数
func (ct *CacheTable) RemoveDeleteItemCallback() {
	ct.Lock()
	defer ct.Unlock()
	ct.deletedItem = nil
}

// AddDeleteItemCallback 新增删除缓存项时触发的回调函数
func (ct *CacheTable) AddDeleteItemCallback(f func(*CacheItem)) {
	ct.Lock()
	defer ct.Unlock()
	ct.deletedItem = append(ct.deletedItem, f)
}

// Count 返获取缓存项的个数
func (ct *CacheTable) Count() int {
	ct.RLock()
	defer ct.RUnlock()
	return len(ct.items)
}

// Foreach 对所有缓存项进行遍历操作
func (ct *CacheTable) Foreach(op func(interface{}, *CacheItem)) {
	ct.RLock()
	defer ct.RUnlock()

	for k, v := range ct.items {
		op(k, v)
	}
}

// 遍历所有的缓存项进行超时检查，更新定时器的持续时间为所有缓存项中距离超时最近的时间，并且异步调用本身
func (ct *CacheTable) expirationCheck() {
	ct.Lock()
	// 在每一次调用本函数时，需要停止上一次的计时器，以方便本次设置
	if ct.cleanupTimer != nil {
		ct.cleanupTimer.Stop()
	}

	if ct.cleanupDuration > 0 {
		ct.log(ct.name+"缓存表的定时器将于", ct.cleanupDuration, "秒后触发")
	} else {
		ct.log(ct.name + "缓存表的定时器已注册")
	}

	now := time.Now()
	// 当前缓存项中距离过期最短的时间
	smallestDuration := 0 * time.Second
	for k, v := range ct.items {
		// 通过局部变量保存，减少持有锁的时间
		v.RLock()
		lifeSpan := v.LifeSpan()
		accessedTime := v.accessedTime
		v.RUnlock()

		// 对于存活时间为0的缓存项不去管理
		if lifeSpan == 0 {
			continue
		}
		// 距离上次访问经历的时间
		curDuration := lifeSpan - now.Sub(accessedTime)
		if curDuration <= 0 {
			ct.Unlock()
			// 超时的缓存项进行删除操作
			if _, err := ct.deleteInternal(k); err != nil {
				ct.log("缓存表：", ct.name, " 删除缓存项：", k, " 失败")
			}
			ct.Lock()
		} else {
			// 如果是第一次设置或当前缓存项的持续时间小于记录的最小持续时间就更新
			if curDuration < smallestDuration || smallestDuration == 0 {
				smallestDuration = curDuration
			}
		}
	}
	// 更新table的定时器持续时间
	ct.cleanupDuration = smallestDuration
	if smallestDuration > 0 {
		// 如果当前持续时间大于0，则代表需要继续更新，time.AfterFunc是非阻塞的延时函数
		// 它会在一段时间后创建协程再次进行超时检查
		ct.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go ct.expirationCheck()
		})
	}
	ct.Unlock()
}

// 增加缓存项
func (ct *CacheTable) addInternal(item *CacheItem) {
	ct.log("向", ct.name, "缓存表中插入数据，key是", item.Key(), "lifeSpan是", item.LifeSpan())
	ct.Lock()
	ct.items[item.key] = item
	expDur := ct.cleanupDuration
	addedItem := ct.addedItem
	ct.Unlock()

	// 在插入数据后执行回调函数
	if addedItem != nil {
		for _, callback := range addedItem {
			callback(item)
		}
	}

	// 首先当存活时间大于0时，需要进行超时检查
	// 如果没有设置定时器，或当前的存活时间小于当前表记录的最短存活时间，立即进行一次超时检查
	if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
		ct.expirationCheck()
	}
}

// Add 新增缓存项，传入键值对和存活时间
func (ct *CacheTable) Add(key, data interface{}, lifeSpan time.Duration) *CacheItem {
	item := NewCacheItem(key, data, lifeSpan)

	ct.addInternal(item)

	return item
}

// 删除缓存项
func (ct *CacheTable) deleteInternal(key interface{}) (*CacheItem, error) {
	ct.Lock()
	item, ok := ct.items[key]
	if !ok {
		return nil, ErrCacheNotFound
	}
	deletedItem := ct.deletedItem
	// 调用缓存表删除之前的回调函数
	if deletedItem != nil {
		for _, callback := range deletedItem {
			callback(item)
		}
	}
	// 调用缓存项删除之前的回调函数
	item.RLock()
	defer item.RUnlock()
	if item.aboutToExpire != nil {
		for _, callback := range item.aboutToExpire {
			callback(key)
		}
	}
	ct.log("删除了位于缓存表", ct.name, "中名为", key, "缓存项，创建时间是：", item.createTime, "访问次数是：", item.accessCount)
	delete(ct.items, key)
	ct.Unlock()
	return item, nil
}

// Delete 删除缓存项，传入键
func (ct *CacheTable) Delete(key interface{}) (*CacheItem, error) {
	return ct.deleteInternal(key)
}

// Exists 通过键检查缓存项是否存在，如果不存在不会进行创建
func (ct *CacheTable) Exists(key interface{}) bool {
	ct.RLock()
	defer ct.RUnlock()
	_, ok := ct.items[key]

	return ok
}

// NotFoundAdd 通过键检查缓存项是否存在，如果不存在就会进行创建，不会执行loadData
func (ct *CacheTable) NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool {
	ct.Lock()

	if _, ok := ct.items[key]; ok {
		ct.Unlock()
		return false
	}
	ct.Unlock()
	item := NewCacheItem(key, data, lifeSpan)
	ct.addInternal(item)

	return true
}

// Value 根据键获取值，并延长存活时间，如果未设置loadData不会创建新的缓存项，可传入参数为loadData函数使用
func (ct *CacheTable) Value(key interface{}, args ...interface{}) (*CacheItem, error) {
	ct.RLock()

	r, ok := ct.items[key]
	loadData := ct.loadData
	ct.RUnlock()
	if ok {
		// 更新缓存项的访问次数和最后访问时间
		r.KeepAlive()
		return r, nil
	}

	// 如果缓存不存在且存在loadData回调函数，那么就执行loadData，并创建缓存项
	if loadData != nil {
		item := loadData(key, args...)
		if item != nil {
			ct.Add(key, item.data, item.lifeSpan)
			return item, nil
		}
		return nil, ErrCacheNotFoundOrLoadable
	}
	return nil, ErrCacheNotFound
}

// Flush 清空缓存表
func (ct *CacheTable) Flush() {
	ct.Lock()
	defer ct.Unlock()

	ct.log("清空", ct.name, "缓存表")

	ct.items = make(map[interface{}]*CacheItem)
	ct.cleanupDuration = 0
	if ct.cleanupTimer != nil {
		ct.cleanupTimer.Stop()
	}
}

// 打印日志
func (ct *CacheTable) log(v ...interface{}) {
	if ct.logger == nil {
		return
	}
	ct.logger.Println(v...)
}

// CacheItemPair 存储键和访问次数
type CacheItemPair struct {
	Key         interface{}
	AccessCount int64
}

// CacheItemPairList 对缓存项按照访问次数排序，实现了sort包下的interface
type CacheItemPairList []CacheItemPair

func (p CacheItemPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CacheItemPairList) Len() int           { return len(p) }
func (p CacheItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }

// MostAccessed 返回最多访问的缓存项，传入限制个数
func (ct *CacheTable) MostAccessed(count int64) []*CacheItem {
	ct.RLock()
	defer ct.RUnlock()

	p := make(CacheItemPairList, len(ct.items))
	i := 0
	for k, v := range ct.items {
		p[i] = CacheItemPair{k, v.accessCount}
		i++
	}
	sort.Sort(p)

	var r []*CacheItem
	c := int64(0)
	for _, v := range p {
		if c >= count {
			break
		}

		item, ok := ct.items[v.Key]
		if ok {
			r = append(r, item)
		}
		c++
	}

	return r
}
