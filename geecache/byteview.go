package geecache

type ByteView struct {
	// 使用字节数据可存储各种类型，包括图片
	b []byte
}

// Len 实现Value接口
func (view ByteView) Len() int {
	return len(view.b)
}

// String 字符串表示
func (view ByteView) String() string {
	return string(view.b)
}

// ByteSlice 返回缓存项的副本
func (view ByteView) ByteSlice() []byte {
	return view.cloneBytes(view.b)
}

// 创建缓存项的副本，防止缓存被改变
func (view ByteView) cloneBytes(b []byte) []byte {
	res := make([]byte, len(b))
	copy(res, b)
	return res
}
