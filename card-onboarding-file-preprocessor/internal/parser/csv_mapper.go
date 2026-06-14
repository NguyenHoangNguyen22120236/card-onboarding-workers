package parser

import (
	"fmt"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

func MapCSVRecordsToOnboardingMessages(
	records []entity.CSVRecord,
	correlationID string,
	jobID string,
	sourceFile string,
) []entity.OnboardingMessage {
	messages := make([]entity.OnboardingMessage, 0, len(records))
	for index, record := range records {
		messages = append(messages, entity.OnboardingMessage{
			CorrelationID: correlationID,
			JobID:         jobID,
			RecordID:      recordID(index + 1),
			SourceFile:    sourceFile,
			RowNumber:     record.RowNumber,
			CustomerID:    record.CustomerID,
			CardType:      record.CardType,
			CardNumber:    record.CardNumber,
			ExpiryDate:    record.ExpiryDate,
			HolderName:    record.HolderName,
			Email:         record.Email,
		})
	}

	return messages
}

func recordID(sequence int) string {
	return fmt.Sprintf("REC-%03d", sequence)
}
