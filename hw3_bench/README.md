## Test

```
$ make bench
```

### Result

```
go test -bench . -benchmem
goos: darwin
goarch: amd64
pkg: github.com/Willsem/golang-coursera/hw3_bench
cpu: Intel(R) Core(TM) i5-5257U CPU @ 2.70GHz
BenchmarkSlow-4   	      37	  33862256 ns/op	19483564 B/op	  189797 allocs/op
BenchmarkFast-4   	     409	   2648243 ns/op	  536116 B/op	    6760 allocs/op
PASS
ok  	github.com/Willsem/golang-coursera/hw3_bench	2.750s
```
