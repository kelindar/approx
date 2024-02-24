package approx

import (
	"math"
	"math/rand"
	"sync/atomic"
)

// n computes the approximate count based on Morris's algorithm
func n(v, a float64) float64 {
	return a * (math.Pow(1+1/a, v) - 1)
}

// ------------------------------------ Count8 ------------------------------------

// Precompute the lookup table for the 8-bit counter
var n8 []float64 = func() []float64 {
	const scale = 31

	lookup := make([]float64, math.MaxUint8+1)
	for i := range lookup {
		lookup[i] = n(float64(i), scale)
	}
	return lookup
}()

// Count8 is a 8-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~100k with relatively mean error rate of
// around ~10%.
type Count8 uint8

// Estimate returns the estimated count
func (c Count8) Estimate() uint {
	return max(uint(n8[c]), uint(c)) // special case for c=1
}

// Increment increments the counter
func (c *Count8) Increment() uint {
	t0 := n8[*c]
	t1 := n8[*c+1]

	// Check for overflow
	if *c >= math.MaxUint8 {
		return uint(t0)
	}

	// Increment the counter depending on the delta
	if delta := 1 / (t1 - t0); rand.Float64() < delta {
		(*c)++
	}

	return max(uint(t1), uint(*c))
}

// ------------------------------------ Count16 ------------------------------------

// Precompute the lookup table for the 16-bit counter
var n16 []float64 = func() []float64 {
	const scale = 5000

	lookup := make([]float64, math.MaxUint16+1)
	for i := range lookup {
		lookup[i] = n(float64(i), scale)
	}
	return lookup
}()

// Count16 is a 16-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~2 billion with relatively low mean error rate of
// around ~0.50%.
type Count16 uint16

// Estimate returns the estimated count
func (c Count16) Estimate() uint {
	return max(uint(n16[c]), uint(c)) // special case for c=1
}

// Increment increments the counter
func (c *Count16) Increment() uint {
	t0 := n16[*c]
	t1 := n16[*c+1]

	// Check for overflow
	if *c >= math.MaxUint16 {
		return uint(t0)
	}

	// Increment the counter depending on the delta
	if delta := 1 / (t1 - t0); rand.Float64() < delta {
		(*c)++
	}

	return max(uint(t1), uint(*c))
}

// ------------------------------------ Count16x4 ------------------------------------

// Count16x4 is a represents 2 16-bit approximate counters, using atomic operations
// to increment the counter.
type Count16x4 struct {
	v atomic.Uint64
}

// estimate16x4 returns the estimated count for all counters.
func estimate16x4(v uint64) [4]uint {
	e := [4]uint16{
		uint16(v & 0xFFFF),
		uint16((v >> 16) & 0xFFFF),
		uint16((v >> 32) & 0xFFFF),
		uint16((v >> 48) & 0xFFFF),
	}

	return [4]uint{
		max(uint(n16[e[0]]), uint(e[0])),
		max(uint(n16[e[1]]), uint(e[1])),
		max(uint(n16[e[2]]), uint(e[2])),
		max(uint(n16[e[3]]), uint(e[3])),
	}
}

// Estimate returns the estimated count for all counters.
func (c *Count16x4) Estimate() [4]uint {
	return estimate16x4((*c).v.Load())
}

// EstimateAt returns the estimated count for the counter at the given index.
func (c *Count16x4) EstimateAt(i int) uint {
	if i < 0 || i > 3 {
		return 0
	}

	return c.Estimate()[i]
}

// IncrementAt increments the counter at the given index.
func (c *Count16x4) IncrementAt(i int) uint {
	if i < 0 || i > 3 {
		return 0
	}

	for {
		// Load the counter
		loaded := (*c).v.Load()
		counter := Count16(loaded >> uint(i*16))
		estimate := counter.Increment()

		// Pack the counter back
		updated := (uint64(counter) << uint(i*16)) | (loaded & ^(0xFFFF << uint(i*16)))

		// Try to swap the counters
		if (*c).v.CompareAndSwap(loaded, updated) {
			return estimate
		}
	}
}

// Reset resets the counter to zero. It returns the estimated count for all counters.
func (c *Count16x4) Reset() [4]uint {
	return estimate16x4((*c).v.Swap(0))
}
