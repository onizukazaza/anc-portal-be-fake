package domain

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Roles        []string
}

type Session struct {
	AccessToken string   `json:"accessToken"`
	UserID      string   `json:"userId"`
	Roles       []string `json:"roles"`
}
