package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	Publish(ctx context.Context, message entity.OnboardingMessage) error
}

type api interface {
	SendMessage(ctx context.Context, params *awssqs.SendMessageInput, optFns ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error)
}

type Client struct {
	queueURL string
	api      api
	initErr  error
}

func NewClient(queueURL string) *Client {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return &Client{queueURL: queueURL, initErr: fmt.Errorf("load aws config: %w", err)}
	}

	return NewClientFromConfig(queueURL, cfg)
}

func NewClientFromConfig(queueURL string, cfg aws.Config) *Client {
	return NewClientWithAPI(queueURL, awssqs.NewFromConfig(cfg))
}

func NewClientWithAPI(queueURL string, api api) *Client {
	return &Client{queueURL: queueURL, api: api}
}

func (client *Client) Publish(ctx context.Context, message entity.OnboardingMessage) error {
	if client.initErr != nil {
		return client.initErr
	}
	if client.api == nil {
		return fmt.Errorf("sqs api is nil")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal onboarding message: %w", err)
	}

	_, err = client.api.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    aws.String(client.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
