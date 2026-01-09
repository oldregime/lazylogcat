package util

import "sync"

type RingBuffer struct {
	Size int

	lines    []string
	head     int
	capacity int
	mu       sync.RWMutex
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	return &RingBuffer{
		lines:    make([]string, capacity),
		head:     0,
		Size:     0,
		capacity: capacity,
	}
}

func (b *RingBuffer) Append(l string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines[b.head] = l
	b.head = (b.head + 1) % b.capacity
	if b.Size < b.capacity {
		b.Size++
	}
}

func (b *RingBuffer) Recent(n int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.Size == 0 {
		return []string{}
	}
	n = min(n, b.Size)
	result := make([]string, n)
	if b.Size < b.capacity {
		copy(result, b.lines[b.head-n:b.head])
		return result
	}
	for i := range n {
		index := (b.head - n + b.capacity + i) % b.capacity
		result[i] = b.lines[index]
	}
	return result
}

func (b *RingBuffer) All() []string {
	return b.Recent(b.Size)
}
