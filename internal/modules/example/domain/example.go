package domain

import "time"

// Example — domain entity (pure struct, no imports from other layers).
type Example struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
