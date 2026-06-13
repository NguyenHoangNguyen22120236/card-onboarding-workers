package entity

type CSVRecord struct {
	RowNumber  int    `json:"rowNumber"`
	CustomerID string `json:"customerId"`
	CardType   string `json:"cardType"`
	CardNumber string `json:"cardNumber"`
	ExpiryDate string `json:"expiryDate"`
	HolderName string `json:"holderName"`
	Email      string `json:"email"`
}
