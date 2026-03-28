package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
)

type AuthUserSeedInput struct {
	ID       string
	Username string
	Password string
	Roles    []string
}

type AuthUserSeedRecord struct {
	ID           string
	Username     string
	PasswordHash string
	RolesJSON    []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func SeedAuthUsers(ctx context.Context, db *pgxpool.Pool) error {
	inputs := defaultAuthUserSeedInputs()

	for _, input := range inputs {
		record, err := buildAuthUserSeedRecord(input)
		if err != nil {
			return fmt.Errorf("build auth user seed record username=%s: %w", input.Username, err)
		}

		if err := upsertAuthUser(ctx, db, record); err != nil {
			return fmt.Errorf("upsert auth user username=%s: %w", input.Username, err)
		}
	}

	return nil
}

func defaultAuthUserSeedInputs() []AuthUserSeedInput {
	return []AuthUserSeedInput{
		{
			ID:       "usr-admin-001",
			Username: enum.RoleAdmin,
			Password: "admin123",
			Roles:    []string{enum.RoleAdmin},
		},
		{
			ID:       "usr-ops-001",
			Username: "ops",
			Password: "ops123",
			Roles:    []string{"operator"},
		},
		{
			ID:       "usr-viewer-001",
			Username: enum.RoleViewer,
			Password: "viewer123",
			Roles:    []string{enum.RoleViewer},
		},
	}
}

func buildAuthUserSeedRecord(input AuthUserSeedInput) (*AuthUserSeedRecord, error) {
	id := strings.TrimSpace(input.ID)
	username := strings.TrimSpace(strings.ToLower(input.Username))

	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(input.Roles) == 0 {
		return nil, fmt.Errorf("roles are required")
	}

	rolesJSON, err := json.Marshal(input.Roles)
	if err != nil {
		return nil, fmt.Errorf("marshal roles json: %w", err)
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()

	return &AuthUserSeedRecord{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		RolesJSON:    rolesJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func upsertAuthUser(ctx context.Context, db *pgxpool.Pool, record *AuthUserSeedRecord) error {
	const query = `
		INSERT INTO users (
			id,
			username,
			password_hash,
			roles,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4::jsonb,
			$5,
			$6
		)
		ON CONFLICT (id)
		DO UPDATE SET
			username = EXCLUDED.username,
			password_hash = EXCLUDED.password_hash,
			roles = EXCLUDED.roles,
			updated_at = NOW()
	`

	_, err := db.Exec(
		ctx,
		query,
		record.ID,
		record.Username,
		record.PasswordHash,
		record.RolesJSON,
		record.CreatedAt,
		record.UpdatedAt,
	)
	return err
}

func hashPassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}
