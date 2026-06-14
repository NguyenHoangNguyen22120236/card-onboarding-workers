package parser

import (
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func TestParseCSV_ValidCSVParsesAcceptedRecords(t *testing.T) {
	content := strings.Join([]string{
		"customer_id,card_type,card_number,expiry_date,holder_name,email",
		"customer-1,ANY_CARD,not-a-number,not-a-date,Jane Doe,not-an-email",
		"customer-2,SOMETHING_ELSE,1234,2026-12,John Doe,john@example.com",
	}, "\n")

	records, rejectedResults, err := ParseCSV(content)
	if err != nil {
		t.Fatalf("ParseCSV returned error: %v", err)
	}

	if len(rejectedResults) != 0 {
		t.Fatalf("rejectedResults length = %d, want 0", len(rejectedResults))
	}

	if len(records) != 2 {
		t.Fatalf("records length = %d, want 2", len(records))
	}

	firstRecord := records[0]
	if firstRecord.RowNumber != 2 {
		t.Errorf("RowNumber = %d, want 2", firstRecord.RowNumber)
	}
	if firstRecord.CustomerID != "customer-1" {
		t.Errorf("CustomerID = %q, want %q", firstRecord.CustomerID, "customer-1")
	}
	if firstRecord.CardType != "ANY_CARD" {
		t.Errorf("CardType = %q, want %q", firstRecord.CardType, "ANY_CARD")
	}
	if firstRecord.CardNumber != "not-a-number" {
		t.Errorf("CardNumber = %q, want %q", firstRecord.CardNumber, "not-a-number")
	}
	if firstRecord.ExpiryDate != "not-a-date" {
		t.Errorf("ExpiryDate = %q, want %q", firstRecord.ExpiryDate, "not-a-date")
	}
	if firstRecord.HolderName != "Jane Doe" {
		t.Errorf("HolderName = %q, want %q", firstRecord.HolderName, "Jane Doe")
	}
	if firstRecord.Email != "not-an-email" {
		t.Errorf("Email = %q, want %q", firstRecord.Email, "not-an-email")
	}
}

func TestParseCSV_MissingHeaderReturnsError(t *testing.T) {
	_, _, err := ParseCSV("")
	if err == nil {
		t.Fatal("ParseCSV returned nil error")
	}

	if !strings.Contains(err.Error(), "missing header") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "missing header")
	}
}

func TestParseCSV_WrongHeaderReturnsError(t *testing.T) {
	content := strings.Join([]string{
		"customer_id,card_type,email",
		"customer-1,ANY_CARD,jane@example.com",
	}, "\n")

	_, _, err := ParseCSV(content)
	if err == nil {
		t.Fatal("ParseCSV returned nil error")
	}

	if !strings.Contains(err.Error(), "does not match expected header") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "does not match expected header")
	}
}

func TestParseCSV_NoDataRowsReturnsError(t *testing.T) {
	_, _, err := ParseCSV("customer_id,card_type,card_number,expiry_date,holder_name,email")
	if err == nil {
		t.Fatal("ParseCSV returned nil error")
	}

	if !strings.Contains(err.Error(), "at least one data row") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "at least one data row")
	}
}

func TestParseCSV_RowWithFewerThanSixColumnsIsRejected(t *testing.T) {
	content := strings.Join([]string{
		"customer_id,card_type,card_number,expiry_date,holder_name,email",
		"customer-1,ANY_CARD,1234",
	}, "\n")

	records, rejectedResults, err := ParseCSV(content)
	if err != nil {
		t.Fatalf("ParseCSV returned error: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("records length = %d, want 0", len(records))
	}

	assertSingleRejectedRow(t, rejectedResults, 2, "customer-1", "3 columns, expected 6")
}

func TestParseCSV_RowWithMoreThanSixColumnsIsRejected(t *testing.T) {
	content := strings.Join([]string{
		"customer_id,card_type,card_number,expiry_date,holder_name,email",
		"customer-1,ANY_CARD,1234,not-a-date,Jane Doe,jane@example.com,extra",
	}, "\n")

	records, rejectedResults, err := ParseCSV(content)
	if err != nil {
		t.Fatalf("ParseCSV returned error: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("records length = %d, want 0", len(records))
	}

	assertSingleRejectedRow(t, rejectedResults, 2, "customer-1", "7 columns, expected 6")
}

func assertSingleRejectedRow(t *testing.T, rejectedResults []entity.PreprocessResult, rowNumber int, customerID string, message string) {
	t.Helper()

	if len(rejectedResults) != 1 {
		t.Fatalf("rejectedResults length = %d, want 1", len(rejectedResults))
	}

	rejectedResult := rejectedResults[0]
	if rejectedResult.RowNumber != rowNumber {
		t.Errorf("RowNumber = %d, want %d", rejectedResult.RowNumber, rowNumber)
	}
	if rejectedResult.CustomerID != customerID {
		t.Errorf("CustomerID = %q, want %q", rejectedResult.CustomerID, customerID)
	}
	if rejectedResult.PreprocessStatus != entity.PreprocessStatusRejected {
		t.Errorf("PreprocessStatus = %q, want %q", rejectedResult.PreprocessStatus, entity.PreprocessStatusRejected)
	}
	if !strings.Contains(rejectedResult.ErrorMessage, message) {
		t.Errorf("ErrorMessage = %q, want message to contain %q", rejectedResult.ErrorMessage, message)
	}
}
