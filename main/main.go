package main

import (
	"fmt"
	"math"

	"github.com/kelindar/approx"
)

func main() {
	const upper = 50

	var c approx.Count16
	meanerr := 0.0

	for i := 1; i <= upper; i++ {
		c.Increment()
		e := c.Estimate()

		err := math.Abs(float64(e)-float64(i)) / float64(i) * 100
		meanerr += err

		if i%1 == 0 {
			fmt.Printf("Actual: %v, Estimate: %v (#%d), Error: %.2f%%\n",
				i, c.Estimate(), c, err,
			)
		}
	}

	fmt.Printf("Mean error: %.2f%%\n", meanerr/upper)
}
