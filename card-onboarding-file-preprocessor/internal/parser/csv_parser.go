package parser

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func ParseCSV(content string) ([]entity.CSVRecord, []entity.PreprocessResult, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New("csv file is missing header row")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read csv header: %w", err)
	}
	if err := ValidateCSVHeader(header); err != nil {
		return nil, nil, err
	}

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read csv data rows: %w", err)
	}
	if err := ValidateCSVDataRows(rows); err != nil {
		return nil, nil, err
	}

	records := make([]entity.CSVRecord, 0, len(rows))
	rejectedResults := make([]entity.PreprocessResult, 0)
	for index, row := range rows {
		rowNumber := index + 2
		if err := ValidateCSVRow(row); err != nil {
			rejectedResults = append(rejectedResults, entity.PreprocessResult{
				RowNumber:        rowNumber,
				CustomerID:       customerIDFromRow(row),
				PreprocessStatus: entity.PreprocessStatusRejected,
				ErrorMessage:     err.Error(),
			})
			continue
		}

		records = append(records, entity.CSVRecord{
			RowNumber:  rowNumber,
			CustomerID: row[0],
			CardType:   row[1],
			CardNumber: row[2],
			ExpiryDate: row[3],
			HolderName: row[4],
			Email:      row[5],
		})
	}

	return records, rejectedResults, nil
}

func customerIDFromRow(row []string) string {
	if len(row) == 0 {
		return ""
	}

	return row[0]
}
