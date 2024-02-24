package approx

import (
	"container/heap"
	"sync"
	"sync/atomic"

	"github.com/zeebo/xxh3"
)

// Element represents a value and it's associated count
type Element struct {
	Hash  uint64 // The hash of the value
	Value []byte // The associated value
	Count uint32 // The count of the value
}

// An minheap is a min-heap of elements.
type minheap []Element

func (e *minheap) Len() int           { return len(*e) }
func (e *minheap) Less(i, j int) bool { return (*e)[i].Count < (*e)[j].Count }
func (e *minheap) Swap(i, j int)      { (*e)[i], (*e)[j] = (*e)[j], (*e)[i] }

func (e *minheap) Push(x any) {
	*e = append(*e, x.(Element))
}

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
	k         uint
	cms       *CountMin
	elements  minheap
}

// NewTopK creates a new TopK backed by a Count-Min sketch whose relative
// accuracy is within a factor of epsilon with probability delta. It tracks the
// k-most frequent elements.
func NewTopK(k uint) (*TopK, error) {
	cms, err := NewCountMinWithSize(4, 2048)
	if err != nil {
		return nil, err
	}

	elements := make(minheap, 0, k)
	heap.Init(&elements)
	return &TopK{
		cms:      cms,
		k:        k,
		elements: elements,
	}, nil
}

// Update will add the data to the Count-Min Sketch and update the top-k heap if
// applicable.
func (t *TopK) Update(data []byte) {
	hash := xxh3.Hash(data)
	count := uint32(t.cms.UpdateHash(hash))

	if t.isTop(count) {
		t.insert(data, hash, count)
	}
}

// isTop indicates if the given frequency falls within the top-k heap.
func (t *TopK) isTop(count uint32) bool {
	return t.min.Load() <= count || uint(t.size.Load()) < t.k
}

// insert adds the data to the top-k heap. If the data is already an element,
// the frequency is updated. If the heap already has k elements, the element
// with the minimum frequency is removed.
func (t *TopK) insert(value []byte, hash uint64, count uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Same check as isTop, but protected by mutex to ensure consistency
	if t.elements.Len() == int(t.k) && count < t.elements[0].Count {
		return
	}

	// If the element is already in the top-k, update it's count
	for i := range t.elements {
		if elem := &t.elements[i]; hash == elem.Hash {
			elem.Count = count
			heap.Fix(&t.elements, i)
			//t.min.Store((*t.elements)[0].Count)
			return
		}
	}

	// Remove minimum-frequency element.
	if t.elements.Len() == int(t.k) {
		heap.Pop(&t.elements)
	} else {
		t.size.Store(uint32(t.elements.Len()))
	}

	// Add element to top-k and update min count
	heap.Push(&t.elements, Element{Value: value, Hash: hash, Count: count})
	t.min.Store(t.elements[0].Count)
}

// Elements returns the top-k elements from lowest to highest frequency.
func (t *TopK) Elements() []Element {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.elements.Len() == 0 {
		return make([]Element, 0)
	}

	elements := make(minheap, t.elements.Len())
	copy(elements, t.elements)
	heap.Init(&elements)
	topK := make([]Element, 0, t.k)

	for elements.Len() > 0 {
		topK = append(topK, heap.Pop(&elements).(Element))
	}

	return topK
}

// Reset restores the TopK to its original state. It returns itself to allow
// for chaining.
func (t *TopK) Reset() *TopK {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cms.Reset()
	elements := make(minheap, 0, t.k)
	heap.Init(&elements)
	t.elements = elements
	return t
}
