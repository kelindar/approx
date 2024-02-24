package approx

import (
	"errors"
	"math"
	"sync/atomic"

	"github.com/zeebo/xxh3"
)

const (
	defaultEpsilon    = 0.001
	defaultConfidence = 0.99
)

// CountMin is a sketch data structure for estimating the frequency of items in a stream
type CountMin struct {
	total  uint64   // total number of items seen
	depth  int      // number of hash functions
	width  int      // number of counters per hash function
	counts []uint32 // 2D array of counters
}

// NewCountMin creates a new CountMin sketch with default epsilon and confidence
func NewCountMin() (*CountMin, error) {
	return NewCountMinWithEstimates(defaultEpsilon, defaultConfidence)
}

// NewCountMinWithEpsilon creates a new CountMin sketch with the given epsilon and delta. The epsilon
// parameter controls the accuracy of the estimates, and the confidence parameter controls the
// probability that the estimates are within the specified error bounds.
func NewCountMinWithEstimates(epsilon, confidence float64) (*CountMin, error) {
	switch {
	case epsilon <= 0 || epsilon >= 1:
		return nil, errors.New("sketch: value of epsilon should be in range of (0, 1)")
	case confidence <= 0 || confidence >= 1:
		return nil, errors.New("sketch: value of delta should be in range of (0, 1)")
	}

	delta := 1 - confidence
	width := uint(math.Ceil(math.E / epsilon))
	depth := uint(math.Ceil(math.Log(1 / delta)))
	return NewCountMinWithSize(depth, width)
}

// NewCountMinWithSize creates a new CountMin sketch with the given depth and width
func NewCountMinWithSize(depth, width uint) (*CountMin, error) {
	switch {
	case depth > 128:
		return nil, errors.New("sketch: depth should be less than 128")
	case width > math.MaxInt32:
		return nil, errors.New("sketch: width should be less than MaxInt32")
	}

	return &CountMin{
		depth:  int(depth),
		width:  int(width),
		counts: make([]uint32, depth*width),
	}, nil
}

// CountTotal returns the total number of items seen
func (c *CountMin) CountTotal() uint {
	return uint(atomic.LoadUint64(&c.total))
}

// Update increments the counter for the given item
func (c *CountMin) Update(item []byte) uint {
	return c.UpdateHash(xxh3.Hash(item))
}

// UpdateString increments the counter for the given item
func (c *CountMin) UpdateString(item string) uint {
	return c.UpdateHash(xxh3.HashString(item))
}

// UpdateHash increments the counter for the given item
func (c *CountMin) UpdateHash(hash uint64) uint {
	lo := hash & ((1 << 32) - 1) // Lower 32 bits
	hi := hash >> 32             // Upper 32 bits

	// Increment the total count, atomically
	atomic.AddUint64(&c.total, 1)

	// Find the minimum counter value and increment the counter at the given index
	x := ^uint32(0)
	w := c.width
	for i := 0; i < c.depth; i++ {
		hx := lo + uint64(i)*hi
		at := &c.counts[i*w+int(hx)%w]
		x = min(x, atomic.AddUint32(at, 1))
	}

	return uint(x)
}

// Count returns the estimated frequency of the given item
func (c *CountMin) Count(item []byte) uint {
	return c.CountHash(xxh3.Hash(item))
}

// CountString returns the estimated frequency of the given item
func (c *CountMin) CountString(item string) uint {
	return c.CountHash(xxh3.HashString(item))
}

// CountHash returns the estimated frequency of the given item
func (c *CountMin) CountHash(hash uint64) uint {
	lo := hash & ((1 << 32) - 1) // Lower 32 bits
	hi := hash >> 32             // Upper 32 bits

	x := ^uint32(0)
	w := c.width
	for i := 0; i < c.depth && x > 0; i++ {
		hx := lo + uint64(i)*hi
		at := &c.counts[i*w+int(hx)%w]
		x = min(x, atomic.LoadUint32(at))
	}
	return uint(x)
}

// Reset sets all counters to zero
func (c *CountMin) Reset() {
	atomic.StoreUint64(&c.total, 0)
	for i := range c.counts {
		atomic.StoreUint32(&c.counts[i], 0)
	}
}
