package smoketest

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type smokeConfig struct {
	Enabled                 bool
	Region                  string
	InputBucket             string
	OutputBucket            string
	WorkerQueueURL          string
	WorkerDLQURL            string
	OnboardServiceBaseURL   string
	PollTimeout             time.Duration
	PollInterval            time.Duration
	QueueVisibilityTimeout  int32
	SourcePrefix            string
	RequireQueueObservation bool
	RequireDLQObservation   bool
	RequestStatusTableName  string
	AccountDetailsTableName string
}

func loadSmokeConfig(t *testing.T) smokeConfig {
	t.Helper()

	cfg := smokeConfig{
		Enabled:                 boolEnv("SMOKE_TEST_ENABLED", false),
		Region:                  envOrDefault("SMOKE_AWS_REGION", envOrDefault("AWS_REGION", "ap-southeast-1")),
		InputBucket:             strings.TrimSpace(os.Getenv("SMOKE_INPUT_BUCKET")),
		OutputBucket:            strings.TrimSpace(os.Getenv("SMOKE_OUTPUT_BUCKET")),
		WorkerQueueURL:          strings.TrimSpace(os.Getenv("SMOKE_WORKER_QUEUE_URL")),
		WorkerDLQURL:            strings.TrimSpace(os.Getenv("SMOKE_WORKER_DLQ_URL")),
		OnboardServiceBaseURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("SMOKE_ONBOARD_SERVICE_BASE_URL")), "/"),
		PollTimeout:             durationEnv("SMOKE_POLL_TIMEOUT", 3*time.Minute),
		PollInterval:            durationEnv("SMOKE_POLL_INTERVAL", 3*time.Second),
		QueueVisibilityTimeout:  int32Env("SMOKE_QUEUE_VISIBILITY_TIMEOUT_SECONDS", 5),
		SourcePrefix:            strings.Trim(strings.TrimSpace(envOrDefault("SMOKE_SOURCE_PREFIX", "smoke-test")), "/"),
		RequireQueueObservation: boolEnv("SMOKE_REQUIRE_QUEUE_OBSERVATION", true),
		RequireDLQObservation:   boolEnv("SMOKE_REQUIRE_DLQ_OBSERVATION", true),
		RequestStatusTableName:  strings.TrimSpace(os.Getenv("SMOKE_REQUEST_STATUS_TABLE")),
		AccountDetailsTableName: strings.TrimSpace(os.Getenv("SMOKE_ACCOUNT_DETAILS_TABLE")),
	}

	if !cfg.Enabled {
		t.Skip("set SMOKE_TEST_ENABLED=true to run full-platform smoke tests")
	}

	required := map[string]string{
		"SMOKE_INPUT_BUCKET":             cfg.InputBucket,
		"SMOKE_OUTPUT_BUCKET":            cfg.OutputBucket,
		"SMOKE_WORKER_QUEUE_URL":         cfg.WorkerQueueURL,
		"SMOKE_WORKER_DLQ_URL":           cfg.WorkerDLQURL,
		"SMOKE_ONBOARD_SERVICE_BASE_URL": cfg.OnboardServiceBaseURL,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("%s is required when SMOKE_TEST_ENABLED=true", key)
		}
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int32Env(key string, fallback int32) int32 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}
