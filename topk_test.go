// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package approx

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkTopK/k=5-24         	20144570	        57.62 ns/op	       0 B/op	       0 allocs/op
BenchmarkTopK/k=100-24       	19997566	        58.16 ns/op	       0 B/op	       0 allocs/op
BenchmarkTopK/k=1000-24      	20048851	        58.38 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkTopK(b *testing.B) {
	const cardinality = 10000
	data := deck(cardinality)

	for _, k := range []uint{5, 100, 1000} {
		topk, err := NewTopK(k)
		assert.NoError(b, err)

		b.Run(fmt.Sprintf("k=%d", k), func(b *testing.B) {
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				topk.UpdateString(data[n%cardinality])
			}
		})
	}
}

func TestTopK(t *testing.T) {
	const cardinality = 100
	for _, k := range []uint{2, 5, 10, 15} {
		k := k // capture
		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			k := uint(k)
			topk, err := NewTopK(k)
			assert.NoError(t, err)

			for _, v := range deck(cardinality) {
				topk.UpdateString(v)
			}

			elements := topk.Values()
			assert.Len(t, elements, int(k))

			fmt.Printf("-----------------\n")

			for _, e := range elements {
				fmt.Printf("v=%v, c=%v\n", string(e.Value), e.Count)
			}

			x := 0
			for i := cardinality - k; i < cardinality; i++ {
				//assert.Equal(t, strconv.Itoa(int(i)), string(elements[x].Value))
				assert.InDelta(t, uint32(i), elements[x].Count, 10)
				x++
			}
		})
	}
}

func TestTopK_Simple(t *testing.T) {
	topk, err := NewTopK(5)
	assert.NoError(t, err)

	// Add 10 elements to the topk
	for _, v := range deck(10) {
		topk.UpdateString(v)
	}

	elements := topk.Values()
	assert.Len(t, elements, 5)

	// The top 5 elements should be 5, 6, 7, 8, 9
	for i, e := range elements {
		assert.Equal(t, strconv.Itoa(5+i), string(e.Value))
		assert.Equal(t, uint32(5+i), e.Count)
	}
}

// Generate a random set of values
func deck(n int) []string {
	values := make([]string, 0, n)
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			values = append(values, strconv.Itoa(i))
		}
	}

	// Randomly shuffle the values
	for i := range values {
		j := int(rand.Int63n(int64(n)))
		values[i], values[j] = values[j], values[i]
	}

	return values
}
