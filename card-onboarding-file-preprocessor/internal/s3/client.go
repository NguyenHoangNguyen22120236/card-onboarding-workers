package s3

import (
	"context"
	"errors"
)

type S3Client interface {
	Download(ctx context.Context, bucket string, key string) ([]byte, error)
	Upload(ctx context.Context, bucket string, key string, body []byte) error
}

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (client *Client) Download(_ context.Context, _ string, _ string) ([]byte, error) {
	return nil, errors.New("s3 client is not implemented")
}

func (client *Client) Upload(_ context.Context, _ string, _ string, _ []byte) error {
	return errors.New("s3 client is not implemented")
}
