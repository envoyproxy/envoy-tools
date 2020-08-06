.PHONY: all # Install dependencies, build binary.
all: init test install

.PHONY: build # Build binary.
build:
	go build

.PHONY: test # Run tests.
test:
	go test ./...

.PHONY: help # Print help message.
help:
	@grep -E '^\.PHONY: [a-zA-Z_-]+ .*?# .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "(: |#)"}; {printf "%-30s %s\n", $$2, $$3}'

.PHONY: init # Install missing dependencies.
init:
	go get

.PHONY: install # Install built binary.
install:
	go install