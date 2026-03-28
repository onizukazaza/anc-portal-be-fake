package importer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InsurerInstallmentRow struct {
	InsurerCode      string
	InstallmentMonth int
	InterestRate     float64
	Status           string
}

func ImportInsurerInstallment(db *pgxpool.Pool, filePath string) error {
	csvData, err := ReadCSV(filePath)
	if err != nil {
		return err
	}

	requiredColumns := []string{
		"insurer_code",
		"installment_month",
		"interest_rate",
	}

	indexMap := headerIndexMap(csvData.Header)

	for _, col := range requiredColumns {
		if _, ok := indexMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	totalRows := 0
	successRows := 0

	for i, row := range csvData.Rows {
		lineNo := i + 2

		if isEmptyRow(row) {
			continue
		}

		data, err := mapInsurerInstallmentRow(row, indexMap)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		if err := upsertInsurerInstallment(ctx, tx, data); err != nil {
			return fmt.Errorf("line %d: upsert insurer_installment: %w", lineNo, err)
		}

		totalRows++
		successRows++
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("import insurer_installment success: total=%d success=%d\n", totalRows, successRows)
	return nil
}

func mapInsurerInstallmentRow(row []string, indexMap map[string]int) (*InsurerInstallmentRow, error) {
	insurerCode := getCell(row, indexMap, "insurer_code")
	monthText := getCell(row, indexMap, "installment_month")
	interestText := getCell(row, indexMap, "interest_rate")
	status := getCell(row, indexMap, "status")

	if insurerCode == "" {
		return nil, fmt.Errorf("insurer_code is required")
	}
	if monthText == "" {
		return nil, fmt.Errorf("installment_month is required")
	}
	if interestText == "" {
		return nil, fmt.Errorf("interest_rate is required")
	}

	month, err := strconv.Atoi(monthText)
	if err != nil {
		return nil, fmt.Errorf("invalid installment_month: %s", monthText)
	}

	interestRate, err := strconv.ParseFloat(interestText, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid interest_rate: %s", interestText)
	}

	if status == "" {
		status = "active"
	}

	return &InsurerInstallmentRow{
		InsurerCode:      insurerCode,
		InstallmentMonth: month,
		InterestRate:     interestRate,
		Status:           strings.ToLower(status),
	}, nil
}

func upsertInsurerInstallment(ctx context.Context, tx pgx.Tx, row *InsurerInstallmentRow) error {
	query := `
		INSERT INTO insurer_installment (
			insurer_code,
			installment_month,
			interest_rate,
			status
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (insurer_code, installment_month)
		DO UPDATE SET
			interest_rate = EXCLUDED.interest_rate,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := tx.Exec(ctx, query, row.InsurerCode, row.InstallmentMonth, row.InterestRate, row.Status)
	return err
}
