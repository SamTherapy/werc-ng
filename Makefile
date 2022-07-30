GO ?= go
GOFLAGS ?= -ldflags "-s -w"

CGO_ENABLED ?= 0

all: go2werc

go2werc:
	$(GO) build $(GOFLAGS) .

fmt:
	gofmt -w -s .

vet:
	$(GO) vet ./...

lint: fmt vet
	-golangci-lint run

clean:
	$(GO) clean

.PHONY: clean lint test fmt vet help