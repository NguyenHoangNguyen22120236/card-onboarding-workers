ENV ?= dev
MAX_FILE_SIZE_BYTES ?= 10485760
ONBOARD_SERVICE_BASE_URL ?= http://localhost:8080
ONBOARD_SERVICE_TIMEOUT ?= 5s
DIST_DIR ?= dist
GO ?= go

.PHONY: help lint build test coverage clean install lambda-package package-lambdas cdk-synth deploy-test smoke-test

help:
	@echo "Available targets:"
	@echo "  lint            - Run Go formatting check and vet"
	@echo "  build           - Build the project"
	@echo "  test            - Run tests"
	@echo "  coverage        - Run tests with coverage output"
	@echo "  clean           - Clean build artifacts"
	@echo "  install         - Install dependencies"
	@echo "  lambda-package  - Package Lambda functions"
	@echo "  cdk-synth       - Package Lambdas and synthesize AWS CDK"

lint:
	@test -z "$$(gofmt -l .)"
	$(GO) vet ./...

build:
	$(GO) build ./...

test:
	$(GO) test ./...

coverage:
	$(GO) test ./... -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -func=coverage.out

clean:
	$(GO) clean
	rm -rf $(DIST_DIR) coverage.out

install:
	$(GO) mod download

lambda-package:
	$(GO) run ./tools/package-lambdas -dist '$(DIST_DIR)'

package-lambdas: lambda-package

cdk-synth: lambda-package
	cd infra && cdk synth -c env='$(ENV)' -c maxFileSizeBytes='$(MAX_FILE_SIZE_BYTES)' -c onboardServiceBaseUrl='$(ONBOARD_SERVICE_BASE_URL)' -c onboardServiceTimeout='$(ONBOARD_SERVICE_TIMEOUT)'

deploy-test: lambda-package
	cd infra && cdk deploy --require-approval never -c env='test' -c maxFileSizeBytes='$(MAX_FILE_SIZE_BYTES)' -c onboardServiceBaseUrl='$(ONBOARD_SERVICE_BASE_URL)' -c onboardServiceTimeout='$(ONBOARD_SERVICE_TIMEOUT)'

smoke-test:
	$(GO) test ./... -run 'TestLocalE2ESimulation'
