package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"card-onboarding-workers/card-onboarding-worker/internal/client"
	"card-onboarding-workers/card-onboarding-worker/internal/handler"
	"card-onboarding-workers/card-onboarding-worker/internal/service"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	onboardServiceBaseURLEnv = "ONBOARD_SERVICE_BASE_URL"
	onboardServiceTimeoutEnv = "ONBOARD_SERVICE_TIMEOUT"
	defaultOnboardTimeout    = 5 * time.Second
)

type config struct {
	OnboardServiceBaseURL string
	OnboardServiceTimeout time.Duration
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	onboardClient, err := client.NewOnboardClient(client.OnboardClientConfig{
		BaseURL: cfg.OnboardServiceBaseURL,
		Timeout: cfg.OnboardServiceTimeout,
	})
	if err != nil {
		log.Fatalf("create onboard client: %v", err)
	}

	workerService := service.NewWorkerService(onboardClient)
	handler := handler.NewHandler(workerService)

	lambda.Start(handler.Handle)
}

func loadConfig() (config, error) {
	baseURL := os.Getenv(onboardServiceBaseURLEnv)
	if baseURL == "" {
		return config{}, fmt.Errorf("%s is required", onboardServiceBaseURLEnv)
	}

	timeout := defaultOnboardTimeout
	if rawTimeout := os.Getenv(onboardServiceTimeoutEnv); rawTimeout != "" {
		parsedTimeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return config{}, fmt.Errorf("%s must be a valid duration: %w", onboardServiceTimeoutEnv, err)
		}
		timeout = parsedTimeout
	}

	return config{
		OnboardServiceBaseURL: baseURL,
		OnboardServiceTimeout: timeout,
	}, nil
}
