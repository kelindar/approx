// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package approx

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkCMS/update-24         	45178000	        25.74 ns/op	       0 B/op	       0 allocs/op
BenchmarkCMS/count-24          	88864532	        13.59 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkCMS(b *testing.B) {
	b.Run("update", func(b *testing.B) {
		c, _ := NewCountMin()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.UpdateString("foo")
		}
	})

	b.Run("count", func(b *testing.B) {
		c, _ := NewCountMin()
		c.UpdateString("foo")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.CountString("foo")
		}
	})
}

func TestCounter_HighCardinality(t *testing.T) {
	const n = 1e6
	const delta = n * defaultEpsilon

	c, err := NewCountMin()
	assert.NoError(t, err)

	for i := 0; i < n; i++ {
		c.UpdateString(strconv.Itoa(i))
	}

	var hits float64
	for i := 0; i < n; i++ {
		if c.CountString(strconv.Itoa(i)) <= delta {
			hits++
		}
	}

	maxError := (1 - defaultConfidence) * 100
	errorRate := (1 - (hits / n)) * 100
	assert.Less(t, errorRate, maxError, "error is %.2f%%", errorRate)
}

func TestCounter_Simple(t *testing.T) {
	c, err := NewCountMin()
	assert.NoError(t, err)

	c.UpdateString("foo")
	c.UpdateString("foo")
	c.UpdateString("bar")

	assert.Equal(t, uint(2), c.CountString("foo"))
	assert.Equal(t, uint(1), c.CountString("bar"))
}

func TestCounter_Binary(t *testing.T) {
	c, err := NewCountMin()
	assert.NoError(t, err)

	c.Update([]byte("foo"))
	c.Update([]byte("foo"))
	c.Update([]byte("bar"))

	assert.Equal(t, uint(2), c.Count([]byte("foo")))
	assert.Equal(t, uint(1), c.Count([]byte("bar")))

}

func TestCounter_Overflow(t *testing.T) {
	c, err := NewCountMin()
	assert.NoError(t, err)

	for i := 0; i < 1000; i++ {
		c.UpdateString("foo")
	}

	assert.InDelta(t, uint(1000), c.CountString("foo"), 100)
}

func TestCounter_Validation(t *testing.T) {
	_, err := NewCountMinWithEstimates(0, 0)
	assert.Error(t, err)

	_, err = NewCountMinWithEstimates(.1, 1)
	assert.Error(t, err)

	_, err = NewCountMinWithSize(129, 1)
	assert.Error(t, err)

	_, err = NewCountMinWithSize(1, 1<<31)
	assert.Error(t, err)
}

func TestCounterParallel(t *testing.T) {
	c, err := NewCountMin()
	assert.NoError(t, err)

	const parallelism = 32

	var wg sync.WaitGroup
	wg.Add(parallelism)
	for g := 0; g < parallelism; g++ {
		go func() {
			for i := 0; i < 1000; i++ {
				c.UpdateString("foo")
				c.CountString("foo")
			}

			c.Reset()
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestCountMin_Size(t *testing.T) {
	c, err := NewCountMin()
	assert.NoError(t, err)
	assert.Equal(t, 256, len(c.counts[0]))
}
