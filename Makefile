
.PHONY: test
test:
	go clean -testcache && go test -v -race -count=1 ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: lint
lint:
	golangci-lint run ./...
