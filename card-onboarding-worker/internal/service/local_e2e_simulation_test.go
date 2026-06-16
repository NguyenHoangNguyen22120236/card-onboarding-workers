package service

import (
	"context"
	"encoding/json"
	"testing"

	"card-onboarding-workers/card-onboarding-worker/internal/entity"
)

func TestLocalE2ESimulation_WorkerPassesAcceptedRecordToOnboardService(t *testing.T) {
	onboardClient := &fakeOnboardClient{}
	worker := NewWorkerService(onboardClient)
	message := entity.OnboardingMessage{
		CorrelationID: "local-job-001",
		JobID:         "local-job-001",
		RecordID:      "REC-001",
		SourceFile:    "e2e_cards.csv",
		RowNumber:     2,
		CustomerID:    "CUST-E2E-001",
		CardType:      "VISA",
		CardNumber:    "4111111111111111",
		ExpiryDate:    "12/29",
		HolderName:    "Alex Customer",
		Email:         "alex@example.com",
	}
	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	err = worker.ProcessMessage(context.Background(), string(body))
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v, want nil", err)
	}

	if onboardClient.callCount != 1 {
		t.Fatalf("onboard-service calls = %d, want 1", onboardClient.callCount)
	}
	if onboardClient.message != message {
		t.Fatalf("onboard-service message = %#v, want %#v", onboardClient.message, message)
	}
}
