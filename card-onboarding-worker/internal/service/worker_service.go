package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"card-onboarding-workers/card-onboarding-worker/internal/client"
	"card-onboarding-workers/card-onboarding-worker/internal/parser"
	"card-onboarding-workers/card-onboarding-worker/internal/util"
	"card-onboarding-workers/card-onboarding-worker/internal/validator"
)

type WorkerService struct {
	onboardClient client.OnboardClient
}

func NewWorkerService(onboardClient client.OnboardClient) *WorkerService {
	return &WorkerService{
		onboardClient: onboardClient,
	}
}

func (s *WorkerService) ProcessMessage(ctx context.Context, body string) error {
	message, err := parser.ParseSQSMessage(body)
	if err != nil {
		return fmt.Errorf("parse sqs message: %w", err)
	}

	maskedCardNumber := util.MaskCardNumber(message.CardNumber)
	log.Printf(
		"processing onboarding message correlationId=%s jobId=%s recordId=%s cardNumber=%s",
		message.CorrelationID,
		message.JobID,
		message.RecordID,
		maskedCardNumber,
	)

	if err := validator.ValidateBusinessRules(message); err != nil {
		log.Printf(
			"business validation failed correlationId=%s jobId=%s recordId=%s cardNumber=%s error=%v",
			message.CorrelationID,
			message.JobID,
			message.RecordID,
			maskedCardNumber,
			err,
		)
		return nil
	}

	if err := s.onboardClient.OnboardCard(ctx, message); err != nil {
		if errors.Is(err, client.ErrOnboardBusiness) {
			log.Printf(
				"onboard-service business error correlationId=%s jobId=%s recordId=%s cardNumber=%s error=%v",
				message.CorrelationID,
				message.JobID,
				message.RecordID,
				maskedCardNumber,
				err,
			)
			return nil
		}

		// For technical errors, we return the error to trigger retry
		return fmt.Errorf("onboard card: %w", err)
	}

	return nil
}
