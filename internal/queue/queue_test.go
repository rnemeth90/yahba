package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	size := 100
	q := NewCircularBuffer(size)

	for i := 0; i < (size - 1); i++ {
		assert.NoError(t, q.Enqueue(i+1))
	}
	// can't insert new data.
	assert.ErrorIs(t, q.Enqueue(0), ErrFull)

	for i := 0; i < (size - 1); i++ {
		v, err := q.Dequeue()
		assert.Equal(t, i+1, v.(int))
		assert.NoError(t, err)
	}

	_, err := q.Dequeue()
	// no task
	assert.ErrorIs(t, err, ErrEmpty)
}

func TestCircularBufferPushEvictsOldest(t *testing.T) {
	q := NewCircularBuffer(3)

	q.Push(1)
	q.Push(2)
	assert.Equal(t, 2, q.Len())

	// Buffer usable capacity is one less than size (one slot always kept
	// empty to distinguish full from empty), so this third push already
	// evicts the oldest item (1).
	q.Push(3)
	q.Push(4)

	assert.Equal(t, []T{3, 4}, q.Items())
}
