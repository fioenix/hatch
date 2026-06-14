BINARY := hatch
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/fioenix/overclaud/hatch/internal/cli.Version=$(VERSION)

.PHONY: build install test lint vet fmt tidy clean

build: ## Build the hatch binary into ./bin
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/hatch

install: ## Install hatch into GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/hatch

test: ## Run all tests
	go test $(PKG)

vet: ## go vet
	go vet $(PKG)

fmt: ## gofmt -s -w
	gofmt -s -w $(shell find . -name '*.go' -not -path './vendor/*')

lint: vet ## Lint: vet + gofmt check
	@unformatted=$$(gofmt -s -l $$(find . -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

tidy: ## go mod tidy
	go mod tidy

clean:
	rm -rf bin
