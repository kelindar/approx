package approx

import (
	"math"
	"sync/atomic"

	"github.com/kelindar/xxrand"
)

// n computes the approximate count based on Morris's algorithm
func n(v, a float64) float64 {
	return a * (math.Pow(1+1/a, v) - 1)
}

// ------------------------------------ Count8 ------------------------------------

// Scaling factor for the 8-bit counter
const scale8 = 31

// Count8 is a 8-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~255 with relatively mean error rate of
// around ~10%.
type Count8 uint8

// Estimate returns the estimated count
func (c Count8) Estimate() uint {
	return uint(n(float64(c), scale8))
}

// Increment increments the counter
func (c *Count8) Increment() {
	if *c == math.MaxUint8 {
		return // Overflow
	}

	count := float64(*c)
	delta := 1 / (n(count+1, scale8) - n(count, scale8))
	if xxrand.Float64() < delta {
		(*c)++
	}
}

// ------------------------------------ Count16 ------------------------------------

// Scaling factor for the 16-bit counter
const scale16 = 5000

// Count16 is a 16-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~2 billion with relatively low mean error rate of
// around ~0.50%.
type Count16 uint16

// Estimate returns the estimated count
func (c Count16) Estimate() uint {
	return uint(n(float64(c), scale16))
}

// Increment increments the counter
func (c *Count16) Increment() {
	if *c == math.MaxUint16 {
		return // Overflow
	}

	count := float64(*c)
	delta := 1 / (n(count+1, scale16) - n(count, scale16))
	if xxrand.Float64() < delta {
		(*c)++
	}
}

// ------------------------------------ Count16x4 ------------------------------------

// Count16x4 is a represents 2 16-bit approximate counters, using atomic operations
// to increment the counter.
type Count16x4 uint64

// Estimate returns the estimated count for all counters.
func (c *Count16x4) Estimate() [4]uint {
	v := atomic.LoadUint64((*uint64)(c))
	return [4]uint{
		uint(n(float64(v>>0), scale16)),
		uint(n(float64(v>>16), scale16)),
		uint(n(float64(v>>32), scale16)),
		uint(n(float64(v>>48), scale16)),
	}
}

// Increment increments the counter at the given index.
func (c *Count16x4) Increment(i int) {
	if i < 0 || i > 3 {
		return
	}

	for {
		// Load the counter
		loaded := atomic.LoadUint64((*uint64)(c))
		counter := Count16(loaded >> uint(i*16))
		counter.Increment()

		// Pack the counter back
		updated := (uint64(counter) << uint(i*16)) | (loaded & ^(0xFFFF << uint(i*16)))

		// Try to swap the counters
		if atomic.CompareAndSwapUint64((*uint64)(c), loaded, updated) {
			return
		}
	}
}
