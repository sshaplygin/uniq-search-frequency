
.PHONY: test
test:
	go clean -testcache && go test -v -race -count=1 ./...
