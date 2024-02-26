// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package approx

import (
	"math"
	"sync/atomic"

	_ "unsafe" // For go:linkname
)

// n computes the approximate count based on Morris's algorithm
func n(v, a float64) float64 {
	return a * (math.Pow(1+1/a, v) - 1)
}

//go:linkname runtime_rand runtime.rand
func runtime_rand() uint64

// fastrand is a fast thread local random function built into the Go runtime
// but not normally exposed.  On Linux x86_64, this is aesrand seeded by
// /dev/urandom.
//
//go:linkname fastrand runtime.fastrand
func fastrand() uint32

// roll32 returns a random float32 in the range [0, 1)
func roll32() float32 {
	return float32(uint32(runtime_rand())<<8>>8) / (1 << 24)

}

// ------------------------------------ Count8 ------------------------------------

const (
	scale8    = 31                // scale factor
	upper8    = math.MaxUint8 + 1 // upper bound
	MaxCount8 = 101681            // n(math.MaxUint8, 31)
)

// Precompute the lookup table for the 8-bit counter
var n8 [upper8]uint = func() [upper8]uint {
	var lookup [upper8]uint
	for i := range lookup {
		lookup[i] = uint(n(float64(i), scale8))
	}
	lookup[1] = 1 // special case for c=1
	return lookup
}()

// Precompute the delta table for the 8-bit counter
var d8 [upper8]float32 = func() [upper8]float32 {
	var lookup [upper8]float32
	for i := 0; i < len(lookup)-1; i++ {
		lookup[i] = float32(1 / (n(float64(i+1), scale8) - n(float64(i), scale8)))
	}
	lookup[math.MaxUint8] = 0 // no chance to increment
	return lookup
}()

// Count8 is a 8-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~100k with relatively mean error rate of
// around ~10%.
type Count8 uint8

// Estimate returns the estimated count
func (c Count8) Estimate() uint {
	return n8[c]
}

// Increment increments the counter
func (c *Count8) Increment() uint {
	if roll32() < d8[*c] {
		(*c)++
	}
	return n8[*c]
}

// ------------------------------------ Count16 ------------------------------------

const (
	scale16    = 5000               // scale factor
	upper16    = math.MaxUint16 + 1 // upper bound
	MaxCount16 = 2458655843         // n(math.MaxUint16, 5000)
)

// Precompute the lookup table for the 16-bit counter
var n16 [upper16]uint = func() [upper16]uint {
	var lookup [upper16]uint
	for i := range lookup {
		lookup[i] = uint(n(float64(i), scale16))
	}
	lookup[1] = 1 // special case for c=1
	return lookup
}()

// Precompute the delta table for the 16-bit counter
var d16 [upper16]float32 = func() [upper16]float32 {
	var lookup [upper16]float32
	for i := 0; i < len(lookup)-1; i++ {
		lookup[i] = float32(1 / (n(float64(i+1), scale16) - n(float64(i), scale16)))
	}
	return lookup
}()

// Count16 is a 16-bit counter that uses Morris's algorithm to estimate the count. The
// counter was tuned to count up to ~2 billion with relatively low mean error rate of
// around ~0.50%.
type Count16 uint16

// Estimate returns the estimated count
func (c Count16) Estimate() uint {
	return n16[c]
}

// Increment increments the counter
func (c *Count16) Increment() uint {
	if roll32() < d16[*c] {
		(*c)++
	}
	return n16[*c]
}

// ------------------------------------ Count16x4 ------------------------------------

// Count16x4 is a represents 4 16-bit approximate counters, using atomic operations
// to increment the counter.
type Count16x4 struct {
	v atomic.Uint64
}

// estimate16x4 returns the estimated count for all counters.
func estimate16x4(v uint64) [4]uint {
	return [4]uint{
		n16[uint16(v&0xFFFF)],
		n16[uint16((v>>16)&0xFFFF)],
		n16[uint16((v>>32)&0xFFFF)],
		n16[uint16((v>>48)&0xFFFF)],
	}
}

// Estimate returns the estimated count for all counters.
func (c *Count16x4) Estimate() [4]uint {
	return estimate16x4(c.v.Load())
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

	return c.incrementAt(i, roll32())
}

// IncrementAt increments the counter at the given index with a given probability of success.
func (c *Count16x4) incrementAt(i int, roll float32) uint {
	shft := uint(i * 16) // number of bits to shift
	for {
		loaded := c.v.Load()

		// Inlined version of Count16.Increment. Early return allows us to avoid the
		// cost of the atomic operation if we don't need to increment the counter.
		counter := uint16(loaded >> shft)
		if roll >= d16[counter] {
			return n16[counter]
		}

		// Increment the counter and pack it back
		counter++
		updated := (uint64(counter) << shft) | (loaded & ^(0xFFFF << shft))

		// Now try to swap the value atomically.
		if c.v.CompareAndSwap(loaded, updated) {
			return n16[counter]
		}
	}
}

// Reset resets the counter to zero. It returns the estimated count for all counters.
func (c *Count16x4) Reset() [4]uint {
	return estimate16x4((*c).v.Swap(0))
}
