package service

import (
	"encoding/csv"
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func TestGeneratePreprocessResultCSV_HasCorrectHeader(t *testing.T) {
	rows := generatePreprocessResultCSVRows(t, nil)

	assertRow(t, rows[0], []string{
		"record_id",
		"row_number",
		"customer_id",
		"preprocess_status",
		"error_message",
	})
}

func TestGeneratePreprocessResultCSV_IncludesAcceptedRow(t *testing.T) {
	rows := generatePreprocessResultCSVRows(t, []entity.PreprocessResult{
		{
			RecordID:         "REC-001",
			RowNumber:        2,
			CustomerID:       "CUST001",
			PreprocessStatus: entity.PreprocessStatusAccepted,
		},
	})

	assertRow(t, rows[1], []string{
		"REC-001",
		"2",
		"CUST001",
		entity.PreprocessStatusAccepted,
		"",
	})
}

func TestGeneratePreprocessResultCSV_IncludesRejectedRow(t *testing.T) {
	rows := generatePreprocessResultCSVRows(t, []entity.PreprocessResult{
		{
			RecordID:         "REC-002",
			RowNumber:        3,
			PreprocessStatus: entity.PreprocessStatusRejected,
			ErrorMessage:     "malformed row: expected 6 columns but got 4",
		},
	})

	assertRow(t, rows[1], []string{
		"REC-002",
		"3",
		"",
		entity.PreprocessStatusRejected,
		"malformed row: expected 6 columns but got 4",
	})
}

func generatePreprocessResultCSVRows(t *testing.T, results []entity.PreprocessResult) [][]string {
	t.Helper()

	content, err := GeneratePreprocessResultCSV(results)
	if err != nil {
		t.Fatalf("GeneratePreprocessResultCSV returned error: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(content)).ReadAll()
	if err != nil {
		t.Fatalf("generated csv could not be parsed: %v", err)
	}

	return rows
}

func assertRow(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("row length = %d, want %d; row = %#v", len(got), len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf("row[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
