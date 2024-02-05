[![Go](https://github.com/romulets/test-cloudwatch-expressions-comparisson/actions/workflows/go.yml/badge.svg)](https://github.com/romulets/test-cloudwatch-expressions-comparisson/actions/workflows/go.yml)

# test-cloudwatch-expressions-comparisson

Compares cloudwatch expressions

## Benchmark tests

> `go test -bench . -benchmem`

```
goos: darwin
goarch: arm64
pkg: github.com/romulets/test-cloudwatch-expressions-comparisson
BenchmarkAreCloudWatchExpressionsEquivalent-12    	   24568	     44908 ns/op	   30224 B/op	     242 allocs/op
PASS
ok  	github.com/romulets/test-cloudwatch-expressions-comparisson	1.789s
```