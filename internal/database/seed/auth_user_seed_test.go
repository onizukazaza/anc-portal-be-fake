package seed

import (
	"encoding/json"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBuildAuthUserSeedRecordSuccess(t *testing.T) {
	record, err := buildAuthUserSeedRecord(AuthUserSeedInput{
		ID:       "usr-admin-001",
		Username: "Admin",
		Password: "admin123",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if record.ID != "usr-admin-001" {
		t.Fatalf("expected id usr-admin-001, got %s", record.ID)
	}
	if record.Username != "admin" {
		t.Fatalf("expected username admin, got %s", record.Username)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte("admin123")); err != nil {
		t.Fatalf("expected bcrypt password hash, got compare error %v", err)
	}

	var roles []string
	if err := json.Unmarshal(record.RolesJSON, &roles); err != nil {
		t.Fatalf("unmarshal roles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "admin" {
		t.Fatalf("expected roles [admin], got %+v", roles)
	}
}

func TestBuildAuthUserSeedRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   AuthUserSeedInput
		wantErr string
	}{
		{
			name:    "missing id",
			input:   AuthUserSeedInput{Username: "admin", Password: "admin123", Roles: []string{"admin"}},
			wantErr: "id is required",
		},
		{
			name:    "missing username",
			input:   AuthUserSeedInput{ID: "usr-admin-001", Password: "admin123", Roles: []string{"admin"}},
			wantErr: "username is required",
		},
		{
			name:    "missing password",
			input:   AuthUserSeedInput{ID: "usr-admin-001", Username: "admin", Roles: []string{"admin"}},
			wantErr: "password is required",
		},
		{
			name:    "missing roles",
			input:   AuthUserSeedInput{ID: "usr-admin-001", Username: "admin", Password: "admin123"},
			wantErr: "roles are required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildAuthUserSeedRecord(tc.input)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}
