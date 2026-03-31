package pagination

import (
	"strings"
	"testing"
)

var testAllowed = AllowedColumns{
	"created_at": true,
	"doc_no":     true,
	"amount":     true,
}

// ─── SQLClause (legacy — backward-compatible) ───

func TestSQLClauseWithAllowedSort(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Sort: "doc_no", Order: "asc"}
	got := SQLClause(req, "created_at", testAllowed)
	want := "ORDER BY doc_no ASC LIMIT 10 OFFSET 0"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestSQLClauseFallbackWhenSortNotInWhitelist(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Sort: "malicious_column", Order: "asc"}
	got := SQLClause(req, "created_at", testAllowed)
	want := "ORDER BY created_at ASC LIMIT 10 OFFSET 0"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestSQLClauseFallbackWhenSortEmpty(t *testing.T) {
	req := Request{Page: 2, Limit: 20, Sort: "", Order: "desc"}
	got := SQLClause(req, "created_at", testAllowed)
	want := "ORDER BY created_at DESC LIMIT 20 OFFSET 20"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestSQLClauseOffset(t *testing.T) {
	req := Request{Page: 3, Limit: 5, Sort: "amount", Order: "desc"}
	got := SQLClause(req, "created_at", testAllowed)
	want := "ORDER BY amount DESC LIMIT 5 OFFSET 10"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// ─── CountQuery (legacy) ───

func TestCountQueryWithWhere(t *testing.T) {
	got := CountQuery("quotations", "WHERE customer_id = $1")
	want := "SELECT count(*) FROM quotations WHERE customer_id = $1"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestCountQueryWithoutWhere(t *testing.T) {
	got := CountQuery("products", "")
	want := "SELECT count(*) FROM products"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// ─── Query Builder: DataSQL ───

func TestDataSQLBasic(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Order: "desc"}
	got := From("quotations").
		Select("id", "doc_no", "status").
		Where("customer_id = $1").
		Paginate(req, "created_at", testAllowed).
		DataSQL()

	assertContains(t, got, "SELECT id, doc_no, status, COUNT(*) OVER() AS total_count")
	assertContains(t, got, "FROM quotations")
	assertContains(t, got, "WHERE customer_id = $1")
	assertContains(t, got, "ORDER BY created_at DESC")
	assertContains(t, got, "LIMIT 10 OFFSET 0")
}

func TestDataSQLRejectsBadSort(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Sort: "DROP TABLE--", Order: "asc"}
	got := From("users").
		Select("id", "name").
		Paginate(req, "created_at", testAllowed).
		DataSQL()

	// injection ถูก block → ใช้ default sort แทน
	assertContains(t, got, "ORDER BY created_at ASC")
	assertNotContains(t, got, "DROP")
}

func TestDataSQLNoWhere(t *testing.T) {
	req := Request{Page: 2, Limit: 5, Order: "asc"}
	got := From("products").
		Select("id", "name").
		Paginate(req, "created_at", testAllowed).
		DataSQL()

	assertNotContains(t, got, "WHERE")
	assertContains(t, got, "LIMIT 5 OFFSET 5")
}

// ─── Query Builder: CountSQL ───

func TestCountSQL(t *testing.T) {
	got := From("quotations").
		Where("customer_id = $1").
		Paginate(Request{Page: 1, Limit: 10, Order: "desc"}, "created_at", testAllowed).
		CountSQL()

	want := "SELECT count(*) FROM quotations\nWHERE customer_id = $1"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestCountSQLNoWhere(t *testing.T) {
	got := From("products").
		Paginate(Request{Page: 1, Limit: 10, Order: "desc"}, "created_at", testAllowed).
		CountSQL()

	want := "SELECT count(*) FROM products"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// ─── Query Builder: Search ───

func TestDataSQLWithSearch(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Order: "desc", Search: "hello"}
	got := From("quotations").
		Select("id", "doc_no").
		Where("customer_id = $1", 1).
		Search("doc_no", "status").
		Paginate(req, "created_at", testAllowed).
		DataSQL()

	// search param = $2 (เพราะ where มี $1 แล้ว)
	assertContains(t, got, "(doc_no ILIKE $2 OR status ILIKE $2)")
}

func TestDataSQLSearchIgnoredWhenEmpty(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Order: "desc", Search: ""}
	got := From("quotations").
		Select("id").
		Where("customer_id = $1").
		Search("doc_no").
		Paginate(req, "created_at", testAllowed).
		DataSQL()

	assertNotContains(t, got, "ILIKE")
}

func TestHasSearchTrue(t *testing.T) {
	req := Request{Search: "test"}
	q := From("t").Search("col1").Paginate(req, "id", testAllowed)
	if !q.HasSearch() {
		t.Fatal("want HasSearch=true")
	}
}

func TestHasSearchFalseWhenNoKeyword(t *testing.T) {
	req := Request{Search: ""}
	q := From("t").Search("col1").Paginate(req, "id", testAllowed)
	if q.HasSearch() {
		t.Fatal("want HasSearch=false")
	}
}

func TestSearchPattern(t *testing.T) {
	req := Request{Search: "hello"}
	q := From("t").Paginate(req, "id", testAllowed)
	if got := q.SearchPattern(); got != "%hello%" {
		t.Fatalf("want %%hello%%, got %q", got)
	}
}

// ─── Query Builder: PlainSQL ───

func TestPlainSQLNoCountOver(t *testing.T) {
	req := Request{Page: 1, Limit: 10, Order: "desc"}
	got := From("quotations").
		Select("id", "doc_no").
		Where("customer_id = $1").
		Paginate(req, "created_at", testAllowed).
		PlainSQL()

	assertContains(t, got, "SELECT id, doc_no")
	assertNotContains(t, got, "COUNT(*) OVER()")
	assertContains(t, got, "WHERE customer_id = $1")
}

// ─── Helpers ───

func assertContains(t *testing.T, got, sub string) {
	t.Helper()
	if !strings.Contains(got, sub) {
		t.Fatalf("expected %q to contain %q", got, sub)
	}
}

func assertNotContains(t *testing.T, got, sub string) {
	t.Helper()
	if strings.Contains(got, sub) {
		t.Fatalf("expected %q NOT to contain %q", got, sub)
	}
}
