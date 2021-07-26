test:
	go test -v -cover

cover:
	go test -coverprofile=$@.out
	go tool cover -html=$@.out -o $@.html
	open $@.html
	sleep 1
	rm cover.out $@.html
