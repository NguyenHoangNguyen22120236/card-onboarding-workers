package handler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler_SuccessfulRecordReturnsNil(t *testing.T) {
	preprocessor := &fakePreprocessor{}
	handler := New(preprocessor)

	err := handler.Handle(context.Background(), sqsEvent("message-1", "body-1"))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	assertProcessedBodies(t, preprocessor, []string{"body-1"})
}

func TestHandler_FailedRecordReturnsError(t *testing.T) {
	preprocessor := &fakePreprocessor{err: errors.New("process failed")}
	handler := New(preprocessor)

	err := handler.Handle(context.Background(), sqsEvent("message-1", "body-1"))
	if err == nil {
		t.Fatal("Handle returned nil error")
	}
	if !strings.Contains(err.Error(), "process failed") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "process failed")
	}

	assertProcessedBodies(t, preprocessor, []string{"body-1"})
}

func TestHandler_MultipleRecordsSuccessReturnsNil(t *testing.T) {
	preprocessor := &fakePreprocessor{}
	handler := New(preprocessor)

	err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-1", Body: "body-1"},
		{MessageId: "message-2", Body: "body-2"},
		{MessageId: "message-3", Body: "body-3"},
	}})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	assertProcessedBodies(t, preprocessor, []string{"body-1", "body-2", "body-3"})
}

func TestHandler_SecondRecordFailureReturnsError(t *testing.T) {
	preprocessor := &fakePreprocessor{errByBody: map[string]error{
		"body-2": errors.New("second record failed"),
	}}
	handler := New(preprocessor)

	err := handler.Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "message-1", Body: "body-1"},
		{MessageId: "message-2", Body: "body-2"},
		{MessageId: "message-3", Body: "body-3"},
	}})
	if err == nil {
		t.Fatal("Handle returned nil error")
	}
	if !strings.Contains(err.Error(), "second record failed") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "second record failed")
	}

	assertProcessedBodies(t, preprocessor, []string{"body-1", "body-2"})
}

type fakePreprocessor struct {
	err       error
	errByBody map[string]error
	bodies    []string
}

func (preprocessor *fakePreprocessor) Process(_ context.Context, rawSQSMessageBody string) error {
	preprocessor.bodies = append(preprocessor.bodies, rawSQSMessageBody)
	if preprocessor.errByBody != nil && preprocessor.errByBody[rawSQSMessageBody] != nil {
		return preprocessor.errByBody[rawSQSMessageBody]
	}

	return preprocessor.err
}

func sqsEvent(messageID string, body string) events.SQSEvent {
	return events.SQSEvent{Records: []events.SQSMessage{{MessageId: messageID, Body: body}}}
}

func assertProcessedBodies(t *testing.T, preprocessor *fakePreprocessor, want []string) {
	t.Helper()

	if !reflect.DeepEqual(preprocessor.bodies, want) {
		t.Fatalf("processed bodies = %#v, want %#v", preprocessor.bodies, want)
	}
}
