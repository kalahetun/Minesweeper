.PHONY: test test-coverage build clean help

test:
	@echo "Running CLI unit tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage report..."
	go test -v ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

build:
	@echo "Building CLI..."
	go build -o bin/hfi-cli ./main.go

run: build
	@echo "Running CLI..."
	./bin/hfi-cli

clean:
	rm -f coverage.out coverage.html
	rm -rf bin/
	go clean -testcache

.PHONY: help
help:
	@echo "CLI test targets:"
	@echo "  make test          - Run unit tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make build         - Build CLI binary"
	@echo "  make run           - Build and run CLI"
	@echo "  make clean         - Clean build artifacts"
