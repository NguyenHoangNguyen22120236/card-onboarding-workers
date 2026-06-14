package service

import (
	"context"
	"encoding/csv"
	"errors"
	"reflect"
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func TestPreprocessorService_ValidCSVPublishesAcceptedRecords(t *testing.T) {
	s3Client := &fakeS3Client{
		downloadBody: []byte(validCSV(
			"customer-1,VISA,4111111111111111,12/29,Jane Doe,jane@example.com",
			"customer-2,MASTERCARD,5555555555554444,10/30,John Doe,john@example.com",
		)),
	}
	sqsClient := &fakeSQSClient{}
	service := newTestPreprocessorService(s3Client, sqsClient)

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if s3Client.downloadBucket != "input-bucket" {
		t.Errorf("downloadBucket = %q, want %q", s3Client.downloadBucket, "input-bucket")
	}
	if s3Client.downloadKey != "input/cards.csv" {
		t.Errorf("downloadKey = %q, want %q", s3Client.downloadKey, "input/cards.csv")
	}
	if s3Client.uploadBucket != "output-bucket" {
		t.Errorf("uploadBucket = %q, want %q", s3Client.uploadBucket, "output-bucket")
	}
	if s3Client.uploadKey != "processed/job-123/cards.csv_preprocess_result.csv" {
		t.Errorf("uploadKey = %q, want %q", s3Client.uploadKey, "processed/job-123/cards.csv_preprocess_result.csv")
	}

	wantMessages := []entity.OnboardingMessage{
		{
			CorrelationID: "job-123",
			JobID:         "job-123",
			RecordID:      "REC-001",
			SourceFile:    "cards.csv",
			RowNumber:     2,
			CustomerID:    "customer-1",
			CardType:      "VISA",
			CardNumber:    "4111111111111111",
			ExpiryDate:    "12/29",
			HolderName:    "Jane Doe",
			Email:         "jane@example.com",
		},
		{
			CorrelationID: "job-123",
			JobID:         "job-123",
			RecordID:      "REC-002",
			SourceFile:    "cards.csv",
			RowNumber:     3,
			CustomerID:    "customer-2",
			CardType:      "MASTERCARD",
			CardNumber:    "5555555555554444",
			ExpiryDate:    "10/30",
			HolderName:    "John Doe",
			Email:         "john@example.com",
		},
	}
	if !reflect.DeepEqual(sqsClient.publishedMessages, wantMessages) {
		t.Fatalf("publishedMessages = %#v, want %#v", sqsClient.publishedMessages, wantMessages)
	}

	rows := uploadedResultRows(t, s3Client)
	assertResultRow(t, rows[1], []string{"REC-001", "2", "customer-1", entity.PreprocessStatusAccepted, ""})
	assertResultRow(t, rows[2], []string{"REC-002", "3", "customer-2", entity.PreprocessStatusAccepted, ""})
}

func TestPreprocessorService_MalformedRowsAreRejectedInResultFile(t *testing.T) {
	s3Client := &fakeS3Client{
		downloadBody: []byte(validCSV(
			"customer-1,VISA,4111111111111111,12/29,Jane Doe,jane@example.com",
			"customer-2,MASTERCARD,5555555555554444",
		)),
	}
	sqsClient := &fakeSQSClient{}
	service := newTestPreprocessorService(s3Client, sqsClient)

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if len(sqsClient.publishedMessages) != 1 {
		t.Fatalf("publishedMessages length = %d, want 1", len(sqsClient.publishedMessages))
	}

	rows := uploadedResultRows(t, s3Client)
	if len(rows) != 3 {
		t.Fatalf("result rows length = %d, want 3; rows = %#v", len(rows), rows)
	}
	assertResultRow(t, rows[1], []string{"REC-001", "2", "customer-1", entity.PreprocessStatusAccepted, ""})
	assertResultRow(t, rows[2], []string{"", "3", "customer-2", entity.PreprocessStatusRejected, "csv row has 3 columns, expected 6"})
}

func TestPreprocessorService_InvalidS3EventReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{}, &fakeSQSClient{})

	err := service.Process(context.Background(), "{")
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "failed to parse s3 event") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "failed to parse s3 event")
	}
}

