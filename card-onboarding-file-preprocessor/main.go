package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/handler"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/s3"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/service"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/sqs"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	outputBucketNameEnv = "OUTPUT_BUCKET_NAME"
	maxFileSizeBytesEnv = "MAX_FILE_SIZE_BYTES"
	workerQueueURLEnv   = "WORKER_QUEUE_URL"
)

type config struct {
	OutputBucketName string
	MaxFileSizeBytes int64
	WorkerQueueURL   string
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	s3Client := s3.NewClient()
	sqsClient := sqs.NewClient(cfg.WorkerQueueURL)
	preprocessorService := service.NewPreprocessorService(
		s3Client,
		sqsClient,
		cfg.OutputBucketName,
		cfg.MaxFileSizeBytes,
	)
	handler := handler.New(preprocessorService)

	lambda.Start(handler.Handle)
}

func loadConfig() (config, error) {
	outputBucketName := os.Getenv(outputBucketNameEnv)
	if outputBucketName == "" {
		return config{}, fmt.Errorf("%s is required", outputBucketNameEnv)
	}

	rawMaxFileSizeBytes := os.Getenv(maxFileSizeBytesEnv)
	if rawMaxFileSizeBytes == "" {
		return config{}, fmt.Errorf("%s is required", maxFileSizeBytesEnv)
	}
	maxFileSizeBytes, err := strconv.ParseInt(rawMaxFileSizeBytes, 10, 64)
	if err != nil {
		return config{}, fmt.Errorf("%s must be a valid integer: %w", maxFileSizeBytesEnv, err)
	}
	if maxFileSizeBytes <= 0 {
		return config{}, fmt.Errorf("%s must be greater than 0", maxFileSizeBytesEnv)
	}

	workerQueueURL := os.Getenv(workerQueueURLEnv)
	if workerQueueURL == "" {
		return config{}, fmt.Errorf("%s is required", workerQueueURLEnv)
	}

	return config{
		OutputBucketName: outputBucketName,
		MaxFileSizeBytes: maxFileSizeBytes,
		WorkerQueueURL:   workerQueueURL,
	}, nil
}
