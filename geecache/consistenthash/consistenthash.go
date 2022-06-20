package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	//hash函数
	hash Hash
	//虚拟节点倍数
	replicas int
	//哈希环
	keys []int // Sorted
	//虚拟节点与真实节点的映射表
	//虚拟节点的hash值和虚拟节点名称
	hashMap map[int]string
}

// New creates a Map instance
//允许自定义虚拟节点和Hash函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

//实现添加真实节点/机器的Add()方法
// Add adds some keys to the hash.
//对于每一个真实节点，创建m.replicas个虚拟节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//获得虚拟节点的hash值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//将hash值放入哈希环中
			m.keys = append(m.keys, hash)
			//添加虚拟节点与真实节点映射
			m.hashMap[hash] = key
		}
	}
	//对哈希环进行排序
	sort.Ints(m.keys)
}

//实现选择节点的Get()方法
//用二分法，将目标hash值的下一个hash节点作为存储该key的虚拟节点
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	//Binary search for appropriate replica.
	//二分法找hash值
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	//根据hash值找到虚拟节点名称
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
