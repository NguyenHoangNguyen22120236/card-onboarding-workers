package validator

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"card-onboarding-workers/card-onboarding-worker/internal/entity"
)

var (
	cardNumberPattern = regexp.MustCompile(`^[0-9]+$`)
	expiryDatePattern = regexp.MustCompile(`^(0[1-9]|1[0-2])/[0-9]{2}$`)
)

func ValidateBusinessRules(message entity.OnboardingMessage) error {
	var validationErrors []error

	if strings.TrimSpace(message.CustomerID) == "" {
		validationErrors = append(validationErrors, errors.New("customerId is required"))
	}
	if strings.TrimSpace(message.CardType) == "" {
		validationErrors = append(validationErrors, errors.New("cardType is required"))
	} else if !isSupportedCardType(message.CardType) {
		validationErrors = append(validationErrors, errors.New("cardType must be one of VISA, MASTERCARD, AMEX"))
	}
	if strings.TrimSpace(message.CardNumber) == "" {
		validationErrors = append(validationErrors, errors.New("cardNumber is required"))
	} else if !cardNumberPattern.MatchString(message.CardNumber) {
		validationErrors = append(validationErrors, errors.New("cardNumber must be numeric"))
	}
	if strings.TrimSpace(message.ExpiryDate) == "" {
		validationErrors = append(validationErrors, errors.New("expiryDate is required"))
	} else if !expiryDatePattern.MatchString(message.ExpiryDate) {
		validationErrors = append(validationErrors, errors.New("expiryDate must follow MM/YY format"))
	}
	if strings.TrimSpace(message.HolderName) == "" {
		validationErrors = append(validationErrors, errors.New("holderName is required"))
	}
	if strings.TrimSpace(message.Email) == "" {
		validationErrors = append(validationErrors, errors.New("email is required"))
	} else if _, err := mail.ParseAddress(message.Email); err != nil {
		validationErrors = append(validationErrors, errors.New("email must be valid email format"))
	}

	return errors.Join(validationErrors...)
}

func isSupportedCardType(cardType string) bool {
	switch cardType {
	case "VISA", "MASTERCARD", "AMEX":
		return true
	default:
		return false
	}
}
