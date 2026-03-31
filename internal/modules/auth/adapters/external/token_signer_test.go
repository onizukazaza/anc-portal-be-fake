package external_test

import (
	"context"
	"testing"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/external"
)

func TestJWTTokenSigner_SignAndVerify(t *testing.T) {
	signer := external.NewJWTTokenSigner("test-secret-key-32chars!!", 1*time.Hour)
	ctx := context.Background()

	token, err := signer.SignAccessToken(ctx, "user-123", []string{"admin", "viewer"})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("SignAccessToken() returned empty token")
	}

	claims, err := signer.VerifyAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "admin" || claims.Roles[1] != "viewer" {
		t.Errorf("Roles = %v, want [admin viewer]", claims.Roles)
	}
}

func TestJWTTokenSigner_ExpiredToken(t *testing.T) {
	signer := external.NewJWTTokenSigner("test-secret-key-32chars!!", -1*time.Hour)
	ctx := context.Background()

	token, err := signer.SignAccessToken(ctx, "user-123", []string{"admin"})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}

	_, err = signer.VerifyAccessToken(ctx, token)
	if err == nil {
		t.Fatal("VerifyAccessToken() expected error for expired token, got nil")
	}
}

func TestJWTTokenSigner_WrongSecret(t *testing.T) {
	signer1 := external.NewJWTTokenSigner("secret-key-one-xxxxxxxxx", 1*time.Hour)
	signer2 := external.NewJWTTokenSigner("secret-key-two-xxxxxxxxx", 1*time.Hour)
	ctx := context.Background()

	token, err := signer1.SignAccessToken(ctx, "user-123", []string{"admin"})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}

	_, err = signer2.VerifyAccessToken(ctx, token)
	if err == nil {
		t.Fatal("VerifyAccessToken() expected error for wrong secret, got nil")
	}
}

func TestJWTTokenSigner_InvalidToken(t *testing.T) {
	signer := external.NewJWTTokenSigner("test-secret-key-32chars!!", 1*time.Hour)
	ctx := context.Background()

	_, err := signer.VerifyAccessToken(ctx, "not-a-jwt")
	if err == nil {
		t.Fatal("VerifyAccessToken() expected error for garbage token, got nil")
	}
}

func TestSimpleTokenSigner_SignAndVerify(t *testing.T) {
	signer := external.NewSimpleTokenSigner()
	ctx := context.Background()

	token, err := signer.SignAccessToken(ctx, "user-456", []string{"editor"})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}

	claims, err := signer.VerifyAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if claims.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-456")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "editor" {
		t.Errorf("Roles = %v, want [editor]", claims.Roles)
	}
}

func TestSimpleTokenSigner_InvalidFormat(t *testing.T) {
	signer := external.NewSimpleTokenSigner()
	ctx := context.Background()

	_, err := signer.VerifyAccessToken(ctx, "not-dev-token")
	if err == nil {
		t.Fatal("VerifyAccessToken() expected error for invalid format, got nil")
	}
}
