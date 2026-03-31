package domain

// DBStatus represents the health/diagnostic result for one external database.
type DBStatus struct {
	Name            string `json:"name"`
	Driver          string `json:"driver,omitempty"`
	Status          string `json:"status"`
	CurrentDatabase string `json:"currentDatabase,omitempty"`
	Version         string `json:"version,omitempty"`
	Error           string `json:"error,omitempty"`
}
