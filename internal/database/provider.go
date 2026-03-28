package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Provider defines database access points used by application modules.
type Provider interface {
	// main external
	Main() *pgxpool.Pool

	// external รับชื่อฐานข้อมูลภายนอก → คืน pgxpool หรือ error ถ้าไม่พบหรือมีปัญหา
	External(name string) (*pgxpool.Pool, error)

	Read() *pgxpool.Pool
	Write() *pgxpool.Pool
	HealthCheck(ctx context.Context) error
	Close()
}
