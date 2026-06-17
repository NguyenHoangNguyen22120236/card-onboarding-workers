ENV ?= dev
MAX_FILE_SIZE_BYTES ?= 10485760
ONBOARD_SERVICE_BASE_URL ?= http://localhost:8080
ONBOARD_SERVICE_TIMEOUT ?= 5s
DIST_DIR ?= dist

.PHONY: help build test clean install lambda-package cdk-synth

help:
	@echo "Available targets:"
	@echo "  build   - Build the project"
	@echo "  test    - Run tests"
	@echo "  clean   - Clean build artifacts"
	@echo "  install - Install dependencies"
	@echo "  cdk-synth - Package Lambdas and synthesize AWS CDK"

build:
	go build ./...

test:
	go test ./...

clean:
	go clean

install:
	go mod download

lambda-package:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/package-lambdas.ps1 -DistDir '$(DIST_DIR)'

cdk-synth: lambda-package
	powershell -NoProfile -ExecutionPolicy Bypass -Command "Set-Location infra; cdk synth -c env='$(ENV)' -c maxFileSizeBytes='$(MAX_FILE_SIZE_BYTES)' -c onboardServiceBaseUrl='$(ONBOARD_SERVICE_BASE_URL)' -c onboardServiceTimeout='$(ONBOARD_SERVICE_TIMEOUT)'"
