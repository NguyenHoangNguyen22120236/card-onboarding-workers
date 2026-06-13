package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, body string) error
}

type Handler struct {
	workerService MessageProcessor
}

func NewHandler(workerService MessageProcessor) *Handler {
	return &Handler{
		workerService: workerService,
	}
}

func (h *Handler) Handle(ctx context.Context, event events.SQSEvent) error {
	return h.HandleSQSEvent(ctx, event)
}

func (h *Handler) HandleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		if err := h.workerService.ProcessMessage(ctx, record.Body); err != nil {
			return fmt.Errorf("process sqs record messageId=%s: %w", record.MessageId, err)
		}
	}

	return nil
}
