.PHONY: test test-coverage bench build clean help

test:
	@echo "Running Wasm Plugin unit tests..."
	cargo test --lib

test-coverage:
	@echo "Running tests with coverage..."
	cargo tarpaulin --out Html --output-dir coverage

test-all:
	@echo "Running all tests (unit + integration + e2e)..."
	cargo test

bench:
	@echo "Running benchmarks..."
	cargo bench --bench '*'

build:
	@echo "Building Wasm Plugin..."
	cargo build --target wasm32-unknown-unknown --release

clean:
	cargo clean
	rm -rf coverage/

.PHONY: help
help:
	@echo "Wasm Plugin test targets:"
	@echo "  make test          - Run unit tests"
	@echo "  make test-coverage - Generate coverage report"
	@echo "  make test-all      - Run all tests"
	@echo "  make bench         - Run performance benchmarks"
	@echo "  make build         - Build WASM binary"
	@echo "  make clean         - Clean build artifacts"
