APP_NAME := slacko
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: build install test lint fmt vet clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

clean:
	rm -f $(APP_NAME)
	rm -rf dist/
