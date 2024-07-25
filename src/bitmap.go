package main

//import "sync"

const DEFAULTMAPSIZE = 33124

type Bitmap struct {
	bits []uint64
	//mutex sync.RWMutex
}

// NewBitmap 创建一个新的位图
func NewBitmap(size int) *Bitmap {
	if size <= 0 {
		size = DEFAULTMAPSIZE
	}
	return &Bitmap{
		bits: make([]uint64, (size+63)/64),
	}
}

// Set 设置指定位置的位为1，如果需要扩容，则自动扩容
func (b *Bitmap) Set(pos int) {
	if pos < 0 {
		return
	}
	index := pos / 64

	//b.mutex.Lock()
	//defer b.mutex.Unlock()

	if index >= len(b.bits) {
		b.expand(index + 1)
	}
	b.bits[index] |= 1 << (pos % 64)
}

// Clear 清除指定位置的位
func (b *Bitmap) Clear(pos int) {
	if pos < 0 {
		return
	}
	index := pos / 64

	//b.mutex.Lock()
	//defer b.mutex.Unlock()

	if index < len(b.bits) {
		b.bits[index] &^= 1 << (pos % 64)
	}
}

// IsSet 检查指定位置的位是否为1
func (b *Bitmap) IsSet(pos int) bool {
	if pos < 0 {
		return false
	}
	index := pos / 64

	//b.mutex.RLock()
	//defer b.mutex.RUnlock()

	if index < len(b.bits) {
		return b.bits[index]&(1<<(pos%64)) != 0
	}
	return false
}

// expand 扩容位图
func (b *Bitmap) expand(newSize int) {
	if newSize > len(b.bits) {
		newBits := make([]uint64, newSize)
		copy(newBits, b.bits)
		b.bits = newBits
	}
}
