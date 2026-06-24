ENV ?= dev
MAX_FILE_SIZE_BYTES ?= 10485760
ONBOARD_SERVICE_BASE_URL ?= http://localhost:8080
ONBOARD_SERVICE_TIMEOUT ?= 5s
DIST_DIR ?= dist
GO ?= go

.PHONY: help lint build test coverage clean install lambda-package package-lambdas cdk-synth cdk-synth-monitoring deploy-test deploy-prod deploy-monitoring smoke-test check-prod-config

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
	@echo "  cdk-synth-monitoring - Synthesize CloudWatch monitoring stack"
	@echo "  deploy-test     - Package Lambdas and deploy test stack"
	@echo "  deploy-prod     - Package Lambdas and deploy production stack"
	@echo "  deploy-monitoring - Deploy CloudWatch monitoring stack"

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
	$(GO) run ./tools/package-lambdas -dist "$(DIST_DIR)"

package-lambdas: lambda-package

cdk-synth: lambda-package
	cd infra && AWS_PAGER="" cdk synth -c env="$(ENV)" -c maxFileSizeBytes="$(MAX_FILE_SIZE_BYTES)" -c onboardServiceBaseUrl="$(ONBOARD_SERVICE_BASE_URL)" -c onboardServiceTimeout="$(ONBOARD_SERVICE_TIMEOUT)"

cdk-synth-monitoring:
	cd infra && AWS_PAGER="" cdk synth CardOnboardingMonitoringStack -c env="$(ENV)" -c stackGroup="monitoring"

deploy-test: lambda-package
	cd infra && AWS_PAGER="" cdk deploy CardOnboardingWorkersStack --require-approval never -c env="test" -c stackGroup="workers" -c maxFileSizeBytes="$(MAX_FILE_SIZE_BYTES)" -c onboardServiceBaseUrl="$(ONBOARD_SERVICE_BASE_URL)" -c onboardServiceTimeout="$(ONBOARD_SERVICE_TIMEOUT)"

check-prod-config:
	@test "$(ONBOARD_SERVICE_BASE_URL)" != "http://localhost:8080" || (echo "ONBOARD_SERVICE_BASE_URL must be set to the production service URL for deploy-prod" && exit 1)

deploy-prod: check-prod-config lambda-package
	cd infra && AWS_PAGER="" cdk deploy CardOnboardingWorkersStack --require-approval never -c env="prod" -c stackGroup="workers" -c maxFileSizeBytes="$(MAX_FILE_SIZE_BYTES)" -c onboardServiceBaseUrl="$(ONBOARD_SERVICE_BASE_URL)" -c onboardServiceTimeout="$(ONBOARD_SERVICE_TIMEOUT)"

deploy-monitoring:
	cd infra && AWS_PAGER="" cdk deploy CardOnboardingMonitoringStack --require-approval never -c env="$(ENV)" -c stackGroup="monitoring"

smoke-test:
	$(GO) test ./smoke-test -run TestSmokeFullCardOnboardingPlatform -v
