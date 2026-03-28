package app

import (
	"context"
	"errors"
	"testing"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
	"golang.org/x/crypto/bcrypt"
)

func TestServiceLogin(t *testing.T) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	testkit.MustNoError(t, err, "generate bcrypt hash")

	dbErr := errors.New("db unavailable")
	signErr := errors.New("sign failed")

	tests := []struct {
		name      string
		repo      *fakeUserRepo
		signer    *fakeTokenSigner
		username  string
		password  string
		wantErr   error
		wantToken string
		wantUID   string
	}{
		{
			name:      "success with plain password",
			repo:      &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
			signer:    &fakeTokenSigner{token: "token-123"},
			username:  "admin",
			password:  "admin123",
			wantToken: "token-123",
			wantUID:   "u1",
		},
		{
			name:      "success with bcrypt password",
			repo:      &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: string(bcryptHash), Roles: []string{"admin"}}},
			signer:    &fakeTokenSigner{token: "token-123"},
			username:  "admin",
			password:  "admin123",
			wantToken: "token-123",
			wantUID:   "u1",
		},
		{
			name:     "invalid credentials",
			repo:     &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
			signer:   &fakeTokenSigner{token: "token-123"},
			username: "admin",
			password: "wrong-pass",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "user repo error",
			repo:     &fakeUserRepo{err: dbErr},
			signer:   &fakeTokenSigner{token: "token-123"},
			username: "admin",
			password: "admin123",
			wantErr:  dbErr,
		},
		{
			name:     "token signer error",
			repo:     &fakeUserRepo{user: &domain.User{ID: "u1", Username: "admin", PasswordHash: "admin123", Roles: []string{"admin"}}},
			signer:   &fakeTokenSigner{err: signErr},
			username: "admin",
			password: "admin123",
			wantErr:  signErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.repo, tc.signer)
			session, err := svc.Login(context.Background(), tc.username, tc.password)

			if tc.wantErr != nil {
				testkit.ErrorIs(t, err, tc.wantErr)
				testkit.Nil(t, session, "session")
				return
			}

			testkit.NoError(t, err)
			testkit.Equal(t, session.AccessToken, tc.wantToken, "token")
			testkit.Equal(t, session.UserID, tc.wantUID, "userID")
		})
	}
}
