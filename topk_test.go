package approx

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/kelindar/xxrand"
	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkTopK/k=5-24         	30690928	        39.04 ns/op	       0 B/op	       0 allocs/op
BenchmarkTopK/k=100-24       	30694933	        39.69 ns/op	       0 B/op	       0 allocs/op
BenchmarkTopK/k=1000-24      	29987254	        39.63 ns/op	       0 B/op	       0 allocs/op
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
				topk.Update(data[n%cardinality])
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
				topk.Update([]byte(v))
			}

			elements := topk.Elements()
			assert.Len(t, elements, int(k))

			x := 0
			for i := cardinality - k; i < cardinality; i++ {
				assert.Equal(t, strconv.Itoa(int(i)), string(elements[x].Value))
				assert.Equal(t, uint32(i), elements[x].Count)
				x++
			}
		})
	}
}

// Generate a random set of values
func deck(n int) [][]byte {
	values := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			values = append(values, []byte(strconv.Itoa(i)))
		}
	}

	// Randomly shuffle the values
	for i := range values {
		j := int(xxrand.Uint64n(uint64(n)))
		values[i], values[j] = values[j], values[i]
	}

	return values
}