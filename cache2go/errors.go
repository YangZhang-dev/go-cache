package cache2go

import "errors"

var (
	ErrCacheNotFound           = errors.New("缓存项不存在")
	ErrCacheNotFoundOrLoadable = errors.New("缓存项不存在并且未能加入缓存表中")
)
