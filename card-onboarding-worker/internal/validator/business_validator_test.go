package validator

import (
	"strings"
	"testing"

	"card-onboarding-workers/card-onboarding-worker/internal/entity"
)

func TestValidateBusinessRules_Success(t *testing.T) {
	message := validOnboardingMessage()

	if err := ValidateBusinessRules(message); err != nil {
		t.Fatalf("ValidateBusinessRules returned error: %v", err)
	}
}

func TestValidateBusinessRules_MissingCustomerID(t *testing.T) {
	message := validOnboardingMessage()
	message.CustomerID = ""

	err := ValidateBusinessRules(message)

	assertValidationError(t, err, "customerId is required")
}

func TestValidateBusinessRules_InvalidCardType(t *testing.T) {
	message := validOnboardingMessage()
	message.CardType = "DISCOVER"

	err := ValidateBusinessRules(message)

	assertValidationError(t, err, "cardType must be one of VISA, MASTERCARD, AMEX")
}

func TestValidateBusinessRules_InvalidCardNumber(t *testing.T) {
	message := validOnboardingMessage()
	message.CardNumber = "4111-1111-1111-1111"

	err := ValidateBusinessRules(message)

	assertValidationError(t, err, "cardNumber must be numeric")
}

func TestValidateBusinessRules_InvalidExpiryDate(t *testing.T) {
	message := validOnboardingMessage()
	message.ExpiryDate = "2028-12"

	err := ValidateBusinessRules(message)

	assertValidationError(t, err, "expiryDate must follow MM/YY format")
}

func TestValidateBusinessRules_InvalidEmail(t *testing.T) {
	message := validOnboardingMessage()
	message.Email = "not-an-email"

	err := ValidateBusinessRules(message)

	assertValidationError(t, err, "email must be valid email format")
}

func validOnboardingMessage() entity.OnboardingMessage {
	return entity.OnboardingMessage{
		CorrelationID: "corr-123",
		JobID:         "JOB-20260606-001",
		RecordID:      "REC-001",
		SourceFile:    "cards_20260606.csv",
		RowNumber:     2,
		CustomerID:    "CUST001",
		CardType:      "VISA",
		CardNumber:    "4111111111111111",
		ExpiryDate:    "12/28",
		HolderName:    "Nguyen Van A",
		Email:         "a@example.com",
	}
}

func assertValidationError(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatal("ValidateBusinessRules returned nil error")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want message to contain %q", err.Error(), want)
	}
}
