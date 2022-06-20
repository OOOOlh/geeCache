package geecache

//表示缓存值
type ByteView struct {
	b []byte
}

//返回字节数组长度
func (v ByteView) Len() int {
	return len(v.b)
}

//该方法返回字节数组的拷贝
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//该方法将字节数组转为字符串
func (v ByteView) String() string {
	return string(v.b)
}

//拷贝方法的具体实现。由于ByteView是只读的，所以用copy，防止缓存值被外部程序修改
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
