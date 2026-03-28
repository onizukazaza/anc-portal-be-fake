package httpclient

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateBody_Short(t *testing.T) {
	body := "short body"
	got := truncateBody(body)
	if got != body {
		t.Errorf("expected unchanged body, got %q", got)
	}
}

func TestTruncateBody_ExactlyMaxLen(t *testing.T) {
	body := strings.Repeat("x", maxErrorBodyLen)
	got := truncateBody(body)
	if got != body {
		t.Errorf("expected unchanged body at exact maxLen, got len=%d", len(got))
	}
}

func TestTruncateBody_LongASCII(t *testing.T) {
	body := strings.Repeat("a", maxErrorBodyLen+100)
	got := truncateBody(body)
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Errorf("expected truncated suffix, got %q", got[len(got)-20:])
	}
	// Content before suffix should be maxErrorBodyLen bytes
	content := strings.TrimSuffix(got, "...(truncated)")
	if len(content) != maxErrorBodyLen {
		t.Errorf("expected %d bytes before suffix, got %d", maxErrorBodyLen, len(content))
	}
}

func TestTruncateBody_UTF8Boundary(t *testing.T) {
	// Build a string that has a multi-byte rune crossing the maxErrorBodyLen boundary.
	// "中" is 3 bytes in UTF-8 (0xE4 0xB8 0xAD).
	// Fill up to maxErrorBodyLen-1 with ASCII, then add "中" — total = maxErrorBodyLen+2.
	prefix := strings.Repeat("a", maxErrorBodyLen-1)
	body := prefix + "中"
	if len(body) != maxErrorBodyLen+2 {
		t.Fatalf("test setup: expected len %d, got %d", maxErrorBodyLen+2, len(body))
	}

	got := truncateBody(body)
	content := strings.TrimSuffix(got, "...(truncated)")

	if !utf8.ValidString(content) {
		t.Errorf("truncated content is not valid UTF-8: %q", content)
	}
	// The "中" should be dropped (it crosses the boundary), leaving only the ASCII prefix.
	if content != prefix {
		t.Errorf("expected prefix of len %d, got len %d", len(prefix), len(content))
	}
}

func TestTruncateBody_Empty(t *testing.T) {
	got := truncateBody("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
