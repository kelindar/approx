package approx

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkCount/c8-24 			305973754	        3.941 ns/op	   		0 B/op	       0 allocs/op
BenchmarkCount/c16-24         	256379030	        4.725 ns/op	   		0 B/op	       0 allocs/op
BenchmarkCount/c16x4-24       	81941466	        14.74 ns/op			0 B/op	       0 allocs/op
*/
func BenchmarkCount(b *testing.B) {
	b.Run("c8", func(b *testing.B) {
		var c Count8
		for i := 0; i < b.N; i++ {
			c.Increment()
			if c == 255 {
				c = 0
			}
		}
	})

	b.Run("c16", func(b *testing.B) {
		var c Count16
		for i := 0; i < b.N; i++ {
			c.Increment()
		}
	})

	b.Run("c16x4", func(b *testing.B) {
		var c Count16x4
		for i := 0; i < b.N; i++ {
			c.IncrementAt(1)
		}
	})
}

func TestCount8_MeanError(t *testing.T) {
	const upper = 1e4
	var c Count8

	meanerr := 0.0
	for i := 1; i <= int(upper); i++ {
		c.Increment()
		e := c.Estimate()
		err := math.Abs(float64(e)-float64(i)) / float64(i) * 100
		meanerr += err / upper
	}
	assert.Less(t, meanerr, 25.0, "mean error is %.2f%%", meanerr)
}

func TestCount16_MeanError(t *testing.T) {
	const upper = 1e5
	var c Count16

	meanerr := 0.0
	for i := 1; i <= int(upper); i++ {
		c.Increment()
		e := c.Estimate()
		err := math.Abs(float64(e)-float64(i)) / float64(i) * 100
		meanerr += err / upper
	}
	assert.Less(t, meanerr, 2.0, "mean error is %.2f%%", meanerr)
}

func TestCount16x4_MeanErrort(t *testing.T) {
	const upper = 1e5
	var c Count16x4

	meanerr := 0.0
	for i := 1; i <= int(upper); i++ {
		c.IncrementAt(1)
		e := c.EstimateAt(1)
		err := math.Abs(float64(e)-float64(i)) / float64(i) * 100
		meanerr += err / upper
	}
	assert.Less(t, meanerr, 1.5, "mean error is %.2f%%", meanerr)
}

func TestCount8_Overflow(t *testing.T) {
	var c Count8

	assert.NotPanics(t, func() {
		for i := 0; i < 1e5; i++ {
			c.Increment()
			c.Estimate()
		}
	})
}

func main() {
	const upper = 2e9 // 2 billion

	var c Count16
	meanerr := 0.0

	for i := 1; i <= upper; i++ {
		c.Increment()
		e := c.Estimate()

		err := math.Abs(float64(e)-float64(i)) / float64(i) * 100
		meanerr += err

		if i%1e7 == 0 {
			fmt.Printf("Actual: %v, Estimate: %v (#%d), Error: %.2f%%\n",
				i, c.Estimate(), c, err,
			)
		}
	}

	fmt.Printf("Mean error: %.2f%%\n", meanerr/upper)
}
