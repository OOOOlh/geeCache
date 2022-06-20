package singleflight

import "sync"

type call struct {
	//避免重入
	wg  sync.WaitGroup
	val interface{}
	err error
}

//管理不同key的请求
type Group struct {
	mu sync.Mutex // protects m
	m  map[string]*call
}

//针对相同的key，无论Do被调用多少次，函数fn都只会被调用一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()

	//延迟初始化
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	//发起请求前加锁
	c.wg.Add(1)
	//添加到g.m,表明key已经有对应的请求在处理
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err

}
