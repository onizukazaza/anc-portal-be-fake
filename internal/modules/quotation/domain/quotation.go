package domain

import "time"

// Quotation represents a quotation record from the external ERP database.
type Quotation struct {
	ID          string    `json:"id"`
	DocNo       string    `json:"docNo"`
	CustomerID  string    `json:"customerId"`
	TotalAmount float64   `json:"totalAmount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}
