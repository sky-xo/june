.PHONY: build install test clean

# Build the otto binary
build:
	go build -o otto ./cmd/otto

# Install to $GOPATH/bin
install:
	go install ./cmd/otto

# Run all tests
test:
	go test ./...

# Run tests with coverage
cover:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -f otto

# Build and run the TUI watch
watch:
	go build -o otto ./cmd/otto && ./otto watch
