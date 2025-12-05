.PHONY: test test-coverage bench build clean help oci-build oci-push

# Registry configuration (override with environment variables)
REGISTRY ?= docker.io
REPOSITORY ?= boifi/wasm-plugin
TAG ?= latest
OCI_IMAGE := $(REGISTRY)/$(REPOSITORY):$(TAG)

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

# OCI image build for Istio WasmPlugin deployment
oci-build: build
	@echo "Building OCI image for Wasm Plugin..."
	@echo "Image: $(OCI_IMAGE)"
	docker build -f Dockerfile.wasm -t $(OCI_IMAGE) .

# Push OCI image to registry
oci-push: oci-build
	@echo "Pushing OCI image to registry..."
	docker push $(OCI_IMAGE)

# Build and push in one step
push: oci-push

clean:
	cargo clean
	rm -rf coverage/

.PHONY: help
help:
	@echo "Wasm Plugin build targets:"
	@echo "  make test          - Run unit tests"
	@echo "  make test-coverage - Generate coverage report"
	@echo "  make test-all      - Run all tests"
	@echo "  make bench         - Run performance benchmarks"
	@echo "  make build         - Build WASM binary"
	@echo "  make oci-build     - Build OCI image for Istio deployment"
	@echo "  make oci-push      - Build and push OCI image to registry"
	@echo "  make push          - Alias for oci-push"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Configuration (environment variables):"
	@echo "  REGISTRY    - Container registry (default: docker.io)"
	@echo "  REPOSITORY  - Image repository (default: boifi/wasm-plugin)"
	@echo "  TAG         - Image tag (default: latest)"
