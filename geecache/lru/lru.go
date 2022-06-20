package lru

import "container/list"

type Cache struct {
	//允许使用的最大内存
	maxBytes int64
	//当前已经使用的内存
	nbytes int64
	//使用Go语言标准库实现的双向链表list.List。链表中存的是entry
	ll *list.List
	//字典，键是字符串，值是双向链表中对应节点的指针
	cache map[string]*list.Element
	//记录某条记录被移除时的回调函数，可以为nil
	OnEvicted func(key string, value Value)
}

//双向链表节点的数据类型。除了value，也保存了key。这样可以在淘汰队首节点时，用key从字典中删除对应的映射
type entry struct {
	key   string
	value Value
}

//为了通用性，允许值是实现了Value接口当任意类型。该接口只包含了一个方法，用于返回值所占用当内存大小
type Value interface {
	Len() int
}

//cache实例化。允许用户自定义cache的最大值，以及回调函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//从Cache中查找。先根据key找到map中对应的list,然后在节点值(entry)中找到value。并将list中对应值移动到队尾（在这里约定Front为队尾）
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//删除
//缓存淘汰，移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	//获取list中第一个值，将其删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		//并将cache中对应的键值对删除
		delete(c.cache, kv.key)
		//更新当前所用内存
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//如果回调函数不为nil,则调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

//新增/修改
func (c *Cache) Add(key string, value Value) {
	//根据key，如果存在，就将list节点移动到队尾，并修改对应的value
	//如果不存在，就在list中新建一个元素，并在map中添加
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	//当前的字节数大于最大字节数时，移除最后一个
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

//占用内存的大小
func (c *Cache) Len() int {
	//返回当前链表的长度
	return c.ll.Len()
}
