.PHONY: help build test clean install

help:
	@echo "Available targets:"
	@echo "  build   - Build the project"
	@echo "  test    - Run tests"
	@echo "  clean   - Clean build artifacts"
	@echo "  install - Install dependencies"

build:
	go build ./...

test:
	go test ./...

clean:
	go clean

install:
	go mod download
