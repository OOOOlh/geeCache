package geecache

import (
	"fmt"
	"geeCache/geecache/singleflight"
	pb "geecache/geecachepb"
	"log"
	"sync"
)

//回调Getter,如果缓存没命中，又不想从远程节点获取数据，就调用这个回调方法。该方法由用户自己实现，即用户自己决定缓存没命中时要怎么做
//接口类型
type Getter interface {
	Get(key string) ([]byte, error)
}

//函数类型
type GetterFunc func(key string) ([]byte, error)

//函数类型实现了Getter接口，称为接口型函数。方便调用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//
type Group struct {
	//Group名字
	name string
	//缓存未名中时获取源数据的回调(callback)
	getter Getter
	//一开始实现的并发缓存
	mainCache cache

	peers PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

//实例化Group，并将group存储在全局变量groups中
//输入Group名字和最大值以及缓存未命中时的回调
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}

	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//根据key得到value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	//如果缓存命中
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	//如果缓存没命中
	return g.load(key)
}

//从本地节点找
func (g *Group) getLocally(key string) (ByteView, error) {

	//回调函数
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	//将源数据添加到cache中
	g.populateCache(key, value)
	return value, nil

}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//缓存没命中，就调用该方法
func (g *Group) load(key string) (value ByteView, err error) {

	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	//并发场景下，针对相同的key,load过程只会调用一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {

		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
