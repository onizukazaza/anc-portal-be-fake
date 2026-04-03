package domain

// User — domain entity for authentication.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	Roles        []string
}

// Session — returned after successful login.
type Session struct {
	AccessToken string   `json:"accessToken"`
	UserID      string   `json:"userId"`
	Roles       []string `json:"roles"`
}
