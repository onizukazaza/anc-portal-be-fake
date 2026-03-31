package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

var allowedSorts = pagination.AllowedColumns{
	"created_at":   true,
	"doc_no":       true,
	"total_amount": true,
	"status":       true,
}

var selectColumns = []string{"id", "doc_no", "customer_id", "total_amount", "status", "created_at"}

// QuotationRepository reads quotation data from an external database.
type QuotationRepository struct {
	pool *pgxpool.Pool
}

// NewQuotationRepository creates a repository backed by the given external DB pool.
func NewQuotationRepository(pool *pgxpool.Pool) *QuotationRepository {
	return &QuotationRepository{pool: pool}
}

func (r *QuotationRepository) FindByID(ctx context.Context, id string) (*domain.Quotation, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationRepo).Start(ctx, "FindByID")
	defer span.End()

	const q = `
		SELECT id, doc_no, customer_id, total_amount, status, created_at
		FROM quotations
		WHERE id = $1
		LIMIT 1
	`

	var qt domain.Quotation
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&qt.ID, &qt.DocNo, &qt.CustomerID, &qt.TotalAmount, &qt.Status, &qt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &qt, nil
}

func (r *QuotationRepository) FindByCustomerID(ctx context.Context, customerID string, pg pagination.Request) ([]domain.Quotation, int, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationRepo).Start(ctx, "FindByCustomerID")
	defer span.End()

	// 1 query ดึงทั้ง data + total ด้วย COUNT(*) OVER()
	q := pagination.From("quotations").
		Select(selectColumns...).
		Where("customer_id = $1", 1).
		Paginate(pg, "created_at", allowedSorts)

	rows, err := r.pool.Query(ctx, q.DataSQL(), customerID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []domain.Quotation
	var total int
	for rows.Next() {
		var qt domain.Quotation
		if err := rows.Scan(&qt.ID, &qt.DocNo, &qt.CustomerID, &qt.TotalAmount, &qt.Status, &qt.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		results = append(results, qt)
	}
	return results, total, rows.Err()
}
