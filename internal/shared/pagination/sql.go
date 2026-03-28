package pagination

import (
	"fmt"
	"strings"
)

// AllowedColumns — whitelist คอลัมน์ที่อนุญาตให้ sort (ป้องกัน SQL injection)
type AllowedColumns map[string]bool

// ───────────────────────────────────────────────────────────────────
// Legacy helpers (backward-compatible)
// ───────────────────────────────────────────────────────────────────

// SQLClause สร้าง ORDER BY + LIMIT + OFFSET clause
func SQLClause(req Request, defaultSort string, allowed AllowedColumns) string {
	return safeSort(req, defaultSort, allowed) + " " + limitOffset(req)
}

// CountQuery สร้าง SELECT count(*) query
func CountQuery(table, baseWhere string) string {
	if baseWhere == "" {
		return fmt.Sprintf("SELECT count(*) FROM %s", table)
	}
	return fmt.Sprintf("SELECT count(*) FROM %s %s", table, baseWhere)
}

// ───────────────────────────────────────────────────────────────────
// Query — fluent SQL builder สำหรับ paginated queries
// ───────────────────────────────────────────────────────────────────
//
// ใช้ method-chain สร้าง query:
//
//	q := pagination.From("quotations").
//	    Select("id", "doc_no", "total_amount", "status", "created_at").
//	    Where("customer_id = $1").
//	    Search("doc_no", "status").
//	    Paginate(pg, "created_at", allowedSorts)
//
//	q.DataSQL()  → SELECT ... WITH count(*) OVER() ... ORDER BY ... LIMIT ...
//	q.CountSQL() → SELECT count(*) FROM quotations WHERE ...

// Query เก็บ state ของ paginated query ที่สร้างแบบ fluent
type Query struct {
	table      string
	columns    []string
	where      string
	searchCols []string
	req        Request
	defSort    string
	allowed    AllowedColumns
}

// From เริ่มสร้าง query จากชื่อ table
func From(table string) *Query {
	return &Query{table: table}
}

// Select กำหนด columns ที่ต้องการ SELECT
func (q *Query) Select(cols ...string) *Query {
	q.columns = cols
	return q
}

// Where กำหนด WHERE clause (ไม่ต้องใส่คำว่า "WHERE")
func (q *Query) Where(clause string) *Query {
	q.where = clause
	return q
}

// Search กำหนด columns สำหรับ ILIKE search —
// ถ้า Request.Search ว่างจะไม่เพิ่มเงื่อนไข
func (q *Query) Search(cols ...string) *Query {
	q.searchCols = cols
	return q
}

// Paginate กำหนด pagination, sort whitelist, default sort
func (q *Query) Paginate(req Request, defaultSort string, allowed AllowedColumns) *Query {
	q.req = req
	q.defSort = defaultSort
	q.allowed = allowed
	return q
}

// DataSQL สร้าง SELECT query พร้อม COUNT(*) OVER() ใน query เดียว
//
// ตัวอย่าง output:
//
//	SELECT id, doc_no, status, COUNT(*) OVER() AS total_count
//	FROM quotations
//	WHERE customer_id = $1
//	ORDER BY created_at DESC LIMIT 20 OFFSET 0
func (q *Query) DataSQL() string {
	cols := "*"
	if len(q.columns) > 0 {
		cols = strings.Join(q.columns, ", ")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "SELECT %s, COUNT(*) OVER() AS total_count\nFROM %s", cols, q.table)

	if where := q.fullWhere(); where != "" {
		fmt.Fprintf(&b, "\nWHERE %s", where)
	}

	fmt.Fprintf(&b, "\n%s %s", safeSort(q.req, q.defSort, q.allowed), limitOffset(q.req))
	return b.String()
}

// CountSQL สร้าง SELECT count(*) query (สำหรับกรณีที่ไม่ใช้ window function)
func (q *Query) CountSQL() string {
	if where := q.fullWhere(); where != "" {
		return fmt.Sprintf("SELECT count(*) FROM %s\nWHERE %s", q.table, where)
	}
	return fmt.Sprintf("SELECT count(*) FROM %s", q.table)
}

// PlainSQL สร้าง SELECT query ธรรมดา (ไม่มี COUNT OVER)
func (q *Query) PlainSQL() string {
	cols := "*"
	if len(q.columns) > 0 {
		cols = strings.Join(q.columns, ", ")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "SELECT %s\nFROM %s", cols, q.table)

	if where := q.fullWhere(); where != "" {
		fmt.Fprintf(&b, "\nWHERE %s", where)
	}

	fmt.Fprintf(&b, "\n%s %s", safeSort(q.req, q.defSort, q.allowed), limitOffset(q.req))
	return b.String()
}

// SearchParamIndex คืน parameter index ($N) สำหรับ search value
// ใช้เมื่อ Request.Search ไม่ว่าง — caller ต้องส่ง "%"+req.Search+"%" เป็น param ตัวนี้
//
// ตัวอย่าง: Where ใช้ $1 → SearchParamIndex(2) → search ใช้ $2
func (q *Query) SearchParamIndex(startIdx int) int {
	return startIdx
}

// HasSearch return true ถ้ามี search condition
func (q *Query) HasSearch() bool {
	return q.req.Search != "" && len(q.searchCols) > 0
}

// SearchPattern คืน "%keyword%" สำหรับส่งเป็น query parameter
func (q *Query) SearchPattern() string {
	return "%" + q.req.Search + "%"
}

// ───────────────────────────────────────────────────────────────────
// Internal helpers
// ───────────────────────────────────────────────────────────────────

func (q *Query) fullWhere() string {
	parts := make([]string, 0, 2)

	if q.where != "" {
		parts = append(parts, q.where)
	}

	if q.HasSearch() {
		parts = append(parts, q.searchClause())
	}

	return strings.Join(parts, " AND ")
}

// searchClause สร้าง (col1 ILIKE $N OR col2 ILIKE $N)
// ใช้ $N โดย N = จำนวน $-params ใน where + 1
func (q *Query) searchClause() string {
	paramIdx := strings.Count(q.where, "$") + 1
	placeholder := fmt.Sprintf("$%d", paramIdx)

	conditions := make([]string, len(q.searchCols))
	for i, col := range q.searchCols {
		conditions[i] = col + " ILIKE " + placeholder
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func safeSort(req Request, defaultSort string, allowed AllowedColumns) string {
	sort := defaultSort
	if req.Sort != "" && allowed[req.Sort] {
		sort = req.Sort
	}
	return fmt.Sprintf("ORDER BY %s %s", sort, strings.ToUpper(req.Order))
}

func limitOffset(req Request) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", req.Limit, req.Offset())
}
