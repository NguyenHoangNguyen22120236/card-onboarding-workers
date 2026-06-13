package entity

const (
	PreprocessStatusAccepted = "ACCEPTED"
	PreprocessStatusRejected = "REJECTED"
)

type PreprocessResult struct {
	RecordID         string `json:"recordId"`
	RowNumber        int    `json:"rowNumber"`
	CustomerID       string `json:"customerId"`
	PreprocessStatus string `json:"preprocessStatus"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}
