package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-worker/internal/client"
	"card-onboarding-workers/card-onboarding-worker/internal/entity"
)

func TestWorkerService_ValidMessageCallsOnboardClient(t *testing.T) {
	onboardClient := &fakeOnboardClient{}
	service := NewWorkerService(onboardClient)

	err := service.ProcessMessage(context.Background(), validMessageBody())

	if err != nil {
		t.Fatalf("ProcessMessage returned error: %v", err)
	}
	if onboardClient.callCount != 1 {
		t.Fatalf("onboard client call count = %d, want %d", onboardClient.callCount, 1)
	}
	if onboardClient.message.CardNumber != "4111111111111111" {
		t.Fatalf("onboard message card number = %q, want %q", onboardClient.message.CardNumber, "4111111111111111")
	}
}

func TestWorkerService_BusinessValidationFailureDoesNotCallOnboardClientAndReturnsNil(t *testing.T) {
	onboardClient := &fakeOnboardClient{}
	service := NewWorkerService(onboardClient)

	err := service.ProcessMessage(context.Background(), messageBodyWithCardType("DISCOVER"))

	if err != nil {
		t.Fatalf("ProcessMessage returned error: %v", err)
	}
	if onboardClient.callCount != 0 {
		t.Fatalf("onboard client call count = %d, want %d", onboardClient.callCount, 0)
	}
}

func TestWorkerService_MasksCardNumberBeforeLogging(t *testing.T) {
	var logBuffer bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&logBuffer)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
	})

	service := NewWorkerService(&fakeOnboardClient{})

	err := service.ProcessMessage(context.Background(), validMessageBody())

	if err != nil {
		t.Fatalf("ProcessMessage returned error: %v", err)
	}

	logOutput := logBuffer.String()
	if strings.Contains(logOutput, "4111111111111111") {
		t.Fatalf("log output contains unmasked card number: %q", logOutput)
	}
	if !strings.Contains(logOutput, "************1111") {
		t.Fatalf("log output = %q, want masked card number", logOutput)
	}
}

func TestWorkerService_Onboard4xxReturnsNil(t *testing.T) {
	onboardClient := &fakeOnboardClient{
		err: fmt.Errorf("%w: status 400", client.ErrOnboardBusiness),
	}
	service := NewWorkerService(onboardClient)

	err := service.ProcessMessage(context.Background(), validMessageBody())

	if err != nil {
		t.Fatalf("ProcessMessage returned error: %v", err)
	}
	if onboardClient.callCount != 1 {
		t.Fatalf("onboard client call count = %d, want %d", onboardClient.callCount, 1)
	}
}

func TestWorkerService_Onboard5xxReturnsError(t *testing.T) {
	onboardClient := &fakeOnboardClient{
		err: fmt.Errorf("%w: status 500", client.ErrOnboardTechnical),
	}
	service := NewWorkerService(onboardClient)

	err := service.ProcessMessage(context.Background(), validMessageBody())

	if err == nil {
		t.Fatal("ProcessMessage returned nil error")
	}
	if !errors.Is(err, client.ErrOnboardTechnical) {
		t.Fatalf("error = %v, want ErrOnboardTechnical", err)
	}
}

func TestWorkerService_OnboardTimeoutReturnsError(t *testing.T) {
	onboardClient := &fakeOnboardClient{
		err: fmt.Errorf("%w: context deadline exceeded", client.ErrOnboardTimeout),
	}
	service := NewWorkerService(onboardClient)

	err := service.ProcessMessage(context.Background(), validMessageBody())

	if err == nil {
		t.Fatal("ProcessMessage returned nil error")
	}
	if !errors.Is(err, client.ErrOnboardTimeout) {
		t.Fatalf("error = %v, want ErrOnboardTimeout", err)
	}
}

type fakeOnboardClient struct {
	callCount int
	message   entity.OnboardingMessage
	err       error
}

func (c *fakeOnboardClient) OnboardCard(ctx context.Context, message entity.OnboardingMessage) error {
	c.callCount++
	c.message = message
	return c.err
}

func validMessageBody() string {
	return `{
		"correlationId": "corr-123",
		"jobId": "JOB-20260606-001",
		"recordId": "REC-001",
		"sourceFile": "cards_20260606.csv",
		"rowNumber": 2,
		"customerId": "CUST001",
		"cardType": "VISA",
		"cardNumber": "4111111111111111",
		"expiryDate": "12/28",
		"holderName": "Nguyen Van A",
		"email": "a@example.com"
	}`
}

func messageBodyWithCardType(cardType string) string {
	return fmt.Sprintf(`{
		"correlationId": "corr-123",
		"jobId": "JOB-20260606-001",
		"recordId": "REC-001",
		"sourceFile": "cards_20260606.csv",
		"rowNumber": 2,
		"customerId": "CUST001",
		"cardType": %q,
		"cardNumber": "4111111111111111",
		"expiryDate": "12/28",
		"holderName": "Nguyen Van A",
		"email": "a@example.com"
	}`, cardType)
}
