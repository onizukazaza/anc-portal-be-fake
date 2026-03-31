package external

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
)

// JWTTokenSigner signs and verifies JWT tokens using HS256.
type JWTTokenSigner struct {
	secret []byte
	expiry time.Duration
}

// NewJWTTokenSigner creates a new JWT signer with the given secret and token expiry.
func NewJWTTokenSigner(secret string, expiry time.Duration) *JWTTokenSigner {
	return &JWTTokenSigner{
		secret: []byte(secret),
		expiry: expiry,
	}
}

// jwtClaims is the JWT payload.
type jwtClaims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

func (s *JWTTokenSigner) SignAccessToken(_ context.Context, userID string, roles []string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
			Issuer:    "anc-portal",
		},
		Roles: roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTTokenSigner) VerifyAccessToken(_ context.Context, tokenString string) (*ports.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return &ports.Claims{
		UserID: claims.Subject,
		Roles:  claims.Roles,
	}, nil
}
