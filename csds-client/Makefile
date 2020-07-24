.PHONY: all # Install dependencies, build binary.
all: install build

.PHONY: build # Build binary.
build:
	go build -o bin/main main.go

.PHONY: help # Print help message.
help:
	@grep -E '^\.PHONY: [a-zA-Z_-]+ .*?# .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "(: |#)"}; {printf "%-30s %s\n", $$2, $$3}'

.PHONY: install # Install missing dependencies.
install:
	go get