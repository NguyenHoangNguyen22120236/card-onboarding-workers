package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"card-onboarding-workers/card-onboarding-worker/internal/entity"

	onboardapi "github.com/NguyenHoangNguyen22120236/card-onboarding-services/onboard-service/pkg/onboard"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var (
	ErrOnboardBusiness  = errors.New("onboard business error")
	ErrOnboardTechnical = errors.New("onboard technical error")
	ErrOnboardTimeout   = errors.New("onboard timeout")
)

type OnboardClient interface {
	OnboardCard(ctx context.Context, message entity.OnboardingMessage) error
}

type OnboardClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

type generatedOnboardClient struct {
	client  onboardapi.ClientWithResponsesInterface
	timeout time.Duration
}

func NewOnboardClient(config OnboardClientConfig) (OnboardClient, error) {
	generatedClient, err := onboardapi.NewClientWithResponses(config.BaseURL)
	if err != nil {
		return nil, err
	}

	return &generatedOnboardClient{
		client:  generatedClient,
		timeout: config.Timeout,
	}, nil
}

func (c *generatedOnboardClient) OnboardCard(ctx context.Context, message entity.OnboardingMessage) error {
	callCtx, cancel := onboardContextWithTimeout(ctx, c.timeout)
	defer cancel()

	correlationID := onboardapi.CorrelationIdHeader(message.CorrelationID)
	params := &onboardapi.OnboardCardParams{
		XCorrelationId: &correlationID,
	}

	response, err := c.client.OnboardCardWithResponse(
		callCtx,
		params,
		onboardapi.OnboardCardJSONRequestBody(onboardCardRequest(message)),
		onboardCorrelationHeader(message.CorrelationID),
	)
	if err != nil {
		return mapOnboardCallError(callCtx, err)
	}
	if response == nil {
		return fmt.Errorf("%w: empty onboard response", ErrOnboardTechnical)
	}

	statusCode := response.StatusCode()
	switch {
	case statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices:
		return nil
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return fmt.Errorf("%w: onboard-service returned status %d", ErrOnboardBusiness, statusCode)
	case statusCode >= http.StatusInternalServerError:
		return fmt.Errorf("%w: onboard-service returned status %d", ErrOnboardTechnical, statusCode)
	default:
		return fmt.Errorf("%w: unexpected onboard-service status %d", ErrOnboardTechnical, statusCode)
	}
}

func onboardCardRequest(message entity.OnboardingMessage) onboardapi.OnboardCardRequest {
	return onboardapi.OnboardCardRequest{
		CorrelationId: message.CorrelationID,
		JobId:         message.JobID,
		RecordId:      message.RecordID,
		SourceFile:    message.SourceFile,
		RowNumber:     int32(message.RowNumber),
		CustomerId:    message.CustomerID,
		CardType:      message.CardType,
		CardNumber:    message.CardNumber,
		ExpiryDate:    message.ExpiryDate,
		HolderName:    message.HolderName,
		Email:         openapi_types.Email(message.Email),
	}
}

func onboardCorrelationHeader(correlationID string) onboardapi.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if correlationID != "" {
			req.Header.Set("X-Correlation-Id", correlationID)
		}
		return nil
	}
}

func onboardContextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func mapOnboardCallError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrOnboardTimeout, err)
	}
	return fmt.Errorf("%w: %v", ErrOnboardTechnical, err)
}
