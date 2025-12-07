package audio

import (
	"fmt"
	"sync"
)

// RingBuffer is a circular buffer for audio data
// It provides thread-safe read/write operations for streaming audio
type RingBuffer struct {
	mu       sync.RWMutex
	buffer   []byte
	size     int
	writePos int
	readPos  int
	full     bool
}

// NewRingBuffer creates a new ring buffer with the specified size in bytes
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]byte, size),
		size:   size,
	}
}

// Write writes data to the buffer
// Returns the number of bytes written and an error if the buffer is full
func (rb *RingBuffer) Write(data []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.full {
		return 0, fmt.Errorf("buffer is full")
	}

	bytesWritten := 0
	for _, b := range data {
		rb.buffer[rb.writePos] = b
		rb.writePos = (rb.writePos + 1) % rb.size
		bytesWritten++

		if rb.writePos == rb.readPos {
			rb.full = true
			break
		}
	}

	return bytesWritten, nil
}

// Read reads up to len(data) bytes from the buffer
// Returns the number of bytes read
func (rb *RingBuffer) Read(data []byte) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.readPos == rb.writePos && !rb.full {
		return 0 // Buffer is empty
	}

	bytesRead := 0
	for i := range data {
		data[i] = rb.buffer[rb.readPos]
		rb.readPos = (rb.readPos + 1) % rb.size
		rb.full = false
		bytesRead++

		if rb.readPos == rb.writePos {
			break
		}
	}

	return bytesRead
}

// Available returns the number of bytes available to read
func (rb *RingBuffer) Available() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.full {
		return rb.size
	}

	if rb.writePos >= rb.readPos {
		return rb.writePos - rb.readPos
	}

	return rb.size - rb.readPos + rb.writePos
}

// Free returns the number of bytes available to write
func (rb *RingBuffer) Free() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.full {
		return 0
	}

	return rb.size - rb.Available()
}

// Reset clears the buffer
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.readPos = 0
	rb.writePos = 0
	rb.full = false
}

// Size returns the total size of the buffer
func (rb *RingBuffer) Size() int {
	return rb.size
}

// IsFull returns true if the buffer is full
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.full
}

// IsEmpty returns true if the buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.readPos == rb.writePos && !rb.full
}
