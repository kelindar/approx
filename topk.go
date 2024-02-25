// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Implementation inspired by https://github.com/tylertreat/BoomFilters, which is licensed
// under Apache 2.0 License. Copyright (c) Tyler Treat.

package approx

import (
	"container/heap"
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

// An minheap is a min-heap of top values, ordered by count. It is used to track
// the top-k elements in a stream.
type minheap []TopValue

// Len, Less, Swap implement the sort.Interface.
func (e *minheap) Len() int           { return len(*e) }
func (e *minheap) Less(i, j int) bool { return (*e)[i].Count < (*e)[j].Count }
func (e *minheap) Swap(i, j int)      { (*e)[i], (*e)[j] = (*e)[j], (*e)[i] }

// Push implements the heap.Interface.
func (e *minheap) Push(x any) {
	*e = append(*e, x.(TopValue))
}

// Pop implements the heap.Interface.
func (e *minheap) Pop() any {
	old := *e
	n := len(old)
	x := old[n-1]
	*e = old[0 : n-1]
	return x
}

// TopK uses a Count-Min Sketch to calculate the top-K frequent elements in a
// stream.
type TopK struct {
	mu        sync.Mutex
	min, size atomic.Uint32
	maxSize   uint
	cms       *CountMin
	elements  minheap
}

// NewTopK creates a new structure to track the top-k elements in a stream. The k parameter
// specifies the number of elements to track.
func NewTopK(k uint) (*TopK, error) {
	cms, err := NewCountMinWithSize(4, 2048)
	if err != nil {
		return nil, err
	}

	elements := make(minheap, 0, k)
	heap.Init(&elements)
	return &TopK{
		cms:      cms,
		maxSize:  k,
		elements: elements,
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
	if t.elements.Len() == int(t.maxSize) && count < t.elements[0].Count {
		return
	}

	// If the element is already in the top-k, update it's count
	for i := range t.elements {
		if elem := &t.elements[i]; hash == elem.hash {
			elem.Count = count
			heap.Fix(&t.elements, i)
			//t.min.Store((*t.elements)[0].Count)
			return
		}
	}

	// Remove minimum-frequency element.
	if t.elements.Len() == int(t.maxSize) {
		heap.Pop(&t.elements)
	} else {
		t.size.Store(uint32(t.elements.Len()))
	}

	// Add element to top-k and update min count
	heap.Push(&t.elements, TopValue{Value: value, hash: hash, Count: count})
	t.min.Store(t.elements[0].Count)
}

// Values returns the top-k elements from lowest to highest frequency.
func (t *TopK) Values() []TopValue {
	output := make(minheap, 0, t.maxSize)

	// Copy the elemenst into a new slice
	t.mu.Lock()
	for _, e := range t.elements {
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
	elements := make(minheap, 0, t.maxSize)
	heap.Init(&elements)
	t.elements = elements
	return t
}
