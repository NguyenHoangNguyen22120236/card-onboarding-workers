package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client interface {
	Download(ctx context.Context, bucket string, key string) ([]byte, error)
	Upload(ctx context.Context, bucket string, key string, body []byte) error
}

type api interface {
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
}

type Client struct {
	api     api
	initErr error
}

func NewClient() *Client {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return &Client{initErr: fmt.Errorf("load aws config: %w", err)}
	}

	return NewClientFromConfig(cfg)
}

func NewClientFromConfig(cfg aws.Config) *Client {
	return NewClientWithAPI(awss3.NewFromConfig(cfg))
}

func NewClientWithAPI(api api) *Client {
	return &Client{api: api}
}

func (client *Client) Download(ctx context.Context, bucket string, key string) ([]byte, error) {
	if client.initErr != nil {
		return nil, client.initErr
	}
	if client.api == nil {
		return nil, fmt.Errorf("s3 api is nil")
	}

	output, err := client.api.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	if output.Body == nil {
		return nil, fmt.Errorf("get object returned nil body")
	}
	defer output.Body.Close()

	fileBytes, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}

	return fileBytes, nil
}

func (client *Client) Upload(ctx context.Context, bucket string, key string, body []byte) error {
	if client.initErr != nil {
		return client.initErr
	}
	if client.api == nil {
		return fmt.Errorf("s3 api is nil")
	}

	_, err := client.api.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}
