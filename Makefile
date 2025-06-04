EXPECTED_GO_VERSION := $(shell awk '/^go / {print $$2}' go.mod)
CURRENT_GO_VERSION := $(shell go version | cut -d' ' -f3 | sed 's/go//')


.PHONY: build
build: check-go-version
	go build -o dist/brsp ./cmd/brsp

.PHONY: install
install: check-go-version
	go install github.com/takaishi/brsp/cmd/brsp

.PHONY: test
test: check-go-version
	go test -race ./...

.PHONY: check-go-version
check-go-version:
	@echo EXPECTED_GO_VERSION: $(EXPECTED_GO_VERSION)
	@echo CURRENT_GO_VERSION: $(CURRENT_GO_VERSION)
	@if [ "$(EXPECTED_GO_VERSION)" != "$(CURRENT_GO_VERSION)" ]; then \
		echo "Warning: go.mod version does not match Makefile GO_VERSION ($(EXPECTED_GO_VERSION))"; \
		exit 1; \
	fi
