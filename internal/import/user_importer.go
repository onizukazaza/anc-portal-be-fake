package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRow struct {
	ID                    string
	Role                  []byte
	CitizenID             string
	PassportID            string
	PrefixID              *int32
	FirstName             string
	LastName              string
	NickName              string
	Gender                string
	BirthDate             *time.Time
	PhoneNumber           string
	Email                 string
	IsVerifiedEmail       bool
	VerifiedEmailDatetime *time.Time
	Password              string
	ProfileImageURL       string
	Status                string
	TermsAndConditionsID  *string
	ANCMarketerID         string
	Marketer              []byte
}

func ImportUser(db *pgxpool.Pool, filePath string) error {
	csvData, err := ReadCSV(filePath)
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	indexMap := headerIndexMap(csvData.Header)

	requiredColumns := []string{"id", "role"}
	for _, col := range requiredColumns {
		if _, ok := indexMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	totalRows := 0
	successRows := 0

	for i, row := range csvData.Rows {
		lineNo := i + 2

		if isEmptyRow(row) {
			continue
		}

		totalRows++

		data, err := mapUserRow(row, indexMap)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		if err := upsertUser(ctx, tx, data); err != nil {
			return fmt.Errorf("line %d: upsert user: %w", lineNo, err)
		}

		successRows++
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	fmt.Printf("import users success: total=%d success=%d\n", totalRows, successRows)
	return nil
}

func mapUserRow(row []string, indexMap map[string]int) (*UserRow, error) {
	id := strings.TrimSpace(getCell(row, indexMap, "id"))
	roleText := strings.TrimSpace(getCell(row, indexMap, "role"))
	citizenID := strings.TrimSpace(getCell(row, indexMap, "citizen_id"))
	passportID := strings.TrimSpace(getCell(row, indexMap, "passport_id"))
	prefixIDText := strings.TrimSpace(getCell(row, indexMap, "prefix_id"))
	firstName := strings.TrimSpace(getCell(row, indexMap, "first_name"))
	lastName := strings.TrimSpace(getCell(row, indexMap, "last_name"))
	nickName := strings.TrimSpace(getCell(row, indexMap, "nick_name"))
	gender := strings.TrimSpace(getCell(row, indexMap, "gender"))
	birthDateText := strings.TrimSpace(getCell(row, indexMap, "birth_date"))
	phoneNumber := strings.TrimSpace(getCell(row, indexMap, "phone_number"))
	email := strings.TrimSpace(getCell(row, indexMap, "email"))
	isVerifiedEmailText := strings.TrimSpace(getCell(row, indexMap, "is_verified_email"))
	verifiedEmailDatetimeText := strings.TrimSpace(getCell(row, indexMap, "verified_email_datetime"))
	password := strings.TrimSpace(getCell(row, indexMap, "password"))
	profileImageURL := strings.TrimSpace(getCell(row, indexMap, "profile_image_url"))
	status := strings.TrimSpace(getCell(row, indexMap, "status"))
	termsAndConditionsID := strings.TrimSpace(getCell(row, indexMap, "terms_and_conditions_id"))
	ancMarketerID := strings.TrimSpace(getCell(row, indexMap, "anc_marketer_id"))
	marketerText := strings.TrimSpace(getCell(row, indexMap, "marketer"))

	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if roleText == "" {
		return nil, fmt.Errorf("role is required")
	}

	roleJSON, err := normalizeJSON(roleText)
	if err != nil {
		return nil, fmt.Errorf("invalid role json: %w", err)
	}

	var prefixID *int32
	if prefixIDText != "" {
		v, pErr := parseInt32(prefixIDText)
		if pErr != nil {
			return nil, fmt.Errorf("invalid prefix_id: %s", prefixIDText)
		}
		prefixID = &v
	}

	var birthDate *time.Time
	if birthDateText != "" {
		t, pErr := time.Parse("2006-01-02", birthDateText)
		if pErr != nil {
			return nil, fmt.Errorf("invalid birth_date, expected YYYY-MM-DD: %s", birthDateText)
		}
		birthDate = &t
	}

	isVerifiedEmail := false
	if isVerifiedEmailText != "" {
		v, pErr := parseBool(isVerifiedEmailText)
		if pErr != nil {
			return nil, fmt.Errorf("invalid is_verified_email: %s", isVerifiedEmailText)
		}
		isVerifiedEmail = v
	}

	var verifiedEmailDatetime *time.Time
	if verifiedEmailDatetimeText != "" {
		t, pErr := parseDateTime(verifiedEmailDatetimeText)
		if pErr != nil {
			return nil, fmt.Errorf("invalid verified_email_datetime: %s", verifiedEmailDatetimeText)
		}
		verifiedEmailDatetime = &t
	}

	if status == "" {
		status = "active"
	}
	status = strings.ToLower(status)
	if status != "active" && status != "inactive" {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	var termsAndConditionsIDPtr *string
	if termsAndConditionsID != "" {
		termsAndConditionsIDPtr = &termsAndConditionsID
	}

	var marketerJSON []byte
	if marketerText != "" {
		marketerJSON, err = normalizeJSON(marketerText)
		if err != nil {
			return nil, fmt.Errorf("invalid marketer json: %w", err)
		}
	}

	return &UserRow{
		ID:                    id,
		Role:                  roleJSON,
		CitizenID:             citizenID,
		PassportID:            passportID,
		PrefixID:              prefixID,
		FirstName:             firstName,
		LastName:              lastName,
		NickName:              nickName,
		Gender:                gender,
		BirthDate:             birthDate,
		PhoneNumber:           phoneNumber,
		Email:                 email,
		IsVerifiedEmail:       isVerifiedEmail,
		VerifiedEmailDatetime: verifiedEmailDatetime,
		Password:              password,
		ProfileImageURL:       profileImageURL,
		Status:                status,
		TermsAndConditionsID:  termsAndConditionsIDPtr,
		ANCMarketerID:         ancMarketerID,
		Marketer:              marketerJSON,
	}, nil
}

func upsertUser(ctx context.Context, tx pgx.Tx, row *UserRow) error {
	query := `
		INSERT INTO users (
			id,
			role,
			citizen_id,
			passport_id,
			prefix_id,
			first_name,
			last_name,
			nick_name,
			gender,
			birth_date,
			phone_number,
			email,
			is_verified_email,
			verified_email_datetime,
			password,
			profile_image_url,
			status,
			terms_and_conditions_id,
			anc_marketer_id,
			marketer
		)
		VALUES (
			$1, $2::jsonb, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18::uuid, $19, $20::jsonb
		)
		ON CONFLICT (id)
		DO UPDATE SET
			role = EXCLUDED.role,
			citizen_id = EXCLUDED.citizen_id,
			passport_id = EXCLUDED.passport_id,
			prefix_id = EXCLUDED.prefix_id,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			nick_name = EXCLUDED.nick_name,
			gender = EXCLUDED.gender,
			birth_date = EXCLUDED.birth_date,
			phone_number = EXCLUDED.phone_number,
			email = EXCLUDED.email,
			is_verified_email = EXCLUDED.is_verified_email,
			verified_email_datetime = EXCLUDED.verified_email_datetime,
			password = EXCLUDED.password,
			profile_image_url = EXCLUDED.profile_image_url,
			status = EXCLUDED.status,
			terms_and_conditions_id = EXCLUDED.terms_and_conditions_id,
			anc_marketer_id = EXCLUDED.anc_marketer_id,
			marketer = EXCLUDED.marketer,
			updated_datetime = NOW()
	`

	_, err := tx.Exec(
		ctx,
		query,
		row.ID,
		row.Role,
		nullIfEmptyImport(row.CitizenID),
		nullIfEmptyImport(row.PassportID),
		row.PrefixID,
		nullIfEmptyImport(row.FirstName),
		nullIfEmptyImport(row.LastName),
		nullIfEmptyImport(row.NickName),
		nullIfEmptyImport(row.Gender),
		row.BirthDate,
		nullIfEmptyImport(row.PhoneNumber),
		nullIfEmptyImport(row.Email),
		row.IsVerifiedEmail,
		row.VerifiedEmailDatetime,
		nullIfEmptyImport(row.Password),
		nullIfEmptyImport(row.ProfileImageURL),
		row.Status,
		row.TermsAndConditionsID,
		nullIfEmptyImport(row.ANCMarketerID),
		nullJSON(row.Marketer),
	)
	if err != nil {
		return fmt.Errorf("exec upsert users: %w", err)
	}

	return nil
}

// ------------------------------
// Helper functions
// ------------------------------

func normalizeJSON(input string) ([]byte, error) {
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return nil, err
	}
	return []byte(input), nil
}

func parseInt32(s string) (int32, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return 0, err
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, fmt.Errorf("value %d out of int32 range", v)
	}
	return int32(v), nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean")
	}
}

func parseDateTime(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported datetime format")
}

func nullIfEmptyImport(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
