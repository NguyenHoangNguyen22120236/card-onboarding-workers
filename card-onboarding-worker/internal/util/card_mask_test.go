package util

import "testing"

func TestMaskCardNumber(t *testing.T) {
	tests := []struct {
		name       string
		cardNumber string
		want       string
	}{
		{
			name:       "masks all but last four digits",
			cardNumber: "4111111111111111",
			want:       "************1111",
		},
		{
			name:       "masks five digit card number",
			cardNumber: "12345",
			want:       "*2345",
		},
		{
			name:       "leaves four digit card number unchanged",
			cardNumber: "1234",
			want:       "1234",
		},
		{
			name:       "leaves shorter card number unchanged",
			cardNumber: "123",
			want:       "123",
		},
		{
			name:       "leaves empty card number unchanged",
			cardNumber: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskCardNumber(tt.cardNumber); got != tt.want {
				t.Fatalf("MaskCardNumber returned %q, want %q", got, tt.want)
			}
		})
	}
}
