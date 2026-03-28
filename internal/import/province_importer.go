package importer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProvinceRow struct {
	Code string
	Name string
}

func ImportProvince(db *pgxpool.Pool, filePath string) error {
	csvData, err := ReadCSV(filePath)
	if err != nil {
		return err
	}

	indexMap := headerIndexMap(csvData.Header)

	required := []string{"code", "name"}
	for _, col := range required {
		if _, ok := indexMap[col]; !ok {
			return fmt.Errorf("missing column: %s", col)
		}
	}

	ctx := context.Background()

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, row := range csvData.Rows {
		if isEmptyRow(row) {
			continue
		}

		data := ProvinceRow{
			Code: getCell(row, indexMap, "code"),
			Name: getCell(row, indexMap, "name"),
		}

		if err := upsertProvince(ctx, tx, &data); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func upsertProvince(ctx context.Context, tx pgx.Tx, row *ProvinceRow) error {
	query := `
		INSERT INTO province (
			code,
			name
		)
		VALUES ($1, $2)
		ON CONFLICT (code)
		DO UPDATE SET
			name = EXCLUDED.name
	`

	_, err := tx.Exec(ctx, query, row.Code, row.Name)
	return err
}
