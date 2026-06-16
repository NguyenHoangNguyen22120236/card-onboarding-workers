package service

import (
	"context"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func TestLocalE2ESimulation_PreprocessorFakeS3ToFakeSQS(t *testing.T) {
	s3Client := &fakeS3Client{
		downloadBody: []byte(validCSV(
			"CUST-E2E-001,VISA,4111111111111111,12/29,Alex Customer,alex@example.com",
		)),
	}
	sqsClient := &fakeSQSClient{}
	preprocessor := NewPreprocessorServiceWithIDGenerator(
		s3Client,
		sqsClient,
		"local-output-bucket",
		1024,
		func() string { return "local-job-001" },
	)

	err := preprocessor.Process(context.Background(), validS3EventBody("incoming/e2e_cards.csv"))
	if err != nil {
		t.Fatalf("Process() error = %v, want nil", err)
	}

	if s3Client.downloadBucket != "input-bucket" || s3Client.downloadKey != "incoming/e2e_cards.csv" {
		t.Fatalf("Download() bucket/key = %q/%q, want input-bucket/incoming/e2e_cards.csv", s3Client.downloadBucket, s3Client.downloadKey)
	}
	if s3Client.uploadBucket != "local-output-bucket" {
		t.Fatalf("Upload() bucket = %q, want local-output-bucket", s3Client.uploadBucket)
	}
	if s3Client.uploadKey != "processed/local-job-001/e2e_cards.csv_preprocess_result.csv" {
		t.Fatalf("Upload() key = %q, want processed result key", s3Client.uploadKey)
	}
	if len(s3Client.uploadBody) == 0 {
		t.Fatal("Upload() body is empty, want generated preprocessing result CSV")
	}

	rows, err := csv.NewReader(strings.NewReader(string(s3Client.uploadBody))).ReadAll()
	if err != nil {
		t.Fatalf("uploaded result CSV cannot be parsed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("result row count = %d, want header plus one accepted row; rows = %#v", len(rows), rows)
	}
	assertResultRow(t, rows[1], []string{"REC-001", "2", "CUST-E2E-001", entity.PreprocessStatusAccepted, ""})

	wantMessages := []entity.OnboardingMessage{
		{
			CorrelationID: "local-job-001",
			JobID:         "local-job-001",
			RecordID:      "REC-001",
			SourceFile:    "e2e_cards.csv",
			RowNumber:     2,
			CustomerID:    "CUST-E2E-001",
			CardType:      "VISA",
			CardNumber:    "4111111111111111",
			ExpiryDate:    "12/29",
			HolderName:    "Alex Customer",
			Email:         "alex@example.com",
		},
	}
	if !reflect.DeepEqual(sqsClient.publishedMessages, wantMessages) {
		t.Fatalf("publishedMessages = %#v, want %#v", sqsClient.publishedMessages, wantMessages)
	}
}
