package entity

type OnboardingMessage struct {
	CorrelationID string `json:"correlationId"`
	JobID         string `json:"jobId"`
	RecordID      string `json:"recordId"`
	SourceFile    string `json:"sourceFile"`
	RowNumber     int    `json:"rowNumber"`
	CustomerID    string `json:"customerId"`
	CardType      string `json:"cardType"`
	CardNumber    string `json:"cardNumber"`
	ExpiryDate    string `json:"expiryDate"`
	HolderName    string `json:"holderName"`
	Email         string `json:"email"`
}
