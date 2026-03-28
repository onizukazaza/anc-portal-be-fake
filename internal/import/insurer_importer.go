package importer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InsurerRow struct {
	Code   string
	Name   string
	Status string
}

func ImportInsurer(db *pgxpool.Pool, filePath string) error {
	csvData, err := ReadCSV(filePath)
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	indexMap := headerIndexMap(csvData.Header)

	required := []string{"code", "name"}
	for _, col := range required {
		if _, ok := indexMap[col]; !ok {
			return fmt.Errorf("missing column: %s", col)
		}
	}

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	for i, row := range csvData.Rows {
		lineNo := i + 2

		if isEmptyRow(row) {
			continue
		}

		data := InsurerRow{
			Code:   strings.TrimSpace(getCell(row, indexMap, "code")),
			Name:   strings.TrimSpace(getCell(row, indexMap, "name")),
			Status: normalizeInsurerStatus(getCell(row, indexMap, "status")),
		}

		if err := validateInsurerRow(&data); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		if err := upsertInsurer(ctx, tx, &data); err != nil {
			return fmt.Errorf("line %d: upsert insurer: %w", lineNo, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true

	return nil
}

func validateInsurerRow(row *InsurerRow) error {
	if row.Code == "" {
		return errors.New("code is required")
	}
	if row.Name == "" {
		return errors.New("name is required")
	}
	if row.Status != "active" && row.Status != "inactive" {
		return fmt.Errorf("invalid status: %s", row.Status)
	}
	return nil
}

func normalizeInsurerStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "active"
	}
	return status
}

func upsertInsurer(ctx context.Context, tx pgx.Tx, row *InsurerRow) error {
	query := `
		INSERT INTO insurer (
			code,
			name,
			status
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (code)
		DO UPDATE SET
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := tx.Exec(ctx, query, row.Code, row.Name, row.Status)
	if err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}

	return nil
}
