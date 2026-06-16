package sqs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

func TestClient_PublishMarshalsMessageAndCallsSendMessage(t *testing.T) {
	api := &fakeAPI{}
	client := NewClientWithAPI("https://sqs.us-east-1.amazonaws.com/123/worker", api)
	message := entity.OnboardingMessage{
		CorrelationID: "correlation-1",
		JobID:         "job-1",
		RecordID:      "REC-001",
		SourceFile:    "cards.csv",
		RowNumber:     2,
		CustomerID:    "customer-1",
		CardType:      "VISA",
		CardNumber:    "4111111111111111",
		ExpiryDate:    "12/29",
		HolderName:    "Jane Doe",
		Email:         "jane@example.com",
	}

	err := client.Publish(context.Background(), message)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if aws.ToString(api.sendMessageInput.QueueUrl) != "https://sqs.us-east-1.amazonaws.com/123/worker" {
		t.Fatalf("SendMessage queue URL = %q, want %q", aws.ToString(api.sendMessageInput.QueueUrl), "https://sqs.us-east-1.amazonaws.com/123/worker")
	}

	var got entity.OnboardingMessage
	if err := json.Unmarshal([]byte(aws.ToString(api.sendMessageInput.MessageBody)), &got); err != nil {
		t.Fatalf("SendMessage body is not valid onboarding message JSON: %v", err)
	}
	if got != message {
		t.Fatalf("SendMessage body = %#v, want %#v", got, message)
	}
}

func TestClient_PublishReturnsSendMessageError(t *testing.T) {
	client := NewClientWithAPI("queue-url", &fakeAPI{sendMessageErr: errors.New("send failed")})

	err := client.Publish(context.Background(), entity.OnboardingMessage{CustomerID: "customer-1"})
	if err == nil {
		t.Fatal("Publish returned nil error")
	}
	if !strings.Contains(err.Error(), "send message") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "send message")
	}
}

type fakeAPI struct {
	sendMessageInput *awssqs.SendMessageInput
	sendMessageErr   error
}

func (api *fakeAPI) SendMessage(_ context.Context, input *awssqs.SendMessageInput, _ ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error) {
	api.sendMessageInput = input
	if api.sendMessageErr != nil {
		return nil, api.sendMessageErr
	}

	return &awssqs.SendMessageOutput{}, nil
}
