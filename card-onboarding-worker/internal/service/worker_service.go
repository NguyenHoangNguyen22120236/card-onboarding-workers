package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"card-onboarding-workers/card-onboarding-worker/internal/client"
	"card-onboarding-workers/card-onboarding-worker/internal/entity"
	"card-onboarding-workers/card-onboarding-worker/internal/parser"
	"card-onboarding-workers/card-onboarding-worker/internal/util"
	"card-onboarding-workers/card-onboarding-worker/internal/validator"
	"card-onboarding-workers/internal/observability"
)

const (
	workerComponent = "card-onboarding-worker"

	metricMessageReceived           = "card_onboarding_worker.message_received.count"
	metricBusinessValidationSuccess = "card_onboarding_worker.business_validation_success.count"
	metricBusinessValidationFailed  = "card_onboarding_worker.business_validation_failed.count"
	metricOnboardSuccess            = "card_onboarding_worker.onboard_success.count"
	metricOnboardFailed             = "card_onboarding_worker.onboard_failed.count"
	metricOnboard4xx                = "card_onboarding_worker.onboard_4xx.count"
	metricOnboard5xx                = "card_onboarding_worker.onboard_5xx.count"
	metricOnboardTimeout            = "card_onboarding_worker.onboard_timeout.count"
	metricProcessingDurationMs      = "card_onboarding_worker.processing.duration_ms"
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
	startedAt := time.Now()

	message, err := parser.ParseSQSMessage(body)
	if err != nil {
		return fmt.Errorf("parse sqs message: %w", err)
	}

	fields := metricFields(message)
	defer func() {
		observability.LogMetric(
			observability.Metric{
				Name:  metricProcessingDurationMs,
				Value: float64(time.Since(startedAt).Milliseconds()),
				Unit:  observability.UnitMilliseconds,
			},
			fields,
		)
	}()
	logCountMetric(metricMessageReceived, fields)

	maskedCardNumber := util.MaskCardNumber(message.CardNumber)
	log.Printf(
		"processing onboarding message correlationId=%s jobId=%s recordId=%s cardNumber=%s",
		message.CorrelationID,
		message.JobID,
		message.RecordID,
		maskedCardNumber,
	)

	if err := validator.ValidateBusinessRules(message); err != nil {
		logCountMetric(metricBusinessValidationFailed, fieldsWithError(fields, "business_validation_failed", err))
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
	logCountMetric(metricBusinessValidationSuccess, fields)

	if err := s.onboardClient.OnboardCard(ctx, message); err != nil {
		if errors.Is(err, client.ErrOnboardBusiness) {
			logCountMetric(metricOnboardFailed, fieldsWithError(fields, "onboard_failed", err))
			logCountMetric(metricOnboard4xx, fieldsWithError(fields, "onboard_4xx", err))
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

		if errors.Is(err, client.ErrOnboardTimeout) {
			logCountMetric(metricOnboardTimeout, fieldsWithError(fields, "onboard_timeout", err))
		} else if errors.Is(err, client.ErrOnboardTechnical) {
			logCountMetric(metricOnboard5xx, fieldsWithError(fields, "onboard_5xx", err))
		}
		logCountMetric(metricOnboardFailed, fieldsWithError(fields, "onboard_failed", err))

		// For technical errors, we return the error to trigger retry
		return fmt.Errorf("onboard card: %w", err)
	}

	logCountMetric(metricOnboardSuccess, fields)
	return nil
}

func metricFields(message entity.OnboardingMessage) observability.Fields {
	return observability.Fields{
		Environment:   os.Getenv("ENVIRONMENT_NAME"),
		Component:     workerComponent,
		CorrelationID: message.CorrelationID,
		JobID:         message.JobID,
		RecordID:      message.RecordID,
		CustomerID:    message.CustomerID,
		SourceFile:    message.SourceFile,
		RowNumber:     message.RowNumber,
	}
}

func logCountMetric(name string, fields observability.Fields) {
	observability.LogMetric(
		observability.Metric{
			Name:  name,
			Value: 1,
			Unit:  observability.UnitCount,
		},
		fields,
	)
}

func fieldsWithError(fields observability.Fields, errorCode string, err error) observability.Fields {
	fields.ErrorCode = errorCode
	fields.ErrorMessage = err.Error()
	return fields
}
