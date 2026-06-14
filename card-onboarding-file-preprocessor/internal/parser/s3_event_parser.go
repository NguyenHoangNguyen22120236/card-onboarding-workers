package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"

	"card-onboarding-workers/card-onboarding-file-preprocessor/internal/entity"
)

type s3Event struct {
	Records []s3EventRecord `json:"Records"`
}

type s3EventRecord struct {
	S3 s3Entity `json:"s3"`
}

type s3Entity struct {
	Bucket s3Bucket `json:"bucket"`
	Object s3Object `json:"object"`
}

type s3Bucket struct {
	Name string `json:"name"`
}

type s3Object struct {
	Key string `json:"key"`
}

func ParseS3Event(body string) (entity.S3FileEvent, error) {
	if body == "" {
		return entity.S3FileEvent{}, errors.New("s3 event body is empty")
	}

	var event s3Event
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return entity.S3FileEvent{}, fmt.Errorf("invalid s3 event JSON: %w", err)
	}

	if len(event.Records) == 0 {
		return entity.S3FileEvent{}, errors.New("s3 event contains no Records")
	}

	record := event.Records[0]
	bucketName := record.S3.Bucket.Name
	if bucketName == "" {
		return entity.S3FileEvent{}, errors.New("s3 event bucket name is missing")
	}

	objectKey := record.S3.Object.Key
	if objectKey == "" {
		return entity.S3FileEvent{}, errors.New("s3 event object key is missing")
	}

	decodedObjectKey, err := url.QueryUnescape(objectKey)
	if err != nil {
		return entity.S3FileEvent{}, fmt.Errorf("invalid s3 event object key encoding: %w", err)
	}

	return entity.S3FileEvent{
		BucketName:     bucketName,
		ObjectKey:      decodedObjectKey,
		SourceFileName: path.Base(decodedObjectKey),
	}, nil
}
