package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler_OneSuccessfulSQSRecordReturnsNil(t *testing.T) {
	workerService := &fakeWorkerService{}
	handler := NewHandler(workerService)

	err := handler.HandleSQSEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "message-1", Body: "body-1"},
		},
	})

	if err != nil {
		t.Fatalf("HandleSQSEvent returned error: %v", err)
	}
	if workerService.callCount != 1 {
		t.Fatalf("ProcessMessage call count = %d, want %d", workerService.callCount, 1)
	}
	if workerService.bodies[0] != "body-1" {
		t.Fatalf("ProcessMessage body = %q, want %q", workerService.bodies[0], "body-1")
	}
}

func TestHandler_MultipleSuccessfulSQSRecordsReturnNil(t *testing.T) {
	workerService := &fakeWorkerService{}
	handler := NewHandler(workerService)

	err := handler.HandleSQSEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "message-1", Body: "body-1"},
			{MessageId: "message-2", Body: "body-2"},
		},
	})

	if err != nil {
		t.Fatalf("HandleSQSEvent returned error: %v", err)
	}
	if workerService.callCount != 2 {
		t.Fatalf("ProcessMessage call count = %d, want %d", workerService.callCount, 2)
	}
	if workerService.bodies[0] != "body-1" {
		t.Fatalf("first ProcessMessage body = %q, want %q", workerService.bodies[0], "body-1")
	}
	if workerService.bodies[1] != "body-2" {
		t.Fatalf("second ProcessMessage body = %q, want %q", workerService.bodies[1], "body-2")
	}
}

func TestHandler_OneFailedRecordReturnsError(t *testing.T) {
	expectedErr := errors.New("process failed")
	workerService := &fakeWorkerService{errByBody: map[string]error{"body-1": expectedErr}}
	handler := NewHandler(workerService)

	err := handler.HandleSQSEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "message-1", Body: "body-1"},
		},
	})

	if err == nil {
		t.Fatal("HandleSQSEvent returned nil error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("HandleSQSEvent error = %v, want wrapped %v", err, expectedErr)
	}
	if workerService.callCount != 1 {
		t.Fatalf("ProcessMessage call count = %d, want %d", workerService.callCount, 1)
	}
}

func TestHandler_WhenSecondRecordFailsReturnsError(t *testing.T) {
	expectedErr := errors.New("second record failed")
	workerService := &fakeWorkerService{errByBody: map[string]error{"body-2": expectedErr}}
	handler := NewHandler(workerService)

	err := handler.HandleSQSEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "message-1", Body: "body-1"},
			{MessageId: "message-2", Body: "body-2"},
		},
	})

	if err == nil {
		t.Fatal("HandleSQSEvent returned nil error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("HandleSQSEvent error = %v, want wrapped %v", err, expectedErr)
	}
	if workerService.callCount != 2 {
		t.Fatalf("ProcessMessage call count = %d, want %d", workerService.callCount, 2)
	}
}

type fakeWorkerService struct {
	callCount int
	bodies    []string
	errByBody map[string]error
}

func (s *fakeWorkerService) ProcessMessage(ctx context.Context, body string) error {
	s.callCount++
	s.bodies = append(s.bodies, body)
	return s.errByBody[body]
}
