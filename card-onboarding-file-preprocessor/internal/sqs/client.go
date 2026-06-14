package sqs

import (
	"context"
	"errors"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

type SQSClient interface {
	Publish(ctx context.Context, message entity.OnboardingMessage) error
}

type Client struct {
	queueURL string
}

func NewClient(queueURL string) *Client {
	return &Client{queueURL: queueURL}
}

func (client *Client) Publish(_ context.Context, _ entity.OnboardingMessage) error {
	return errors.New("sqs client is not implemented")
}
