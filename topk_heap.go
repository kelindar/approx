package approx

// minheap is a min-heap of top values, ordered by count.
type minheap []TopValue

// Reset resets the minheap to an empty state.
func (h *minheap) Reset() {
	*h = (*h)[:0]
}

// Len, Less, Swap implement the sort.Interface.
func (h *minheap) Len() int           { return len(*h) }
func (h *minheap) Less(i, j int) bool { return (*h)[i].Count < (*h)[j].Count }
func (h *minheap) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

// Push adds a new element to the heap.
func (h *minheap) Push(x TopValue) {
	*h = append(*h, x)
	h.up(h.Len() - 1)
}

// Pop returns the minimum element from the heap.
func (h *minheap) Pop() TopValue {
	n := h.Len() - 1
	h.Swap(0, n)
	h.down(0, n)

	// Pop the last element
	x := (*h)[n]
	*h = (*h)[:n]
	return x
}

// Update updates the count of the element at index i.
func (h minheap) Update(i int, count uint32) {
	h[i].Count = count
	if !h.down(i, len(h)) {
		h.up(i)
	}
}

func (h minheap) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !(h[j].Count < h[i].Count) {
			break
		}

		h[i], h[j] = h[j], h[i]
		j = i
	}
}

func (h minheap) down(at, n int) bool {
	i := at
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && (h[j2].Count < h[j1].Count) {
			j = j2 // = 2*i + 2  // right child
		}
		if h[i].Count < h[j].Count {
			break
		}

		h[i], h[j] = h[j], h[i]
		i = j
	}
	return i > at
}
