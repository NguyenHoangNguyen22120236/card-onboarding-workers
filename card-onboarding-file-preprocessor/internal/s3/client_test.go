package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestClient_DownloadCallsGetObjectAndReturnsBytes(t *testing.T) {
	api := &fakeAPI{
		getObjectOutput: &awss3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("customer csv")),
		},
	}
	client := NewClientWithAPI(api)

	got, err := client.Download(context.Background(), "input-bucket", "incoming/cards.csv")
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if string(got) != "customer csv" {
		t.Fatalf("Download() = %q, want %q", string(got), "customer csv")
	}
	if aws.ToString(api.getObjectInput.Bucket) != "input-bucket" {
		t.Fatalf("GetObject bucket = %q, want %q", aws.ToString(api.getObjectInput.Bucket), "input-bucket")
	}
	if aws.ToString(api.getObjectInput.Key) != "incoming/cards.csv" {
		t.Fatalf("GetObject key = %q, want %q", aws.ToString(api.getObjectInput.Key), "incoming/cards.csv")
	}
}

func TestClient_DownloadReturnsGetObjectError(t *testing.T) {
	client := NewClientWithAPI(&fakeAPI{getObjectErr: errors.New("get failed")})

	_, err := client.Download(context.Background(), "input-bucket", "incoming/cards.csv")
	if err == nil {
		t.Fatal("Download returned nil error")
	}
	if !strings.Contains(err.Error(), "get object") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "get object")
	}
}

func TestClient_UploadCallsPutObject(t *testing.T) {
	api := &fakeAPI{}
	client := NewClientWithAPI(api)

	err := client.Upload(context.Background(), "output-bucket", "processed/result.csv", []byte("result csv"))
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if aws.ToString(api.putObjectInput.Bucket) != "output-bucket" {
		t.Fatalf("PutObject bucket = %q, want %q", aws.ToString(api.putObjectInput.Bucket), "output-bucket")
	}
	if aws.ToString(api.putObjectInput.Key) != "processed/result.csv" {
		t.Fatalf("PutObject key = %q, want %q", aws.ToString(api.putObjectInput.Key), "processed/result.csv")
	}

	body, err := io.ReadAll(api.putObjectInput.Body)
	if err != nil {
		t.Fatalf("PutObject body could not be read: %v", err)
	}
	if string(body) != "result csv" {
		t.Fatalf("PutObject body = %q, want %q", string(body), "result csv")
	}
}

func TestClient_UploadReturnsPutObjectError(t *testing.T) {
	client := NewClientWithAPI(&fakeAPI{putObjectErr: errors.New("put failed")})

	err := client.Upload(context.Background(), "output-bucket", "processed/result.csv", []byte("result csv"))
	if err == nil {
		t.Fatal("Upload returned nil error")
	}
	if !strings.Contains(err.Error(), "put object") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "put object")
	}
}

type fakeAPI struct {
	getObjectInput  *awss3.GetObjectInput
	getObjectOutput *awss3.GetObjectOutput
	getObjectErr    error
	putObjectInput  *awss3.PutObjectInput
	putObjectErr    error
}

func (api *fakeAPI) GetObject(_ context.Context, input *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	api.getObjectInput = input
	if api.getObjectErr != nil {
		return nil, api.getObjectErr
	}

	return api.getObjectOutput, nil
}

func (api *fakeAPI) PutObject(_ context.Context, input *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	api.putObjectInput = input
	if api.putObjectErr != nil {
		return nil, api.putObjectErr
	}

	return &awss3.PutObjectOutput{}, nil
}
