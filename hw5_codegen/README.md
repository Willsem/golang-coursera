## Test

```
$ make test
```

### Result

```
go build handlers_gen/* && ./codegen api.go api_handlers.go
Completed
go test -v
=== RUN   TestMyApi
--- PASS: TestMyApi (0.01s)
=== RUN   TestOtherApi
--- PASS: TestOtherApi (0.00s)
PASS
ok  	github.com/Willsem/golang-coursera/hw5_codegen	0.017s
```
