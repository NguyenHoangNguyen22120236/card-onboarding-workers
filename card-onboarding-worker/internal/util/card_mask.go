package util

import "strings"

const visibleCardNumberSuffixLength = 4

func MaskCardNumber(cardNumber string) string {
	if len(cardNumber) <= visibleCardNumberSuffixLength {
		return cardNumber
	}

	maskedLength := len(cardNumber) - visibleCardNumberSuffixLength
	return strings.Repeat("*", maskedLength) + cardNumber[maskedLength:]
}
