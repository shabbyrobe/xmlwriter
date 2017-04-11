escapes:
	go build -gcflags="-m" 2>&1 | grep "escapes to heap"

coverhtml:
	go test -coverprofile=/tmp/cvg
	go tool cover -html=/tmp/cvg

bench:
	go test --run=^$$ --bench=. --benchmem
