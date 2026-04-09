package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/domain"
)

// ExternalDBService handles external database health check logic.
type ExternalDBService interface {
	// CheckAll checks connectivity of all registered external databases.
	CheckAll(ctx context.Context) []domain.DBStatus

	// CheckByName checks connectivity of a specific external database.
	CheckByName(ctx context.Context, name string) domain.DBStatus
}
