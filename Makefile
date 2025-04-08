.PHONY: build
build:
	go build -o dist/brsp ./cmd/brsp

.PHONY: install
install:
	go install github.com/takaishi/brsp/cmd/brsp

.PHONY: test
test:
	go test -race ./...
