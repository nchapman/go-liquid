.PHONY: lint lint-fix fmt test test-race bench check tidy

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	golangci-lint fmt ./...

test:
	go test ./...

test-race:
	go test -race ./...

bench:
	go test -bench=. -benchmem -run=^$$ ./...

tidy:
	go mod tidy

check: lint test
