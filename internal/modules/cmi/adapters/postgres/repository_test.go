package postgres

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
)

// ─── Fake pgx.Row ────────────────────────────────────────────────

// fakeRow implements pgx.Row for testing scanCMIPolicy without a real DB.
type fakeRow struct {
	values []any
	err    error
}

func (f *fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(dest) != len(f.values) {
		return fmt.Errorf("scan: expected %d columns, got %d", len(f.values), len(dest))
	}
	for i, val := range f.values {
		switch d := dest[i].(type) {
		case *string:
			*d = val.(string)
		case *bool:
			*d = val.(bool)
		case **int:
			if val == nil {
				*d = nil
			} else {
				v := val.(int)
				*d = &v
			}
		case **string:
			if val == nil {
				*d = nil
			} else {
				v := val.(string)
				*d = &v
			}
		case *[]byte:
			switch v := val.(type) {
			case []byte:
				*d = v
			case nil:
				*d = nil
			}
		case *time.Time:
			*d = val.(time.Time)
		case *json.RawMessage:
			switch v := val.(type) {
			case []byte:
				*d = v
			case nil:
				*d = nil
			}
		default:
			return fmt.Errorf("scan: unsupported type at index %d: %T", i, dest[i])
		}
	}
	return nil
}

// ─── Helper ──────────────────────────────────────────────────────

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	testkit.MustNoError(t, err, "json.Marshal")
	return b
}

// buildSuccessRow returns a fakeRow with all 24 columns for a valid CMI policy.
func buildSuccessRow(t *testing.T) *fakeRow {
	t.Helper()

	insurerID := 42
	insurerName := "AIG"
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)

	motorJSON := mustJSON(t, domain.MotorInfo{
		Year: "2025", Brand: "Toyota", Model: "Camry",
	})
	assetJSON := mustJSON(t, domain.AssetInfo{
		ID: "asset-1", ChassisNo: "CH123", EngineNo: "EN456",
	})
	insuredJSON := mustJSON(t, domain.InsuredInfo{
		ID: "ins-1", FirstName: "John", LastName: "Doe",
	})
	policyJSON := mustJSON(t, domain.PolicyDate{
		CMI: &domain.DateRange{StartDate: "2026-01-01", EndDate: "2027-01-01"},
	})
	addressJSON := mustJSON(t, domain.AddressSet{})
	agentJSON := mustJSON(t, domain.AgentInfo{
		ID: "ag-1", FirstName: "Agent", LastName: "Smith",
	})
	quoteJSON := mustJSON(t, domain.QuoteInfo{
		ID: "q-1", JobID: "job-001",
	})

	return &fakeRow{
		values: []any{
			// 13 scalar fields
			"job-001",    // JobID
			"cmi_only",   // JobType
			"",           // JobRefID
			"quotations", // JobStatus
			"agent-001",  // AgentID
			"",           // CommissionTaxType
			false,        // WithHoldingTax
			false,        // TaxAllocation
			false,        // IsProblem
			insurerID,    // InsurerID (int → **int scan)
			insurerName,  // InsurerName (string → **string scan)
			"prod-001",   // ProductID

			// 10 JSON fields ([]byte)
			motorJSON,
			assetJSON,
			insuredJSON,
			policyJSON,
			addressJSON,
			agentJSON,
			[]byte(`[{"product_id":"p1"}]`), // Products
			[]byte(`[]`),                    // Payments
			[]byte(`[]`),                    // Documents
			[]byte(`[]`),                    // InsuredDocuments
			quoteJSON,

			// 2 timestamps
			now, // CreatedAt
			now, // UpdatedAt
		},
	}
}

// ─── Tests: scanCMIPolicy ────────────────────────────────────────

func TestScanCMIPolicy_Success(t *testing.T) {
	row := buildSuccessRow(t)

	pol, err := scanCMIPolicy(row)

	testkit.NoError(t, err)
	testkit.NotNil(t, pol, "policy")
	testkit.Equal(t, pol.JobID, "job-001", "JobID")
	testkit.Equal(t, pol.JobType, "cmi_only", "JobType")
	testkit.Equal(t, pol.JobStatus, "quotations", "JobStatus")
	testkit.Equal(t, pol.AgentID, "agent-001", "AgentID")
	testkit.Equal(t, pol.ProductID, "prod-001", "ProductID")

	// nested JSON structs
	testkit.NotNil(t, pol.Motor, "Motor")
	testkit.Equal(t, pol.Motor.Brand, "Toyota", "Motor.Brand")
	testkit.Equal(t, pol.Motor.Year, "2025", "Motor.Year")

	testkit.NotNil(t, pol.Asset, "Asset")
	testkit.Equal(t, pol.Asset.ChassisNo, "CH123", "Asset.ChassisNo")

	testkit.NotNil(t, pol.Insured, "Insured")
	testkit.Equal(t, pol.Insured.FirstName, "John", "Insured.FirstName")

	testkit.NotNil(t, pol.Policy, "Policy")
	testkit.NotNil(t, pol.Policy.CMI, "Policy.CMI")

	testkit.NotNil(t, pol.Agent, "Agent")
	testkit.Equal(t, pol.Agent.FirstName, "Agent", "Agent.FirstName")

	testkit.NotNil(t, pol.Quote, "Quote")
	testkit.Equal(t, pol.Quote.JobID, "job-001", "Quote.JobID")
}

