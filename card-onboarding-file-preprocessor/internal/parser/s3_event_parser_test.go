package parser

import (
	"strings"
	"testing"
)

func TestParseS3Event_ValidS3Event(t *testing.T) {
	body := `{
		"Records": [
			{
				"eventName": "ObjectCreated:Put",
				"s3": {
					"bucket": {
						"name": "card-onboarding-input"
					},
					"object": {
						"key": "incoming/cards+20260613.csv"
					}
				}
			}
		]
	}`

	event, err := ParseS3Event(body)
	if err != nil {
		t.Fatalf("ParseS3Event returned error: %v", err)
	}

	if event.BucketName != "card-onboarding-input" {
		t.Errorf("BucketName = %q, want %q", event.BucketName, "card-onboarding-input")
	}
	if event.ObjectKey != "incoming/cards 20260613.csv" {
		t.Errorf("ObjectKey = %q, want %q", event.ObjectKey, "incoming/cards 20260613.csv")
	}
	if event.SourceFileName != "cards 20260613.csv" {
		t.Errorf("SourceFileName = %q, want %q", event.SourceFileName, "cards 20260613.csv")
	}
}

func TestParseS3Event_EmptyBody(t *testing.T) {
	_, err := ParseS3Event("")
	if err == nil {
		t.Fatal("ParseS3Event returned nil error")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "empty")
	}
}

func TestParseS3Event_InvalidJSON(t *testing.T) {
	_, err := ParseS3Event(`{"Records": [`)
	if err == nil {
		t.Fatal("ParseS3Event returned nil error")
	}

	if !strings.Contains(err.Error(), "invalid s3 event JSON") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "invalid s3 event JSON")
	}
}

func TestParseS3Event_MissingRecords(t *testing.T) {
	_, err := ParseS3Event(`{"Records": []}`)
	if err == nil {
		t.Fatal("ParseS3Event returned nil error")
	}

	if !strings.Contains(err.Error(), "no Records") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "no Records")
	}
}

func TestParseS3Event_MissingBucket(t *testing.T) {
	body := `{
		"Records": [
			{
				"s3": {
					"bucket": {},
					"object": {
						"key": "incoming/cards.csv"
					}
				}
			}
		]
	}`

	_, err := ParseS3Event(body)
	if err == nil {
		t.Fatal("ParseS3Event returned nil error")
	}

	if !strings.Contains(err.Error(), "bucket name is missing") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "bucket name is missing")
	}
}

func TestParseS3Event_MissingObjectKey(t *testing.T) {
	body := `{
		"Records": [
			{
				"s3": {
					"bucket": {
						"name": "card-onboarding-input"
					},
					"object": {}
				}
			}
		]
	}`

	_, err := ParseS3Event(body)
	if err == nil {
		t.Fatal("ParseS3Event returned nil error")
	}

	if !strings.Contains(err.Error(), "object key is missing") {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), "object key is missing")
	}
}
