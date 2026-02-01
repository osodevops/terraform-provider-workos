# Copyright (c) OSO DevOps
# SPDX-License-Identifier: MPL-2.0

default: build

# Build the provider
build:
	go build -o terraform-provider-workos

# Install the provider locally for development
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/osodevops/workos/0.0.1/darwin_arm64
	mv terraform-provider-workos ~/.terraform.d/plugins/registry.terraform.io/osodevops/workos/0.0.1/darwin_arm64/

# Run unit tests
test:
	go test -v -cover -timeout=120s -parallel=4 ./...

# Run acceptance tests (requires WORKOS_API_KEY)
testacc:
	TF_ACC=1 go test -v -cover -timeout=20m -parallel=4 ./...

# Run a specific acceptance test
testacc-one:
	TF_ACC=1 go test -v -timeout=20m -run=$(TEST) ./internal/provider

# Format code
fmt:
	go fmt ./...
	terraform fmt -recursive ./examples/

# Run linter
lint:
	golangci-lint run ./...

# Generate documentation
docs:
	go generate ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f terraform-provider-workos
	rm -rf dist/

# Run all checks (format, lint, test)
check: fmt lint test

# Verify the provider can be built and documentation generated
verify: tidy build docs
	@echo "Verification complete"

.PHONY: default build install test testacc testacc-one fmt lint docs tidy clean check verify
