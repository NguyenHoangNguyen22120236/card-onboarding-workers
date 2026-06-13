package parser

import (
	"encoding/json"
	"errors"
	"fmt"

	"card-onboarding-workers/card-onboarding-worker/internal/entity"
)

func ParseSQSMessage(body string) (entity.OnboardingMessage, error) {
	if body == "" {
		return entity.OnboardingMessage{}, errors.New("sqs message body is empty")
	}

	var message entity.OnboardingMessage
	if err := json.Unmarshal([]byte(body), &message); err != nil {
		return entity.OnboardingMessage{}, fmt.Errorf("invalid sqs message JSON: %w", err)
	}

	return message, nil
}
