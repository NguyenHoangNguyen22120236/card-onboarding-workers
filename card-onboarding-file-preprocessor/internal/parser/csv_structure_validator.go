package parser

import (
	"errors"
	"fmt"
	"slices"
)

var expectedCSVHeader = []string{
	"customer_id",
	"card_type",
	"card_number",
	"expiry_date",
	"holder_name",
	"email",
}

func ValidateCSVHeader(header []string) error {
	if len(header) == 0 {
		return errors.New("csv file is missing header row")
	}

	if !slices.Equal(header, expectedCSVHeader) {
		return fmt.Errorf("csv header does not match expected header %v", expectedCSVHeader)
	}

	return nil
}

func ValidateCSVDataRows(rows [][]string) error {
	if len(rows) == 0 {
		return errors.New("csv file must contain at least one data row")
	}

	return nil
}

func ValidateCSVRow(row []string) error {
	if len(row) != len(expectedCSVHeader) {
		return fmt.Errorf("csv row has %d columns, expected %d", len(row), len(expectedCSVHeader))
	}

	return nil
}
