GO ?= go
GOLANGCI_LINT ?= golangci-lint
STATICCHECK ?= staticcheck

.PHONY: build run test coverage generate format format-check tidy verify lint vet staticcheck check info clean

build:
	$(GO) build ./...

run:
	$(GO) run ./ $(ARGS)

test:
	$(GO) clean -testcache
	$(GO) test -v -race -count=1 ./...

coverage:
	$(GO) test -cover -count=1 ./...

generate:
	$(GO) generate ./...

format:
	$(GO) fmt ./...

format-check:
	$(GO) fmt ./...
	git diff --exit-code -- '*.go'

tidy:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

verify:
	$(GO) mod verify

lint:
	$(GOLANGCI_LINT) run ./...

vet:
	$(GO) vet ./...

staticcheck:
	$(STATICCHECK) ./...

check: format-check tidy verify lint vet staticcheck test

info:
	$(GO) version

clean:
	$(GO) clean
