package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/parser"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/s3"
	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/sqs"
	"card-onboarding-workers/internal/observability"
)

const (
	preprocessorComponent = "card-onboarding-file-preprocessor"

	metricFileReceived      = "card_onboarding_file_preprocessor.file_received.count"
	metricFileRejected      = "card_onboarding_file_preprocessor.file_rejected.count"
	metricRecordsTotal      = "card_onboarding_file_preprocessor.records_total.count"
	metricRecordsAccepted   = "card_onboarding_file_preprocessor.records_accepted.count"
	metricRecordsRejected   = "card_onboarding_file_preprocessor.records_rejected.count"
	metricOutputFileUpload  = "card_onboarding_file_preprocessor.output_file_uploaded.count"
	metricSQSPublishSuccess = "card_onboarding_file_preprocessor.sqs_publish_success.count"
	metricSQSPublishFailed  = "card_onboarding_file_preprocessor.sqs_publish_failed.count"
	metricProcessingMs      = "card_onboarding_file_preprocessor.processing.duration_ms"
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
	startedAt := time.Now()
	fields := observability.Fields{
		Environment: os.Getenv("ENVIRONMENT_NAME"),
		Component:   preprocessorComponent,
	}
	defer func() {
		observability.LogMetric(
			observability.Metric{
				Name:  metricProcessingMs,
				Value: float64(time.Since(startedAt).Milliseconds()),
				Unit:  observability.UnitMilliseconds,
			},
			fields,
		)
	}()

	fileEvent, err := parser.ParseS3Event(rawSQSMessageBody)
	if err != nil {
		return fmt.Errorf("failed to parse s3 event: %w", err)
	}
	fields.SourceFile = fileEvent.SourceFileName
	logCountMetric(metricFileReceived, fields)

	fileBytes, err := service.s3Client.Download(ctx, fileEvent.BucketName, fileEvent.ObjectKey)
	if err != nil {
		return fmt.Errorf("failed to download source csv from s3: %w", err)
	}

	if err := parser.ValidateFile(fileEvent.SourceFileName, int64(len(fileBytes)), service.maxFileSizeBytes); err != nil {
		logCountMetric(metricFileRejected, fieldsWithError(fields, "file_validation_failed", err))
		return fmt.Errorf("invalid source file: %w", err)
	}

	records, rejectedResults, err := parser.ParseCSV(string(fileBytes))
	if err != nil {
		logCountMetric(metricFileRejected, fieldsWithError(fields, "csv_validation_failed", err))
		return fmt.Errorf("failed to parse source csv: %w", err)
	}

	jobID := service.generateID()
	fields.CorrelationID = jobID
	fields.JobID = jobID
	logValueMetric(metricRecordsTotal, len(records)+len(rejectedResults), fields)
	logValueMetric(metricRecordsAccepted, len(records), fields)
	logValueMetric(metricRecordsRejected, len(rejectedResults), fields)

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
	logCountMetric(metricOutputFileUpload, fields)

	for _, message := range messages {
		if err := service.sqsClient.Publish(ctx, message); err != nil {
			logCountMetric(metricSQSPublishFailed, fieldsWithError(fields, "sqs_publish_failed", err))
			return fmt.Errorf("failed to publish accepted record to sqs: %w", err)
		}
		logCountMetric(metricSQSPublishSuccess, fields)
	}

	return nil
}

func logCountMetric(name string, fields observability.Fields) {
	logValueMetric(name, 1, fields)
}

func logValueMetric(name string, value int, fields observability.Fields) {
	observability.LogMetric(
		observability.Metric{
			Name:  name,
			Value: float64(value),
			Unit:  observability.UnitCount,
		},
		fields,
	)
}

func fieldsWithError(fields observability.Fields, errorCode string, err error) observability.Fields {
	fields.ErrorCode = errorCode
	fields.ErrorMessage = err.Error()
	return fields
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
