package banner

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestPrint_APIBanner(t *testing.T) {
	// Disable color for deterministic output.
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "ANC Portal API",
		Version:  "1.0.0",
		Env:      "local",
		Port:     20000,
		BootTime: 123 * time.Millisecond,
		Rows: []Row{
			DBRow("Database (main)", "anc_portal", "localhost", 5432),
			DBRow("Database (mpk)", "meprakun_db", "10.0.0.5", 5432),
			KafkaRow(true, []string{"localhost:9092"}, "anc-topic"),
			RedisRow(true, "localhost", 6379),
			OTelRow(false, ""),
			LocalCacheRow(true, 10000, 5*time.Minute),
			RateLimitRow(true, 100, time.Minute),
			SwaggerRow(true, "/v1"),
		},
	})

	output := buf.String()

	// Verify box borders
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┘") {
		t.Error("missing box borders")
	}

	// Verify title
	if !strings.Contains(output, "ANC Portal API") {
		t.Error("missing app name")
	}

	// Verify environment
	if !strings.Contains(output, "[LOCAL]") {
		t.Error("missing environment label")
	}

	// Verify port
	if !strings.Contains(output, ":20000") {
		t.Error("missing port")
	}

	// Verify databases with names
	if !strings.Contains(output, "Database (main)") {
		t.Error("missing main database row")
	}
	if !strings.Contains(output, "anc_portal") {
		t.Error("missing main database name")
	}
	if !strings.Contains(output, "Database (mpk)") {
		t.Error("missing external database row")
	}
	if !strings.Contains(output, "meprakun_db") {
		t.Error("missing external database name")
	}

	// Verify components
	for _, want := range []string{"Kafka", "Redis", "OpenTelemetry", "Local Cache", "Rate Limit", "Swagger"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing row: %s", want)
		}
	}

	// Verify disabled indicator
	if !strings.Contains(output, "disabled") {
		t.Error("missing disabled indicator for OTel")
	}

	// Verify boot time
	if !strings.Contains(output, "Boot time") {
		t.Error("missing boot time")
	}
}

func TestPrint_WorkerBanner(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "ANC Portal Worker",
		Version:  "1.0.0",
		Env:      "staging",
		BootTime: 50 * time.Millisecond,
		Rows: []Row{
			KafkaRow(true, []string{"kafka-1:9092", "kafka-2:9092"}, "events"),
			{Label: "Group ID", Value: "worker-group"},
			{Label: "DLQ Topic", Value: "events-dlq"},
		},
	})

	output := buf.String()

	if !strings.Contains(output, "ANC Portal Worker") {
		t.Error("missing worker app name")
	}
	if !strings.Contains(output, "[STAGING]") {
		t.Error("missing staging env")
	}
	if !strings.Contains(output, "kafka-1:9092") {
		t.Error("missing kafka brokers")
	}
	if !strings.Contains(output, "Group ID") {
		t.Error("missing group ID row")
	}
}

func TestPrint_DisabledComponents(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "Test App",
		Version:  "0.1.0",
		Env:      "production",
		BootTime: 10 * time.Millisecond,
		Rows: []Row{
			DBRow("Database (main)", "prod_db", "db.prod", 5432),
			KafkaRow(false, nil, ""),
			RedisRow(false, "", 0),
			OTelRow(false, ""),
			LocalCacheRow(false, 0, 0),
			RateLimitRow(false, 0, 0),
			SwaggerRow(false, ""),
		},
	})

	output := buf.String()

	// Count disabled indicators
	count := strings.Count(output, "disabled")
	if count < 5 {
		t.Errorf("expected at least 5 disabled rows, got %d", count)
	}
}

func TestDBDisabledRow(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	row := DBDisabledRow("Database (ext)")
	if row.Label != "Database (ext)" {
		t.Errorf("unexpected label: %s", row.Label)
	}
	if !strings.Contains(row.Value, "not configured") {
		t.Errorf("expected 'not configured', got: %s", row.Value)
	}
}

