package pagination

import "testing"

// ─── Request.Defaults ───

func TestDefaultsSetsPageToOneWhenZero(t *testing.T) {
	r := Request{Page: 0, Limit: 10, Order: "asc"}
	r.Defaults()
	if r.Page != 1 {
		t.Fatalf("page: want 1, got %d", r.Page)
	}
}

func TestDefaultsSetsPageToOneWhenNegative(t *testing.T) {
	r := Request{Page: -5, Limit: 10, Order: "asc"}
	r.Defaults()
	if r.Page != 1 {
		t.Fatalf("page: want 1, got %d", r.Page)
	}
}

func TestDefaultsSetsLimitTo20WhenZero(t *testing.T) {
	r := Request{Page: 1, Limit: 0, Order: "asc"}
	r.Defaults()
	if r.Limit != 20 {
		t.Fatalf("limit: want 20, got %d", r.Limit)
	}
}

func TestDefaultsCapsLimitAt100(t *testing.T) {
	r := Request{Page: 1, Limit: 999, Order: "asc"}
	r.Defaults()
	if r.Limit != 100 {
		t.Fatalf("limit: want 100, got %d", r.Limit)
	}
}

func TestDefaultsSetsOrderToDescWhenInvalid(t *testing.T) {
	r := Request{Page: 1, Limit: 10, Order: "RANDOM"}
	r.Defaults()
	if r.Order != "desc" {
		t.Fatalf("order: want desc, got %s", r.Order)
	}
}

func TestDefaultsKeepsValidValues(t *testing.T) {
	r := Request{Page: 3, Limit: 50, Order: "asc"}
	r.Defaults()
	if r.Page != 3 || r.Limit != 50 || r.Order != "asc" {
		t.Fatalf("should keep valid values, got page=%d limit=%d order=%s", r.Page, r.Limit, r.Order)
	}
}

// ─── Request.Offset ───

func TestOffsetFirstPage(t *testing.T) {
	r := Request{Page: 1, Limit: 20}
	if got := r.Offset(); got != 0 {
		t.Fatalf("offset: want 0, got %d", got)
	}
}

func TestOffsetSecondPage(t *testing.T) {
	r := Request{Page: 2, Limit: 20}
	if got := r.Offset(); got != 20 {
		t.Fatalf("offset: want 20, got %d", got)
	}
}

func TestOffsetCustomLimit(t *testing.T) {
	r := Request{Page: 3, Limit: 10}
	if got := r.Offset(); got != 20 {
		t.Fatalf("offset: want 20, got %d", got)
	}
}

// ─── NewResponse ───

func TestNewResponseMetadata(t *testing.T) {
	items := []string{"a", "b", "c"}
	req := Request{Page: 2, Limit: 3}
	resp := NewResponse(items, 10, req)

	if resp.Total != 10 {
		t.Fatalf("total: want 10, got %d", resp.Total)
	}
	if resp.TotalPages != 4 { // expect 4 pages from 10 items at 3 per page
		t.Fatalf("totalPages: want 4, got %d", resp.TotalPages)
	}
	if !resp.HasNext {
		t.Fatal("hasNext: want true")
	}
	if !resp.HasPrev {
		t.Fatal("hasPrev: want true (page=2)")
	}
}

func TestNewResponseFirstPageNoPrev(t *testing.T) {
	resp := NewResponse([]string{"a"}, 5, Request{Page: 1, Limit: 10})
	if resp.HasPrev {
		t.Fatal("hasPrev: want false on first page")
	}
}

func TestNewResponseLastPageNoNext(t *testing.T) {
	resp := NewResponse([]string{"a"}, 3, Request{Page: 1, Limit: 10})
	if resp.HasNext {
		t.Fatal("hasNext: want false when all items fit in one page")
	}
}

func TestNewResponseNilItemsBecomesEmptySlice(t *testing.T) {
	resp := NewResponse[string](nil, 0, Request{Page: 1, Limit: 10})
	if resp.Items == nil {
		t.Fatal("items: want [], got nil")
	}
	if len(resp.Items) != 0 {
		t.Fatalf("items: want empty, got %d elements", len(resp.Items))
	}
}
