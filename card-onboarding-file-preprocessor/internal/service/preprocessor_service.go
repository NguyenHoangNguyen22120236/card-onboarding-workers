package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/parser"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/s3"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/sqs"
)

type IDGenerator func() string

type PreprocessorService struct {
	s3Client         s3.S3Client
	sqsClient        sqs.SQSClient
	outputBucket     string
	maxFileSizeBytes int64
	generateID       IDGenerator
}

func NewPreprocessorService(
	s3Client s3.S3Client,
	sqsClient sqs.SQSClient,
	outputBucket string,
	maxFileSizeBytes int64,
) *PreprocessorService {
	return NewPreprocessorServiceWithIDGenerator(
		s3Client,
		sqsClient,
		outputBucket,
		maxFileSizeBytes,
		func() string { return uuid.NewString() },
	)
}

func NewPreprocessorServiceWithIDGenerator(
	s3Client s3.S3Client,
	sqsClient sqs.SQSClient,
	outputBucket string,
	maxFileSizeBytes int64,
	generateID IDGenerator,
) *PreprocessorService {
	return &PreprocessorService{
		s3Client:         s3Client,
		sqsClient:        sqsClient,
		outputBucket:     outputBucket,
		maxFileSizeBytes: maxFileSizeBytes,
		generateID:       generateID,
	}
}

func (service *PreprocessorService) Process(ctx context.Context, rawSQSMessageBody string) error {
	fileEvent, err := parser.ParseS3Event(rawSQSMessageBody)
	if err != nil {
		return fmt.Errorf("failed to parse s3 event: %w", err)
	}

	fileBytes, err := service.s3Client.Download(ctx, fileEvent.BucketName, fileEvent.ObjectKey)
	if err != nil {
		return fmt.Errorf("failed to download source csv from s3: %w", err)
	}

	if err := parser.ValidateFile(fileEvent.SourceFileName, int64(len(fileBytes)), service.maxFileSizeBytes); err != nil {
		return fmt.Errorf("invalid source file: %w", err)
	}

	records, rejectedResults, err := parser.ParseCSV(string(fileBytes))
	if err != nil {
		return fmt.Errorf("failed to parse source csv: %w", err)
	}

	jobID := service.generateID()
	messages := parser.MapCSVRecordsToOnboardingMessages(records, jobID, jobID, fileEvent.SourceFileName)
	results := appendAcceptedResults(messages, rejectedResults)

	resultCSV, err := GeneratePreprocessResultCSV(results)
	if err != nil {
		return fmt.Errorf("failed to generate preprocess result csv: %w", err)
	}

	outputKey := fmt.Sprintf("processed/%s/%s_preprocess_result.csv", jobID, fileEvent.SourceFileName)
	if err := service.s3Client.Upload(ctx, service.outputBucket, outputKey, []byte(resultCSV)); err != nil {
		return fmt.Errorf("failed to upload preprocess result csv to s3: %w", err)
	}

	for _, message := range messages {
		if err := service.sqsClient.Publish(ctx, message); err != nil {
			return fmt.Errorf("failed to publish accepted record to sqs: %w", err)
		}
	}

	return nil
}

func appendAcceptedResults(messages []entity.OnboardingMessage, rejectedResults []entity.PreprocessResult) []entity.PreprocessResult {
	results := make([]entity.PreprocessResult, 0, len(messages)+len(rejectedResults))
	for _, message := range messages {
		results = append(results, entity.PreprocessResult{
			RecordID:         message.RecordID,
			RowNumber:        message.RowNumber,
			CustomerID:       message.CustomerID,
			PreprocessStatus: entity.PreprocessStatusAccepted,
		})
	}

	results = append(results, rejectedResults...)
	return results
}
