// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Implementation inspired by https://github.com/tylertreat/BoomFilters, which is licensed
// under Apache 2.0 License. Copyright (c) Tyler Treat.

package approx

import (
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/xxh3"
)

// TopValue represents a value and its associated count.s
type TopValue struct {
	hash  uint64 // The hash of the value
	Value []byte `json:"value"` // The associated value
	Count uint32 `json:"count"` // The count of the value
}

// TopK uses a Count-Min Sketch to calculate the top-K frequent elements in a
// stream.
type TopK struct {
	mu        sync.Mutex
	min, size atomic.Uint32
	maxSize   uint
	cms       *CountMin
	heap      minheap
}

// NewTopK creates a new structure to track the top-k elements in a stream. The k parameter
// specifies the number of elements to track.
func NewTopK(k uint) (*TopK, error) {
	cms, err := NewCountMin()
	if err != nil {
		return nil, err
	}

	return &TopK{
		cms:     cms,
		maxSize: k,
		heap:    make(minheap, 0, k),
	}, nil
}

// UpdateString adds the string value to the Count-Min Sketch and updates the top-k heap.
func (t *TopK) UpdateString(value string) {
	t.Update(unsafe.Slice(unsafe.StringData(value), len(value)))
}

// Update adds the binary value to Count-Min Sketch and updates the top-k elements.
func (t *TopK) Update(value []byte) {
	hash := xxh3.Hash(value)
	if count := uint32(t.cms.UpdateHash(hash)); t.isTop(count) {
		t.insert(value, hash, count)
	}
}

// isTop indicates if the given frequency falls within the top-k heap.
func (t *TopK) isTop(count uint32) bool {
	return t.min.Load() <= count || uint(t.size.Load()) < t.maxSize
}

// insert adds the data to the top-k heap. If the data is already an element,
// the frequency is updated. If the heap already has k elements, the element
// with the minimum frequency is removed.
func (t *TopK) insert(value []byte, hash uint64, count uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Same check as isTop, but protected by mutex to ensure consistency
	if t.heap.Len() == int(t.maxSize) && count < t.heap[0].Count {
		return
	}

	// If the element is already in the top-k, update it's count
	for i := range t.heap {
		if elem := &t.heap[i]; hash == elem.hash {
			t.heap.Update(i, count)
			//t.min.Store((*t.elements)[0].Count)
			return
		}
	}

	// Remove minimum-frequency element.
	if t.heap.Len() == int(t.maxSize) {
		t.heap.Pop()
	} else {
		t.size.Store(uint32(t.heap.Len()))
	}

	// Add element to top-k and update min count
	t.heap.Push(TopValue{Value: value, hash: hash, Count: count})
	t.min.Store(t.heap[0].Count)
}

// Values returns the top-k elements from lowest to highest frequency.
func (t *TopK) Values() []TopValue {
	output := make(minheap, 0, t.maxSize)

	// Copy the elemenst into a new slice
	t.mu.Lock()
	for _, e := range t.heap {
		if e.Count > 0 {
			output = append(output, e)
		}
	}
	t.mu.Unlock()

	// Sort the elements before returning
	sort.Sort(&output)
	return output
}

// Reset restores the TopK to its original state.
func (t *TopK) Reset() *TopK {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cms.Reset()
	t.heap.Reset()
	return t
}