func TestPreprocessorService_S3DownloadFailureReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{downloadErr: errors.New("download failed")}, &fakeSQSClient{})

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "failed to download source csv from s3") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "failed to download source csv from s3")
	}
}

func TestPreprocessorService_InvalidFileExtensionReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{downloadBody: []byte("not empty")}, &fakeSQSClient{})

	err := service.Process(context.Background(), validS3EventBody("input/cards.txt"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "only .csv files are supported") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "only .csv files are supported")
	}
}

func TestPreprocessorService_EmptyFileReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{downloadBody: []byte{}}, &fakeSQSClient{})

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "greater than 0") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "greater than 0")
	}
}

func TestPreprocessorService_WrongCSVHeaderReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{
		downloadBody: []byte("customer_id,card_type,email\ncustomer-1,VISA,jane@example.com"),
	}, &fakeSQSClient{})

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "does not match expected header") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "does not match expected header")
	}
}

func TestPreprocessorService_S3UploadFailureReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{
		downloadBody: []byte(validCSV("customer-1,VISA,4111111111111111,12/29,Jane Doe,jane@example.com")),
		uploadErr:    errors.New("upload failed"),
	}, &fakeSQSClient{})

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "failed to upload preprocess result csv to s3") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "failed to upload preprocess result csv to s3")
	}
}

func TestPreprocessorService_SQSPublishFailureReturnsError(t *testing.T) {
	service := newTestPreprocessorService(&fakeS3Client{
		downloadBody: []byte(validCSV("customer-1,VISA,4111111111111111,12/29,Jane Doe,jane@example.com")),
	}, &fakeSQSClient{publishErr: errors.New("publish failed")})

	err := service.Process(context.Background(), validS3EventBody("input/cards.csv"))
	if err == nil {
		t.Fatal("Process returned nil error")
	}
	if !strings.Contains(err.Error(), "failed to publish accepted record to sqs") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "failed to publish accepted record to sqs")
	}
}

type fakeS3Client struct {
	downloadBody   []byte
	downloadErr    error
	downloadBucket string
	downloadKey    string
	uploadErr      error
	uploadBucket   string
	uploadKey      string
	uploadBody     []byte
}

func (client *fakeS3Client) Download(_ context.Context, bucket string, key string) ([]byte, error) {
	client.downloadBucket = bucket
	client.downloadKey = key
	if client.downloadErr != nil {
		return nil, client.downloadErr
	}

	return client.downloadBody, nil
}

func (client *fakeS3Client) Upload(_ context.Context, bucket string, key string, body []byte) error {
	client.uploadBucket = bucket
	client.uploadKey = key
	client.uploadBody = body
	return client.uploadErr
}

type fakeSQSClient struct {
	publishErr        error
	publishedMessages []entity.OnboardingMessage
}

func (client *fakeSQSClient) Publish(_ context.Context, message entity.OnboardingMessage) error {
	if client.publishErr != nil {
		return client.publishErr
	}

	client.publishedMessages = append(client.publishedMessages, message)
	return nil
}

func newTestPreprocessorService(s3Client *fakeS3Client, sqsClient *fakeSQSClient) *PreprocessorService {
	return NewPreprocessorServiceWithIDGenerator(
		s3Client,
		sqsClient,
		"output-bucket",
		1024,
		func() string { return "job-123" },
	)
}

func validS3EventBody(objectKey string) string {
	return `{"Records":[{"s3":{"bucket":{"name":"input-bucket"},"object":{"key":"` + objectKey + `"}}}]}`
}

func validCSV(rows ...string) string {
	allRows := append([]string{"customer_id,card_type,card_number,expiry_date,holder_name,email"}, rows...)
	return strings.Join(allRows, "\n")
}

func uploadedResultRows(t *testing.T, s3Client *fakeS3Client) [][]string {
	t.Helper()

	rows, err := csv.NewReader(strings.NewReader(string(s3Client.uploadBody))).ReadAll()
	if err != nil {
		t.Fatalf("uploaded result csv could not be parsed: %v", err)
	}

	return rows
}

func assertResultRow(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("row = %#v, want %#v", got, want)
	}
}
