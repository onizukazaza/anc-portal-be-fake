package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// QuotationSyncer sync ข้อมูล quotations จาก external DB → main DB.
type QuotationSyncer struct {
	source *pgxpool.Pool // external DB (read)
	dest   *pgxpool.Pool // main DB (write)
}

// NewQuotationSyncer สร้าง syncer สำหรับตาราง quotations.
func NewQuotationSyncer(source, dest *pgxpool.Pool) *QuotationSyncer {
	return &QuotationSyncer{source: source, dest: dest}
}

func (s *QuotationSyncer) Name() string { return "quotations" }

func (s *QuotationSyncer) Sync(ctx context.Context, req SyncRequest) (*SyncResult, error) {
	start := time.Now()
	result := &SyncResult{
		Table:     "quotations",
		Mode:      req.Mode,
		StartedAt: start,
	}

	// 1. Count total จาก source
	total, err := s.countSource(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("count source: %w", err)
	}
	result.Total = total

	if total == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}

	// 2. Sync ทีละ batch
	offset := 0
	for offset < total {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		rows, err := s.fetchBatch(ctx, req, offset)
		if err != nil {
			return nil, fmt.Errorf("fetch batch offset=%d: %w", offset, err)
		}

		if len(rows) == 0 {
			break
		}

		upserted, err := s.upsertBatch(ctx, rows)
		if err != nil {
			return nil, fmt.Errorf("upsert batch offset=%d: %w", offset, err)
		}

		result.Inserted += upserted
		offset += len(rows)

		log.L().Debug().
			Int("offset", offset).
			Int("total", total).
			Int("batch_upserted", upserted).
			Msg("quotation batch synced")
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (s *QuotationSyncer) countSource(ctx context.Context, req SyncRequest) (int, error) {
	var count int
	var err error

	if req.Mode == ModeIncremental {
		err = s.source.QueryRow(ctx,
			"SELECT COUNT(*) FROM quotations WHERE updated_at > $1",
			req.Since,
		).Scan(&count)
	} else {
		err = s.source.QueryRow(ctx, "SELECT COUNT(*) FROM quotations").Scan(&count)
	}

	return count, err
}

type quotationRow struct {
	ID          string
	DocNo       string
	CustomerID  string
	TotalAmount float64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *QuotationSyncer) fetchBatch(ctx context.Context, req SyncRequest, offset int) ([]quotationRow, error) {
	var query string
	var args []any

	baseQuery := `
		SELECT id, doc_no, customer_id, total_amount, status, created_at, updated_at
		FROM quotations
	`

	if req.Mode == ModeIncremental {
		query = baseQuery + " WHERE updated_at > $1 ORDER BY updated_at ASC LIMIT $2 OFFSET $3"
		args = []any{req.Since, req.BatchSize, offset}
	} else {
		query = baseQuery + " ORDER BY id ASC LIMIT $1 OFFSET $2"
		args = []any{req.BatchSize, offset}
	}

	rows, err := s.source.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []quotationRow
	for rows.Next() {
		var r quotationRow
		if err := rows.Scan(&r.ID, &r.DocNo, &r.CustomerID, &r.TotalAmount, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		batch = append(batch, r)
	}

	return batch, rows.Err()
}

func (s *QuotationSyncer) upsertBatch(ctx context.Context, rows []quotationRow) (int, error) {
	tx, err := s.dest.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	const upsertSQL = `
		INSERT INTO quotations (id, doc_no, customer_id, total_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			doc_no       = EXCLUDED.doc_no,
			customer_id  = EXCLUDED.customer_id,
			total_amount = EXCLUDED.total_amount,
			status       = EXCLUDED.status,
			updated_at   = EXCLUDED.updated_at
	`

	count := 0
	for _, r := range rows {
		_, err := tx.Exec(ctx, upsertSQL,
			r.ID, r.DocNo, r.CustomerID, r.TotalAmount, r.Status, r.CreatedAt, r.UpdatedAt,
		)
		if err != nil {
			return count, fmt.Errorf("upsert id=%s: %w", r.ID, err)
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	committed = true

	return count, nil
}
