package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

type preprocessor interface {
	Process(ctx context.Context, rawSQSMessageBody string) error
}

type Handler struct {
	preprocessor preprocessor
}

func New(preprocessor preprocessor) *Handler {
	return &Handler{preprocessor: preprocessor}
}

func (handler *Handler) Handle(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		if err := handler.preprocessor.Process(ctx, record.Body); err != nil {
			return fmt.Errorf("failed to process sqs record %q: %w", record.MessageId, err)
		}
	}

	return nil
}