func TestScanCMIPolicy_ScanError(t *testing.T) {
	scanErr := errors.New("no rows in result set")
	row := &fakeRow{err: scanErr}

	pol, err := scanCMIPolicy(row)

	testkit.Error(t, err)
	testkit.Nil(t, pol, "policy should be nil on error")
}

func TestScanCMIPolicy_NilJSONFields(t *testing.T) {
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)

	row := &fakeRow{
		values: []any{
			"job-002", "cmi_only", "", "draft", "", "", false, false, false,
			nil, nil, "",
			// all JSON fields nil
			([]byte)(nil), ([]byte)(nil), ([]byte)(nil), ([]byte)(nil),
			([]byte)(nil), ([]byte)(nil),
			([]byte)(nil), ([]byte)(nil), ([]byte)(nil), ([]byte)(nil),
			([]byte)(nil),
			now, now,
		},
	}

	pol, err := scanCMIPolicy(row)

	testkit.NoError(t, err)
	testkit.NotNil(t, pol, "policy")
	testkit.Equal(t, pol.JobID, "job-002", "JobID")
	testkit.Nil(t, pol.Motor, "Motor should be nil")
	testkit.Nil(t, pol.Insured, "Insured should be nil")
	testkit.Nil(t, pol.Agent, "Agent should be nil")
}

func TestScanCMIPolicy_InvalidMotorJSON(t *testing.T) {
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)

	row := &fakeRow{
		values: []any{
			"job-003", "cmi_only", "", "draft", "", "", false, false, false,
			nil, nil, "",
			[]byte(`{invalid json`), // bad motor JSON
			([]byte)(nil), ([]byte)(nil), ([]byte)(nil),
			([]byte)(nil), ([]byte)(nil),
			([]byte)(nil), ([]byte)(nil), ([]byte)(nil), ([]byte)(nil),
			([]byte)(nil),
			now, now,
		},
	}

	pol, err := scanCMIPolicy(row)

	testkit.Error(t, err)
	testkit.Nil(t, pol, "policy should be nil on bad JSON")
	testkit.Contains(t, err.Error(), "unmarshal motor", "error message")
}

// ─── Tests: unmarshalIfNotNil ────────────────────────────────────

func TestUnmarshalIfNotNil_NilData(t *testing.T) {
	var m domain.MotorInfo
	err := unmarshalIfNotNil(nil, &m)

	testkit.NoError(t, err)
	testkit.Equal(t, m.Brand, "", "should be zero value")
}

func TestUnmarshalIfNotNil_EmptyData(t *testing.T) {
	var m domain.MotorInfo
	err := unmarshalIfNotNil([]byte{}, &m)

	testkit.NoError(t, err)
}

func TestUnmarshalIfNotNil_ValidJSON(t *testing.T) {
	data := []byte(`{"brand":"Honda","year":"2024"}`)
	var m domain.MotorInfo
	err := unmarshalIfNotNil(data, &m)

	testkit.NoError(t, err)
	testkit.Equal(t, m.Brand, "Honda", "Brand")
	testkit.Equal(t, m.Year, "2024", "Year")
}

func TestUnmarshalIfNotNil_InvalidJSON(t *testing.T) {
	data := []byte(`{broken`)
	var m domain.MotorInfo
	err := unmarshalIfNotNil(data, &m)

	testkit.Error(t, err)
}

// ─── Tests: SQL Fragment Builders ────────────────────────────────

func TestBuildFindPolicyQuery_ContainsFragments(t *testing.T) {
	q := buildFindPolicyQuery()

	// verify key fragments are assembled
	fragments := []string{
		"j.id",                // job fields
		"j.job_type",          // job fields
		"jsonb_build_object",  // motor/asset/address etc.
		"json_build_object",   // insured
		"FROM job j",          // joins
		"LEFT JOIN quotation", // joins
		"LEFT JOIN insured",   // joins
		"WHERE j.id = $1",     // parameterized query
		"LIMIT 1",             // single row
	}

	for _, frag := range fragments {
		testkit.True(t, strings.Contains(q, frag),
			fmt.Sprintf("query should contain %q", frag))
	}
}

func TestSQLFragments_NotEmpty(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
	}{
		{"sqlSelectJobFields", sqlSelectJobFields},
		{"sqlMotorInfo", sqlMotorInfo},
		{"sqlAssetInfo", sqlAssetInfo},
		{"sqlInsured", sqlInsured},
		{"sqlPolicyDates", sqlPolicyDates},
		{"sqlAddressSet", sqlAddressSet},
		{"sqlAgentInfo", sqlAgentInfo},
		{"sqlProducts", sqlProducts},
		{"sqlPayments", sqlPayments},
		{"sqlInsuranceDocs", sqlInsuranceDocs},
		{"sqlInsuredDocs", sqlInsuredDocs},
		{"sqlQuoteInfo", sqlQuoteInfo},
		{"sqlFromJoins", sqlFromJoins},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql := tc.fn()
			testkit.True(t, len(strings.TrimSpace(sql)) > 0,
				fmt.Sprintf("%s should not be empty", tc.name))
		})
	}
}
