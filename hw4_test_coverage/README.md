## Test

```
$ make test
```

### Result

```
go test -v -cover
=== RUN   TestFindUsersOk
--- PASS: TestFindUsersOk (0.00s)
=== RUN   TestFindUsersErrorWithRequest
--- PASS: TestFindUsersErrorWithRequest (0.00s)
=== RUN   TestFindUsersUnauthorized
--- PASS: TestFindUsersUnauthorized (0.00s)
=== RUN   TestFindUsersUnmarshalError
--- PASS: TestFindUsersUnmarshalError (0.00s)
=== RUN   TestFindUsersNilServer
--- PASS: TestFindUsersNilServer (0.00s)
=== RUN   TestFindUsersServerTimeout
--- PASS: TestFindUsersServerTimeout (1.00s)
=== RUN   TestFindUsersInternalError
--- PASS: TestFindUsersInternalError (0.00s)
=== RUN   TestFindUsersBadRequestError
--- PASS: TestFindUsersBadRequestError (0.00s)
PASS
coverage: 100.0% of statements
ok  	github.com/Willsem/golang-coursera/hw4_test_coverage	1.021s
```
