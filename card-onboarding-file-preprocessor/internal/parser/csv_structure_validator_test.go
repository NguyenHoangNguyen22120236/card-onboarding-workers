package parser

import (
	"strings"
	"testing"
)

func TestValidateCSVHeader_ExpectedHeader(t *testing.T) {
	err := ValidateCSVHeader([]string{"customer_id", "card_type", "card_number", "expiry_date", "holder_name", "email"})
	if err != nil {
		t.Fatalf("ValidateCSVHeader returned error: %v", err)
	}
}

func TestValidateCSVHeader_MissingHeader(t *testing.T) {
	err := ValidateCSVHeader(nil)
	if err == nil {
		t.Fatal("ValidateCSVHeader returned nil error")
	}

	if !strings.Contains(err.Error(), "missing header") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "missing header")
	}
}

func TestValidateCSVHeader_WrongHeader(t *testing.T) {
	err := ValidateCSVHeader([]string{"customer_id", "card_type", "email"})
	if err == nil {
		t.Fatal("ValidateCSVHeader returned nil error")
	}

	if !strings.Contains(err.Error(), "does not match expected header") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "does not match expected header")
	}
}

func TestValidateCSVDataRows_NoDataRows(t *testing.T) {
	err := ValidateCSVDataRows(nil)
	if err == nil {
		t.Fatal("ValidateCSVDataRows returned nil error")
	}

	if !strings.Contains(err.Error(), "at least one data row") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "at least one data row")
	}
}

func TestValidateCSVRow_FewerThanExpectedColumns(t *testing.T) {
	err := ValidateCSVRow([]string{"customer-1", "VISA"})
	if err == nil {
		t.Fatal("ValidateCSVRow returned nil error")
	}

	if !strings.Contains(err.Error(), "2 columns, expected 6") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "2 columns, expected 6")
	}
}

func TestValidateCSVRow_MoreThanExpectedColumns(t *testing.T) {
	err := ValidateCSVRow([]string{"customer-1", "VISA", "4111111111111111", "12/29", "Jane Doe", "jane@example.com", "extra"})
	if err == nil {
		t.Fatal("ValidateCSVRow returned nil error")
	}

	if !strings.Contains(err.Error(), "7 columns, expected 6") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "7 columns, expected 6")
	}
}
