// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Implementation inspired by https://github.com/tylertreat/BoomFilters, which is licensed
// under Apache 2.0 License. Copyright (c) Tyler Treat.

package approx

import (
	"sort"
	"sync"

	"github.com/axiomhq/hyperloglog"
	"github.com/zeebo/xxh3"
)

// TopValue represents a value and its associated count.
type TopValue struct {
	hash  uint64 `json:"-"`     // The hash of the value
	Value string `json:"value"` // The associated value
	Count uint32 `json:"count"` // The count of the value
}

// TopK uses a Count-Min Sketch to calculate the top-K frequent elements in a
// stream.
type TopK struct {
	mu   sync.Mutex
	heap minheap
	cms  *CountMin
	hll  *hyperloglog.Sketch
}

// NewTopK creates a new structure to track the top-k elements in a stream. The k parameter
// specifies the number of elements to track.
func NewTopK(k uint) (*TopK, error) {
	cms, err := NewCountMin()
	if err != nil {
		return nil, err
	}

	return &TopK{
		cms:  cms,
		heap: make(minheap, 0, k),
		hll:  hyperloglog.New(),
	}, nil
}

// Update adds the binary value to Count-Min Sketch and updates the top-k elements.
func (t *TopK) Update(value string) {
	hash := xxh3.HashString(value)
	if updated := t.cms.UpdateHash(hash); !updated {
		return // Estimate hasn't changed, skip
	}

	// Try to insert the value into the top-k heap
	count := uint32(t.cms.CountHash(hash))
	t.tryInsert(value, hash, count)
}

// tryInsert adds the data to the top-k heap. If the data is already an element,
// the frequency is updated. If the heap already has k elements, the element
// with the minimum frequency is removed.
func (t *TopK) tryInsert(value string, hash uint64, count uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Add the element to HyperLogLog
	t.hll.InsertHash(hash)
	if cap(t.heap) == 0 {
		return // no tracking
	}

	// If the element is not in the top-k, skip
	if len(t.heap) == cap(t.heap) && count < t.heap[0].Count {
		return
	}

	// If the element is already in the top-k, update it's count
	for i := range t.heap {
		if elem := &t.heap[i]; hash == elem.hash {
			t.heap.Update(i, count)
			return
		}
	}

	// Remove minimum-frequency element.
	if len(t.heap) == cap(t.heap) {
		t.heap.Pop()
	}

	// Copy the string in case the caller reuses the buffer
	clone := string(append([]byte(nil), value...))

	// Add element to top-k and update min count
	t.heap.Push(TopValue{Value: clone, hash: hash, Count: count})
}

// Values returns the top-k elements from lowest to highest frequency.
func (t *TopK) Values() []TopValue {
	t.mu.Lock()
	output := make(minheap, 0, cap(t.heap))
	t.heap.Clone(&output)
	t.mu.Unlock()

	// Sort the elements before returning
	sort.Sort(&output)
	return output
}

// Cardinality returns the estimated cardinality of the stream.
func (t *TopK) Cardinality() uint {
	t.mu.Lock()
	defer t.mu.Unlock()

	return uint(t.hll.Estimate())
}

// Reset restores the TopK to its original state. The function returns the top-k
// elements and their counts as well as the estimated cardinality of the stream.
func (t *TopK) Reset(k int) ([]TopValue, uint) {
	t.mu.Lock()
	output := make(minheap, 0, cap(t.heap))
	n := t.hll.Estimate() // Estimate the cardinality
	t.heap.Clone(&output) // Clone the top-k elements
	t.resize(k)           // Resize the top-k heap
	t.mu.Unlock()

	// Sort the elements before returning
	sort.Sort(&output)
	return output, uint(n)
}

// reset resizes the top-k heap and resets the Count-Min Sketch and HyperLogLog.
func (t *TopK) resize(k int) {
	switch {
	case k <= 0:
		t.heap = make(minheap, 0, 0)
	case k != cap(t.heap):
		t.heap = make(minheap, 0, k)
	case k == cap(t.heap):
		t.heap.Reset()
	}

	// Reset the Count-Min Sketch and HyperLogLog
	t.cms.Reset()
	t.hll = hyperloglog.New()
}
