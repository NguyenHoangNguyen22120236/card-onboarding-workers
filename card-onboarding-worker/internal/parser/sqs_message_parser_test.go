package parser

import (
	"strings"
	"testing"
)

func TestParseSQSMessage_ValidMessage(t *testing.T) {
	body := `{
		"correlationId": "corr-123",
		"jobId": "JOB-20260606-001",
		"recordId": "REC-001",
		"sourceFile": "cards_20260606.csv",
		"rowNumber": 2,
		"customerId": "CUST001",
		"cardType": "VISA",
		"cardNumber": "4111111111111111",
		"expiryDate": "12/28",
		"holderName": "Nguyen Van A",
		"email": "a@example.com"
	}`

	message, err := ParseSQSMessage(body)
	if err != nil {
		t.Fatalf("ParseSQSMessage returned error: %v", err)
	}

	if message.CorrelationID != "corr-123" {
		t.Errorf("CorrelationID = %q, want %q", message.CorrelationID, "corr-123")
	}
	if message.JobID != "JOB-20260606-001" {
		t.Errorf("JobID = %q, want %q", message.JobID, "JOB-20260606-001")
	}
	if message.RecordID != "REC-001" {
		t.Errorf("RecordID = %q, want %q", message.RecordID, "REC-001")
	}
	if message.SourceFile != "cards_20260606.csv" {
		t.Errorf("SourceFile = %q, want %q", message.SourceFile, "cards_20260606.csv")
	}
	if message.RowNumber != 2 {
		t.Errorf("RowNumber = %d, want %d", message.RowNumber, 2)
	}
	if message.CustomerID != "CUST001" {
		t.Errorf("CustomerID = %q, want %q", message.CustomerID, "CUST001")
	}
	if message.CardType != "VISA" {
		t.Errorf("CardType = %q, want %q", message.CardType, "VISA")
	}
	if message.CardNumber != "4111111111111111" {
		t.Errorf("CardNumber = %q, want %q", message.CardNumber, "4111111111111111")
	}
	if message.ExpiryDate != "12/28" {
		t.Errorf("ExpiryDate = %q, want %q", message.ExpiryDate, "12/28")
	}
	if message.HolderName != "Nguyen Van A" {
		t.Errorf("HolderName = %q, want %q", message.HolderName, "Nguyen Van A")
	}
	if message.Email != "a@example.com" {
		t.Errorf("Email = %q, want %q", message.Email, "a@example.com")
	}
}

func TestParseSQSMessage_InvalidJSON(t *testing.T) {
	_, err := ParseSQSMessage(`{"correlationId": "corr-123"`)
	if err == nil {
		t.Fatal("ParseSQSMessage returned nil error")
	}

	if !strings.Contains(err.Error(), "invalid sqs message JSON") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "invalid sqs message JSON")
	}
}

func TestParseSQSMessage_EmptyBody(t *testing.T) {
	_, err := ParseSQSMessage("")
	if err == nil {
		t.Fatal("ParseSQSMessage returned nil error")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "empty")
	}
}
