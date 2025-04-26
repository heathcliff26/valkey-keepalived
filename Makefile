SHELL := bash

REPOSITORY ?= localhost
CONTAINER_NAME ?= valkey-keepalived
TAG ?= latest

# Build the binary
build:
	hack/build.sh

# Build the container image
image:
	podman build -t $(REPOSITORY)/$(CONTAINER_NAME):$(TAG) .

# Run unit tests
test:
	go test -v -coverprofile=coverprofile.out -coverpkg "./pkg/..." ./cmd/... ./pkg/...

# Run end-to-end tests
test-e2e:
	go test -count=1 -v ./tests/e2e/...

# Update project dependencies
update-deps:
	hack/update-deps.sh

# Generate a coverage profile
coverprofile:
	hack/coverprofile.sh

# Run the linter
lint: golangci-lint
	golangci-lint run -v

# Format the codebase
fmt:
	gofmt -s -w ./cmd ./pkg ./tests

# Validate generated files
validate:
	hack/validate.sh

# Scan code for vulnerabilities using gosec
gosec:
	gosec ./...

# Clean up build artifacts
clean:
	hack/clean.sh

# Install golangci-lint
golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show this help message
help:
	@echo "Available targets:"
	@echo ""
	@awk '/^#/{c=substr($$0,3);next}c&&/^[[:alpha:]][[:alnum:]_-]+:/{print substr($$1,1,index($$1,":")),c}1{c=0}' $(MAKEFILE_LIST) | column -s: -t
	@echo ""
	@echo "Run 'make <target>' to execute a specific target."

.PHONY: \
	build \
	image \
	test \
	test-e2e \
	update-deps \
	coverprofile \
	lint \
	fmt \
	validate \
	gosec \
	clean \
	golangci-lint \
	help \
	$(NULL)
