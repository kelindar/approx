<p align="center">
<img width="330" height="110" src=".github/logo.png" border="0" alt="kelindar/approx">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/approx" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/approx"><img src="https://pkg.go.dev/badge/github.com/kelindar/approx" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/approx"><img src="https://goreportcard.com/badge/github.com/kelindar/approx" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/approx"><img src="https://coveralls.io/repos/github/kelindar/approx/badge.svg" alt="Coverage"></a>
</p>

# Probabilistic Data Structures

This Go package provides several data structures for approximate counting and frequency estimation, leveraging Morris's algorithm and other techniques for efficient, probabilistic computations. It is suitable for applications where exact counts are unnecessary or where memory efficiency is paramount. The package includes implementations for 8-bit and 16-bit counters, Count-Min Sketch, and a Top-K frequent elements tracker.

- **8-bit and 16-bit Approximate Counters**: Implements Morris's algorithm to provide approximate counts with low memory usage. Tuned to count up to ~100k (8-bit) and ~2 billion (16-bit) with acceptable error rates.
- **Count-Min Sketch**: A probabilistic data structure for estimating the frequency of events in a stream of data. It allows users to specify desired accuracy and confidence levels.
- **Top-K Frequent Elements Tracking**: Maintains a list of the top-K frequent elements in a stream, using a combination of Count-Min Sketch for frequency estimation and a min-heap for maintaining the top elements.

## Advantages

- **Memory Efficiency**: The package uses probabilistic data structures which offer significant memory savings compared to exact counting methods, especially beneficial for large datasets or streams.
- **Performance**: Incorporates fast, thread-local random number generation and efficient hash functions, optimizing performance for high-throughput applications.
- **Scalability**: Suitable for scaling to large datasets with minimal computational and memory footprint increases.
- **Thread-Safety**: Features such as atomic operations ensure thread safety, making the package suitable for concurrent applications.

## Disadvantages

- **Probabilistic Nature**: As the package relies on probabilistic algorithms, there is an inherent trade-off between memory usage and accuracy. Exact counts are not guaranteed, and there is a specified error margin.

## Usage

### 8-bit and 16-bit Counters

Instantiate a counter and use the `Increment` method to increase its value probabilistically. The `Estimate` method returns the current approximate count.

```go
var counter approx.Count8 // or approx.Count16 for 16-bit
counter.Increment()
fmt.Println(counter.Estimate())
```

### Count-Min Sketch

Create a new Count-Min Sketch with default or custom parameters. Update the sketch with observed items and query their estimated frequencies.

```go
cms, err := approx.NewCountMin()
if err != nil {
    log.Fatal(err)
}
cms.UpdateString("example_item")
fmt.Println(cms.CountString("example_item"))
```

### Top-K Frequent Elements

Track the top-K frequent elements in a stream by creating a `TopK` instance and updating it with observed items.

```go
topK, err := approx.NewTopK(10)
if err != nil {
    log.Fatal(err)
}
topK.UpdateString("example_item")
for _, v := range topK.Values() {
    fmt.Printf("Value: %s, Count: %d\n", v.Value, v.Count)
}
```

## Dependencies

This package depends on external libraries for hashing and HyperLogLog implementations:

- `github.com/zeebo/xxh3`: For fast hashing.
- `github.com/axiomhq/hyperloglog`: For cardinality estimation in the Top-K tracker.

## License

Please review the license agreement for this package before using it in your projects.
