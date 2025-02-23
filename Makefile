SHELL := bash

REPOSITORY ?= localhost
CONTAINER_NAME ?= valkey-keepalived
TAG ?= latest

build:
	hack/build.sh

image:
	podman build -t $(REPOSITORY)/$(CONTAINER_NAME):$(TAG) .

test:
	go test -v -coverprofile=coverprofile.out -coverpkg "./pkg/..." ./cmd/... ./pkg/...

test-e2e:
	go test -count=1 -v ./tests/e2e/...

update-deps:
	hack/update-deps.sh

coverprofile:
	hack/coverprofile.sh

lint: golangci-lint
	golangci-lint run -v

fmt:
	gofmt -s -w ./cmd ./pkg ./tests

validate:
	hack/validate.sh

clean:
	hack/clean.sh

golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

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
	clean \
	golangci-lint \
	$(NULL)