func TestVisLen(t *testing.T) {
	if visLen("hello") != 5 {
		t.Error("plain string length wrong")
	}
	if visLen("\x1b[32mhello\x1b[0m") != 5 {
		t.Error("colored string visible length wrong")
	}
	if visLen("") != 0 {
		t.Error("empty string length wrong")
	}
}

// ─────────────────────────────────────────────────────────────
// Alignment & Frame Integrity Tests
// ─────────────────────────────────────────────────────────────

// checkAlignment verifies every line between ┌ and └ has exactly boxWidth
// visible characters between the left │ and right │, ensuring the frame
// never breaks regardless of content width.
func checkAlignment(t *testing.T, output string) {
	t.Helper()
	lines := strings.Split(output, "\n")

	// Expected visible width for border lines: " ┌" + 58×"─" + "┐" = 1+1+58+1 = 61
	// Expected visible width for content lines: " │" + 58 chars + "│" = 1+1+58+1 = 61
	expectedWidth := 1 + 1 + boxWidth + 1 // space + left-border + content + right-border

	for i, line := range lines {
		if line == "" {
			continue // skip empty lines (before/after banner)
		}

		vis := visibleWidth(line)
		if vis != expectedWidth {
			t.Errorf("line %d: visible width = %d, want %d\n  raw: %q", i+1, vis, expectedWidth, line)
		}

		// Verify the line has proper border characters
		trimmed := strings.TrimSpace(stripANSI(line))
		if len(trimmed) == 0 {
			continue
		}
		first, _ := utf8.DecodeRuneInString(trimmed)
		last, _ := utf8.DecodeLastRuneInString(trimmed)

		validFirstChars := map[rune]bool{'┌': true, '│': true, '├': true, '└': true}
		validLastChars := map[rune]bool{'┐': true, '│': true, '┤': true, '┘': true}

		if !validFirstChars[first] {
			t.Errorf("line %d: unexpected first border char %q", i+1, string(first))
		}
		if !validLastChars[last] {
			t.Errorf("line %d: unexpected last border char %q", i+1, string(last))
		}
	}
}

// visibleWidth calculates the visible character width of a line (stripping ANSI).
func visibleWidth(s string) int {
	return visLen(s)
}

