package domain

// DBStatus represents the health/diagnostic result for one external database.
type DBStatus struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	CurrentDatabase string `json:"currentDatabase,omitempty"`
	Version         string `json:"version,omitempty"`
	Error           string `json:"error,omitempty"`
}
