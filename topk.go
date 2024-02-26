// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Implementation inspired by https://github.com/tylertreat/BoomFilters, which is licensed
// under Apache 2.0 License. Copyright (c) Tyler Treat.

package approx

import (
	"sort"
	"sync"
	"unsafe"

	"github.com/axiomhq/hyperloglog"
	"github.com/zeebo/xxh3"
)

// TopValue represents a value and its associated count.s
type TopValue struct {
	hash  uint64 `json:"-"`     // The hash of the value
	Value []byte `json:"value"` // The associated value
	Count uint32 `json:"count"` // The count of the value
}

// TopK uses a Count-Min Sketch to calculate the top-K frequent elements in a
// stream.
type TopK struct {
	mu   sync.Mutex
	size uint
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
		size: k,
		heap: make(minheap, 0, k),
		hll:  hyperloglog.New(),
	}, nil
}

// UpdateString adds the string value to the Count-Min Sketch and updates the top-k heap.
func (t *TopK) UpdateString(value string) {
	t.Update(unsafe.Slice(unsafe.StringData(value), len(value)))
}

// Update adds the binary value to Count-Min Sketch and updates the top-k elements.
func (t *TopK) Update(value []byte) {
	hash := xxh3.Hash(value)
	if updated := t.cms.UpdateHash(hash); !updated {
		return // Estimate hasn't changed, skip
	}

	// Try to insert the value into the top-k heap
	count := uint32(t.cms.CountHash(hash))
	t.tryInsert(value, hash, count)
}

// isTop indicates if the given frequency falls within the top-k heap.
func (t *TopK) isTop(count uint32) bool {
	return t.heap.Len() < int(t.size) || count >= t.heap[0].Count
}

// tryInsert adds the data to the top-k heap. If the data is already an element,
// the frequency is updated. If the heap already has k elements, the element
// with the minimum frequency is removed.
func (t *TopK) tryInsert(value []byte, hash uint64, count uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Add the element to HyperLogLog
	t.hll.InsertHash(hash)

	// If the element is not in the top-k, skip
	if !t.isTop(count) {
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
	if t.heap.Len() == int(t.size) {
		t.heap.Pop()
	}

	// Add element to top-k and update min count
	t.heap.Push(TopValue{Value: value, hash: hash, Count: count})
}

// Values returns the top-k elements from lowest to highest frequency.
func (t *TopK) Values() []TopValue {
	output := make(minheap, 0, t.size)
	t.mu.Lock()
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
// elements and their counts.
func (t *TopK) Reset() []TopValue {
	output := make(minheap, 0, t.size)
	t.mu.Lock()
	{ // Clone and reset the heap
		t.heap.Clone(&output)
		t.cms.Reset()
		t.heap.Reset()
		t.hll = hyperloglog.New()
	}
	t.mu.Unlock()

	// Sort the elements before returning
	sort.Sort(&output)
	return output
}