// stripANSI removes all ANSI escape sequences from a string.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestAlignment_NormalAPI(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "ANC Portal API",
		Version:  "1.0.0",
		Env:      "local",
		Port:     20000,
		BootTime: 234 * time.Millisecond,
		Rows: []Row{
			DBRow("Database (main)", "anc_portal", "localhost", 5432),
			DBRow("Database (mpk)", "meprakun_db", "10.0.0.5", 5432),
			KafkaRow(true, []string{"localhost:9092"}, "anc-topic"),
			RedisRow(true, "localhost", 6379),
			OTelRow(false, ""),
			LocalCacheRow(true, 10000, 5*time.Minute),
			RateLimitRow(true, 100, time.Minute),
			SwaggerRow(true, "/v1"),
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_LongValues(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "My Super Long Application Name Here",
		Version:  "99.99.99",
		Env:      "production",
		Port:     443,
		BootTime: 5*time.Second + 678*time.Millisecond,
		Rows: []Row{
			DBRow("Database (main)", "very_long_database_name_here", "db-host.internal.company.com", 5432),
			KafkaRow(true, []string{"kafka-broker-1.company.com:9092", "kafka-broker-2.company.com:9092"}, "my-very-long-topic-name"),
			RedisRow(true, "redis-cache.internal.company.com", 6379),
			{Label: "Very Long Label Name", Value: "some-long-value-that-might-overflow-the-box"},
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_ShortValues(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "X",
		Version:  "0",
		Env:      "d",
		Port:     1,
		BootTime: time.Millisecond,
		Rows: []Row{
			{Label: "A", Value: "B"},
			{Label: "DB", Value: "✔ x"},
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_NoRows(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "Empty App",
		Version:  "1.0.0",
		Env:      "test",
		BootTime: time.Millisecond,
		Rows:     nil,
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_NoPort(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "Worker App",
		Version:  "2.0.0",
		Env:      "staging",
		Port:     0, // no port
		BootTime: 100 * time.Millisecond,
		Rows: []Row{
			KafkaRow(true, []string{"localhost:9092"}, "events"),
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_AllDisabled(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "Minimal App",
		Version:  "0.0.1",
		Env:      "local",
		Port:     8080,
		BootTime: 10 * time.Millisecond,
		Rows: []Row{
			DBDisabledRow("Database (main)"),
			KafkaRow(false, nil, ""),
			RedisRow(false, "", 0),
			OTelRow(false, ""),
			LocalCacheRow(false, 0, 0),
			RateLimitRow(false, 0, 0),
			SwaggerRow(false, ""),
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_WithColor(t *testing.T) {
	// Test WITH color enabled — ensure ANSI codes don't break alignment.
	useColor = true

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "ANC Portal API",
		Version:  "1.0.0",
		Env:      "local",
		Port:     20000,
		BootTime: 123 * time.Millisecond,
		Rows: []Row{
			DBRow("Database (main)", "anc_portal", "localhost", 5432),
			KafkaRow(true, []string{"localhost:9092"}, "anc-topic"),
			RedisRow(false, "", 0),
			OTelRow(true, "localhost:4318"),
		},
	})
	checkAlignment(t, buf.String())
}

func TestAlignment_ExtremeOverflow(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // 58 chars (== boxWidth)
		Version:  "999.999.999",
		Env:      "very-long-environment-name",
		Port:     65535,
		BootTime: 999 * time.Hour,
		Rows: []Row{
			{Label: "XXXXXXXXXXXXXXXXXXXX", Value: strings.Repeat("Y", 40)}, // label=20, value=40 → exceeds box
			{Label: strings.Repeat("Z", 30), Value: strings.Repeat("W", 30)},
		},
	})

	// For overflow cases, we don't expect perfect alignment but should not panic.
	output := buf.String()
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┘") {
		t.Error("missing box borders on extreme overflow")
	}
	// Should still have some content
	if len(output) < 100 {
		t.Error("output too short for extreme case")
	}
}

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Minute, "5m"},
		{time.Minute, "1m"},
		{90 * time.Second, "1m30s"},
		{time.Hour, "1h"},
		{90 * time.Minute, "1h30m"},
		{30 * time.Second, "30s"},
		{0, "0s"},
	}
	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// ─── Tests for new helper rows ──────────────────────────────

func TestGoRow(t *testing.T) {
	r := GoRow()
	if r.Label != "Go" {
		t.Errorf("GoRow label = %q, want Go", r.Label)
	}
	if !strings.Contains(r.Value, "go1.") {
		t.Errorf("GoRow value = %q, want to contain go version", r.Value)
	}
}

func TestHostRow(t *testing.T) {
	r := HostRow()
	if r.Label != "Host / PID" {
		t.Errorf("HostRow label = %q", r.Label)
	}
	if !strings.Contains(r.Value, "/") {
		t.Errorf("HostRow value = %q, want host / pid format", r.Value)
	}
}

func TestBuildRow_Dev(t *testing.T) {
	r := BuildRow("dev", "unknown")
	if !strings.Contains(r.Value, "dev") {
		t.Errorf("BuildRow (dev) value = %q, want 'dev'", r.Value)
	}
}

func TestBuildRow_Real(t *testing.T) {
	r := BuildRow("a1b2c3d", "2026-03-28T10:30:00Z")
	if !strings.Contains(r.Value, "a1b2c3d") {
		t.Errorf("BuildRow value = %q, want commit hash", r.Value)
	}
	if !strings.Contains(r.Value, "2026-03-28") {
		t.Errorf("BuildRow value = %q, want build time", r.Value)
	}
}

func TestDBPoolRow(t *testing.T) {
	r := DBPoolRow("Pool (main)", 10, 2)
	if !strings.Contains(r.Value, "10 max") || !strings.Contains(r.Value, "2 min") {
		t.Errorf("DBPoolRow value = %q", r.Value)
	}
}

func TestServerRow(t *testing.T) {
	r := ServerRow(30*time.Second, 4*1024*1024)
	if !strings.Contains(r.Value, "30s") {
		t.Errorf("ServerRow value = %q, want timeout", r.Value)
	}
	if !strings.Contains(r.Value, "4MB") {
		t.Errorf("ServerRow value = %q, want body limit", r.Value)
	}
}

func TestServerRow_KB(t *testing.T) {
	r := ServerRow(time.Minute, 512*1024)
	if !strings.Contains(r.Value, "512KB") {
		t.Errorf("ServerRow KB value = %q", r.Value)
	}
}

func TestExtDBRow(t *testing.T) {
	r := ExtDBRow("partner", "partner_db", "10.0.0.5", 5432)
	if !strings.Contains(r.Label, "↗") {
		t.Errorf("ExtDBRow label = %q, want ↗ icon", r.Label)
	}
	if !strings.Contains(r.Label, "partner") {
		t.Errorf("ExtDBRow label = %q, want name", r.Label)
	}
	if !strings.Contains(r.Value, "partner_db") {
		t.Errorf("ExtDBRow value = %q, want db name", r.Value)
	}
}

func TestSectionRow(t *testing.T) {
	r := SectionRow("External Databases")
	if r.Label != "__section__" {
		t.Errorf("SectionRow label = %q, want __section__", r.Label)
	}
	if r.Value != "External Databases" {
		t.Errorf("SectionRow value = %q", r.Value)
	}
}

func TestMockRow_Disabled(t *testing.T) {
	r := MockRow(false, 0, 0)
	if r.Label != "Mock" {
		t.Errorf("MockRow label = %q, want Mock", r.Label)
	}
	if !strings.Contains(r.Value, "disabled") {
		t.Errorf("MockRow disabled value = %q, want 'disabled'", r.Value)
	}
}

func TestMockRow_AllActive(t *testing.T) {
	r := MockRow(true, 6, 6)
	if !strings.Contains(r.Value, "6/6 routes") {
		t.Errorf("MockRow all-active value = %q, want '6/6 routes'", r.Value)
	}
	if !strings.Contains(r.Value, "✔") {
		t.Errorf("MockRow all-active value = %q, want ✔ icon", r.Value)
	}
}

func TestMockRow_Partial(t *testing.T) {
	r := MockRow(true, 4, 6)
	if !strings.Contains(r.Value, "4/6 routes") {
		t.Errorf("MockRow partial value = %q, want '4/6 routes'", r.Value)
	}
	if !strings.Contains(r.Value, "⚠") {
		t.Errorf("MockRow partial value = %q, want ⚠ icon", r.Value)
	}
}

func TestAlignment_WithSections(t *testing.T) {
	useColor = false
	defer func() { useColor = true }()

	var buf bytes.Buffer
	Fprint(&buf, Options{
		AppName:  "ANC Portal API",
		Version:  "1.0.0",
		Env:      "local",
		Port:     20000,
		BootTime: 123 * time.Millisecond,
		Rows: []Row{
			GoRow(),
			HostRow(),
			BuildRow("a1b2c3d", "2026-03-28T10:30:00Z"),
			DBRow("Database (main)", "anc_portal", "localhost", 5432),
			DBPoolRow("  Pool (main)", 10, 2),
			SectionRow("External Databases"),
			ExtDBRow("partner", "partner_db", "10.0.0.5", 5432),
			ExtDBRow("legacy", "old_system", "10.0.0.6", 3306),
			KafkaRow(true, []string{"localhost:9092"}, "anc-topic"),
			RedisRow(true, "localhost", 6379),
			OTelRow(true, "localhost:4318"),
			LocalCacheRow(true, 10000, 5*time.Minute),
			RateLimitRow(true, 100, time.Minute),
			SwaggerRow(true, "/v1"),
			ServerRow(30*time.Second, 4*1024*1024),
		},
	})
	checkAlignment(t, buf.String())
}
