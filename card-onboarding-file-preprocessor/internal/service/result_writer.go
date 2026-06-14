package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

var preprocessResultCSVHeader = []string{
	"record_id",
	"row_number",
	"customer_id",
	"preprocess_status",
	"error_message",
}

func GeneratePreprocessResultCSV(results []entity.PreprocessResult) (string, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write(preprocessResultCSVHeader); err != nil {
		return "", fmt.Errorf("failed to write preprocess result csv header: %w", err)
	}

	for _, result := range results {
		if err := writer.Write([]string{
			result.RecordID,
			strconv.Itoa(result.RowNumber),
			result.CustomerID,
			result.PreprocessStatus,
			result.ErrorMessage,
		}); err != nil {
			return "", fmt.Errorf("failed to write preprocess result csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("failed to flush preprocess result csv: %w", err)
	}

	return buffer.String(), nil
}
