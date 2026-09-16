package queue

import (
	"errors"
	"sync"
)

type T interface{}

// ErrFull is returned by Enqueue when the buffer has no free slots.
var ErrFull = errors.New("queue: buffer is full")

// ErrEmpty is returned by Dequeue when the buffer holds no items.
var ErrEmpty = errors.New("queue: buffer is empty")

type CircularBuffer struct {
	sync.Mutex
	taskQueue []T
	capacity  int
	head      int
	tail      int
}

func (s *CircularBuffer) IsEmpty() bool {
	return s.head == s.tail
}

func (s *CircularBuffer) IsFull() bool {
	return s.head == (s.tail+1)%s.capacity
}

func (s *CircularBuffer) Enqueue(task T) error {
	if s.IsFull() {
		return ErrFull
	}

	s.Lock()
	s.taskQueue[s.tail] = task
	s.tail = (s.tail + 1) % s.capacity
	s.Unlock()

	return nil
}

func (s *CircularBuffer) Dequeue() (T, error) {
	if s.IsEmpty() {
		return nil, ErrEmpty
	}

	s.Lock()
	data := s.taskQueue[s.head]
	s.head = (s.head + 1) % s.capacity
	s.Unlock()

	return data, nil
}

// Push inserts an item, evicting the oldest item first if the buffer is
// full. Use this instead of Enqueue when the buffer should behave like a
// fixed-size "keep only the most recent N" sliding window rather than
// erroring once full.
func (s *CircularBuffer) Push(item T) {
	if s.IsFull() {
		_, _ = s.Dequeue()
	}
	_ = s.Enqueue(item)
}

// Len returns the number of items currently buffered.
func (s *CircularBuffer) Len() int {
	s.Lock()
	defer s.Unlock()

	if s.tail >= s.head {
		return s.tail - s.head
	}
	return s.capacity - s.head + s.tail
}

// Items returns a snapshot of the buffered items in FIFO (oldest-first)
// order, without removing them.
func (s *CircularBuffer) Items() []T {
	s.Lock()
	defer s.Unlock()

	n := 0
	if s.tail >= s.head {
		n = s.tail - s.head
	} else {
		n = s.capacity - s.head + s.tail
	}

	items := make([]T, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, s.taskQueue[(s.head+i)%s.capacity])
	}
	return items
}

func NewCircularBuffer(size int) *CircularBuffer {
	w := &CircularBuffer{
		taskQueue: make([]T, size),
		capacity:  size,
	}

	return w
}
