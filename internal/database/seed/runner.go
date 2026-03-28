package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, db *pgxpool.Pool, serviceType string) error {
	switch serviceType {
	case "auth_user":
		if err := SeedAuthUsers(ctx, db); err != nil {
			return fmt.Errorf("seed auth users: %w", err)
		}
	default:
		return fmt.Errorf("unsupported service_type: %s", serviceType)
	}

	return nil
}
