package approx

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.Less(t, meanerr, 1.5, "mean error is %.2f%%", meanerr)
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
