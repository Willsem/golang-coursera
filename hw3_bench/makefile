test:
	go test -v

bench:
	go test -bench . -benchmem

profile:
	go test -bench . -benchmem -cpuprofile=cpu.out -memprofile=mem.out -memprofilerate=1
