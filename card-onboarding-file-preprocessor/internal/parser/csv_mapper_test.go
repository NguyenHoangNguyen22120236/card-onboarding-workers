package parser

import (
	"reflect"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func TestMapCSVRecordsToOnboardingMessages_MapsOneAcceptedRecord(t *testing.T) {
	records := []entity.CSVRecord{
		{
			RowNumber:  7,
			CustomerID: "customer-1",
			CardType:   "UNKNOWN_CARD_TYPE",
			CardNumber: "not-a-card-number",
			ExpiryDate: "not-an-expiry-date",
			HolderName: "Jane Doe",
			Email:      "not-an-email",
		},
	}

	messages := MapCSVRecordsToOnboardingMessages(records, "correlation-1", "job-1", "cards.csv")

	want := []entity.OnboardingMessage{
		{
			CorrelationID: "correlation-1",
			JobID:         "job-1",
			RecordID:      "REC-001",
			SourceFile:    "cards.csv",
			RowNumber:     7,
			CustomerID:    "customer-1",
			CardType:      "UNKNOWN_CARD_TYPE",
			CardNumber:    "not-a-card-number",
			ExpiryDate:    "not-an-expiry-date",
			HolderName:    "Jane Doe",
			Email:         "not-an-email",
		},
	}

	if !reflect.DeepEqual(messages, want) {
		t.Fatalf("messages = %#v, want %#v", messages, want)
	}
}

func TestMapCSVRecordsToOnboardingMessages_MapsMultipleAcceptedRecordsWithSequentialRecordIDs(t *testing.T) {
	records := []entity.CSVRecord{
		{
			RowNumber:  2,
			CustomerID: "customer-1",
			CardType:   "VISA",
			CardNumber: "4111111111111111",
			ExpiryDate: "12/29",
			HolderName: "Jane Doe",
			Email:      "jane@example.com",
		},
		{
			RowNumber:  4,
			CustomerID: "customer-2",
			CardType:   "MASTERCARD",
			CardNumber: "5555555555554444",
			ExpiryDate: "10/30",
			HolderName: "John Doe",
			Email:      "john@example.com",
		},
		{
			RowNumber:  6,
			CustomerID: "customer-3",
			CardType:   "AMEX",
			CardNumber: "378282246310005",
			ExpiryDate: "08/31",
			HolderName: "Alex Doe",
			Email:      "alex@example.com",
		},
	}

	messages := MapCSVRecordsToOnboardingMessages(records, "correlation-1", "job-1", "cards.csv")

	if len(messages) != 3 {
		t.Fatalf("messages length = %d, want 3", len(messages))
	}

	wantRecordIDs := []string{"REC-001", "REC-002", "REC-003"}
	wantRowNumbers := []int{2, 4, 6}
	for index, message := range messages {
		if message.RecordID != wantRecordIDs[index] {
			t.Errorf("messages[%d].RecordID = %q, want %q", index, message.RecordID, wantRecordIDs[index])
		}
		if message.RowNumber != wantRowNumbers[index] {
			t.Errorf("messages[%d].RowNumber = %d, want %d", index, message.RowNumber, wantRowNumbers[index])
		}
	}
}
