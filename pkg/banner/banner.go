package banner

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────
// ANSI color helpers (auto-disabled when NO_COLOR is set)
// ─────────────────────────────────────────────────────────────

var useColor = os.Getenv("NO_COLOR") == ""

func cyan(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[36m" + s + "\x1b[0m"
}

func green(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[32m" + s + "\x1b[0m"
}

func yellow(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[33m" + s + "\x1b[0m"
}

func bold(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[1m" + s + "\x1b[0m"
}

func dim(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}

func magenta(s string) string {
	if !useColor {
		return s
	}
	return "\x1b[35m" + s + "\x1b[0m"
}

// ─────────────────────────────────────────────────────────────
// Status icons
// ─────────────────────────────────────────────────────────────

func enabled(detail string) string  { return green("✔") + " " + detail }
func disabled(detail string) string { return dim("✗ " + detail) }

// ─────────────────────────────────────────────────────────────
// Row — single line in the banner
// ─────────────────────────────────────────────────────────────

// Row represents one info line inside the banner box.
type Row struct {
	Label string // e.g. "Database (main)"
	Value string // already formatted with enabled()/disabled()
}

// ─────────────────────────────────────────────────────────────
// Options — configures the banner
// ─────────────────────────────────────────────────────────────

// Options configures what the banner displays.
type Options struct {
	AppName  string // e.g. "ANC Portal API"
	Version  string // e.g. "1.0.0"
	Env      string // e.g. "LOCAL", "STAGING", "PRODUCTION"
	Port     int    // listen port (0 = omit)
	BootTime time.Duration
	Rows     []Row
}

// ─────────────────────────────────────────────────────────────
// Database helpers — convenience for the common DB pattern
// ─────────────────────────────────────────────────────────────

// DBRow creates a Row for a connected database.
func DBRow(label, dbName, host string, port int) Row {
	return Row{
		Label: label,
		Value: enabled(fmt.Sprintf("%s @ %s:%d", dbName, host, port)),
	}
}

// DBDisabledRow creates a Row for a database that is not configured.
func DBDisabledRow(label string) Row {
	return Row{Label: label, Value: disabled("not configured")}
}

// ─────────────────────────────────────────────────────────────
// Component helpers — common infrastructure rows
// ─────────────────────────────────────────────────────────────

func KafkaRow(isEnabled bool, brokers []string, topic string) Row {
	if !isEnabled {
		return Row{Label: "Kafka", Value: disabled("disabled")}
	}
	return Row{Label: "Kafka", Value: enabled(fmt.Sprintf("%s → %s", strings.Join(brokers, ","), topic))}
}

func RedisRow(isEnabled bool, host string, port int) Row {
	if !isEnabled {
		return Row{Label: "Redis", Value: disabled("disabled")}
	}
	return Row{Label: "Redis", Value: enabled(fmt.Sprintf("%s:%d", host, port))}
}

func OTelRow(isEnabled bool, exporterURL string) Row {
	if !isEnabled {
		return Row{Label: "OpenTelemetry", Value: disabled("disabled")}
	}
	return Row{Label: "OpenTelemetry", Value: enabled(exporterURL)}
}

func LocalCacheRow(isEnabled bool, maxSize int, ttl time.Duration) Row {
	if !isEnabled {
		return Row{Label: "Local Cache", Value: disabled("disabled")}
	}
	return Row{Label: "Local Cache", Value: enabled(fmt.Sprintf("%d items / %s TTL", maxSize, fmtDuration(ttl)))}
}

func RateLimitRow(isEnabled bool, max int, expiration time.Duration) Row {
	if !isEnabled {
		return Row{Label: "Rate Limit", Value: disabled("disabled")}
	}
	return Row{Label: "Rate Limit", Value: enabled(fmt.Sprintf("%d req / %s", max, fmtDuration(expiration)))}
}

func SwaggerRow(isEnabled bool, basePath string) Row {
	if !isEnabled {
		return Row{Label: "Swagger", Value: disabled("disabled")}
	}
	return Row{Label: "Swagger", Value: enabled(basePath + "/swagger")}
}

// ─────────────────────────────────────────────────────────────
// Runtime / Build helpers
// ─────────────────────────────────────────────────────────────

// GoRow shows Go version + OS/architecture.
func GoRow() Row {
	return Row{
		Label: "Go",
		Value: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

// HostRow shows hostname + PID for identifying the instance (useful in K8s).
func HostRow() Row {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return Row{
		Label: "Host / PID",
		Value: fmt.Sprintf("%s / %d", host, os.Getpid()),
	}
}

// BuildRow shows git commit + build time (injected via -ldflags at compile time).
// Pass empty strings when build info is not available.
func BuildRow(gitCommit, buildTime string) Row {
	if gitCommit == "" || gitCommit == "dev" {
		return Row{Label: "Build", Value: dim("dev (no build info)")}
	}
	return Row{
		Label: "Build",
		Value: fmt.Sprintf("%s (%s)", gitCommit, buildTime),
	}
}

// DBPoolRow shows connection pool config for a database.
func DBPoolRow(label string, maxConns, minConns int) Row {
	return Row{
		Label: label,
		Value: enabled(fmt.Sprintf("%d max / %d min conns", maxConns, minConns)),
	}
}

// ServerRow shows request timeout + body limit.
func ServerRow(timeout time.Duration, bodyLimitBytes int) Row {
	var bodyStr string
	switch {
	case bodyLimitBytes >= 1<<20:
		bodyStr = fmt.Sprintf("%dMB", bodyLimitBytes/(1<<20))
	case bodyLimitBytes >= 1<<10:
		bodyStr = fmt.Sprintf("%dKB", bodyLimitBytes/(1<<10))
	default:
		bodyStr = fmt.Sprintf("%dB", bodyLimitBytes)
	}
	return Row{
		Label: "Timeout / Body",
		Value: fmt.Sprintf("%s / %s limit", fmtDuration(timeout), bodyStr),
	}
}

// ─────────────────────────────────────────────────────────────
// Section separator + External DB helpers
// ─────────────────────────────────────────────────────────────

// SectionRow renders a labeled thin separator line inside the banner box.
// Use this to visually group rows (e.g. "External Databases").
func SectionRow(label string) Row {
	return Row{Label: "__section__", Value: label}
}

// ExtDBRow creates a Row for an external database with a distinct ↗ icon
// to visually distinguish it from the main database.
func ExtDBRow(name, dbName, host string, port int) Row {
	return Row{
		Label: "DB ↗ " + name,
		Value: magenta("⬡") + " " + fmt.Sprintf("%s @ %s:%d", dbName, host, port),
	}
}

// ─────────────────────────────────────────────────────────────
// Render — builds the box-art banner string
// ─────────────────────────────────────────────────────────────

const boxWidth = 58 // inner content width (between │ and │)

// Print renders the banner to stdout.
func Print(opts Options) {
	Fprint(os.Stdout, opts)
}

// Fprint renders the banner to any writer.
func Fprint(w io.Writer, opts Options) {
	fmt.Fprintln(w)

	// ── top border ──
	fmt.Fprintf(w, " %s\n", cyan("┌"+strings.Repeat("─", boxWidth)+"┐"))

	// ── title ──
	title := bold(opts.AppName)
	if opts.Version != "" {
		title += dim(" v" + opts.Version)
	}
	printCenter(w, title, visLen(opts.AppName)+visLen(" v"+opts.Version))

	// ── environment ──
	envLabel := yellow("[" + strings.ToUpper(opts.Env) + "]")
	envRaw := "[" + strings.ToUpper(opts.Env) + "]"
	printCenter(w, envLabel, len(envRaw))

	// ── separator ──
	fmt.Fprintf(w, " %s\n", cyan("├"+strings.Repeat("─", boxWidth)+"┤"))

	// ── port ──
	if opts.Port > 0 {
		printRow(w, "Port", bold(fmt.Sprintf(":%d", opts.Port)), len(fmt.Sprintf(":%d", opts.Port)))
	}

	// ── rows ──
	for _, r := range opts.Rows {
		if r.Label == "__section__" {
			printSection(w, r.Value)
			continue
		}
		printRow(w, r.Label, r.Value, visLen(r.Value))
	}

	// ── footer ──
	fmt.Fprintf(w, " %s\n", cyan("├"+strings.Repeat("─", boxWidth)+"┤"))
	bootStr := fmt.Sprintf("Boot time: %s", opts.BootTime.Round(time.Millisecond))
	printCenter(w, bold(bootStr), len(bootStr))

	// ── bottom border ──
	fmt.Fprintf(w, " %s\n", cyan("└"+strings.Repeat("─", boxWidth)+"┘"))
	fmt.Fprintln(w)
}

// ─────────────────────────────────────────────────────────────
// Internal rendering helpers
// ─────────────────────────────────────────────────────────────

const dotChar = "·"

func printSection(w io.Writer, label string) {
	text := " " + label + " "
	rawLen := len(text)
	leftDash := 3
	rightDash := boxWidth - leftDash - rawLen
	if rightDash < 1 {
		rightDash = 1
	}
	line := strings.Repeat("─", leftDash) + text + strings.Repeat("─", rightDash)
	fmt.Fprintf(w, " %s%s%s\n", cyan("│"), dim(line), cyan("│"))
}

func printCenter(w io.Writer, text string, rawLen int) {
	if rawLen > boxWidth {
		text = truncateVisible(text, boxWidth-1) + "…"
		rawLen = boxWidth
	}
	pad := boxWidth - rawLen
	left := pad / 2
	right := pad - left
	fmt.Fprintf(w, " %s%s%s%s%s\n", cyan("│"), strings.Repeat(" ", left), text, strings.Repeat(" ", right), cyan("│"))
}

func printRow(w io.Writer, label string, value string, valueRawLen int) {
	const labelWidth = 20
	labelLen := visLen(label)
	dots := labelWidth - labelLen
	if dots < 1 {
		dots = 1
	}
	prefix := "  " + label + " " + dim(strings.Repeat(dotChar, dots)) + " "
	prefixRaw := 2 + labelLen + 1 + dots + 1

	maxValueLen := boxWidth - prefixRaw
	if maxValueLen < 1 {
		maxValueLen = 1
	}
	if valueRawLen > maxValueLen {
		value = truncateVisible(value, maxValueLen-1) + "…"
		valueRawLen = maxValueLen
	}

	rightPad := boxWidth - prefixRaw - valueRawLen
	if rightPad < 0 {
		rightPad = 0
	}

	fmt.Fprintf(w, " %s%s%s%s%s\n", cyan("│"), prefix, value, strings.Repeat(" ", rightPad), cyan("│"))
}

// truncateVisible truncates a string to maxWidth visible characters,
// preserving ANSI escape sequences and appending nothing (caller adds ellipsis).
func truncateVisible(s string, maxWidth int) string {
	var b strings.Builder
	inEsc := false
	visible := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			b.WriteRune(r)
			continue
		}
		if inEsc {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if visible >= maxWidth {
			break
		}
		b.WriteRune(r)
		visible++
	}
	// Close any open ANSI sequence
	if useColor {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// visLen returns the visible length of a string (strips ANSI escape sequences).
func visLen(s string) int {
	inEsc := false
	n := 0
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
		n++
	}
	return n
}

// fmtDuration formats a duration in a human-friendly way: "5m", "1h30m", "30s".
func fmtDuration(d time.Duration) string {
	if d >= time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if d >= time.Minute {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
