package geecache

import (
	"geeCache/geecache/lru"
	"sync"
)

type cache struct {
	mu  sync.Mutex
	lru *lru.Cache
	//缓存最大内存
	cacheBytes int64
}

//操作缓存层面

//添加
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//延迟初始化，提高性能，减少内存要求
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

//根据key获得缓存
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		//强制类型转换
		return v.(ByteView), ok
	}

	return
}
