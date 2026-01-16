package util

import "sync"

type RingBuffer struct {
	lines    []string
	head     int
	size     int
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
		size:     0,
		capacity: capacity,
	}
}

func (b *RingBuffer) Append(l string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines[b.head] = l
	b.head = (b.head + 1) % b.capacity
	if b.size < b.capacity {
		b.size++
	}
}

func (b *RingBuffer) Recent(n int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.size == 0 {
		return []string{}
	}
	n = min(n, b.size)
	result := make([]string, n)
	if b.size < b.capacity {
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
	return b.Recent(b.size)
}

func (b *RingBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

func (b *RingBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = make([]string, b.capacity)
	b.size = 0
	b.head = 0
}
