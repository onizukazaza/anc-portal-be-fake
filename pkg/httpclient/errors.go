package httpclient

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/sony/gobreaker/v2"
)

// maxErrorBodyLen is the maximum number of bytes stored in ResponseError.Body.
// Prevents excessive memory usage and accidental sensitive data leakage in logs.
const maxErrorBodyLen = 1024

// ───────────────────────────────────────────────────────────────────
// ResponseError — error ที่มี HTTP status code ติดมาด้วย
// ───────────────────────────────────────────────────────────────────

// ResponseError เกิดเมื่อ server ตอบ 4xx หรือ 5xx
type ResponseError struct {
	StatusCode int
	Method     string
	URL        string
	Body       string // response body (truncated to maxErrorBodyLen)
	noRetry    bool   // สำหรับป้องกัน retry กรณี 4xx
}

func (e *ResponseError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("httpclient: %s %s returned %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("httpclient: %s %s returned %d", e.Method, e.URL, e.StatusCode)
}

// IsServerError ตรวจว่าเป็น 5xx หรือไม่
func (e *ResponseError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsClientError ตรวจว่าเป็น 4xx หรือไม่
func (e *ResponseError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// ───────────────────────────────────────────────────────────────────
// Helper — ใช้ภายใน package
// ───────────────────────────────────────────────────────────────────

func isResponseError(err error, target **ResponseError) bool {
	return errors.As(err, target)
}

// IsResponseError ตรวจว่า error เป็น *ResponseError หรือไม่ — ใช้ภายนอก package
func IsResponseError(err error) (*ResponseError, bool) {
	var re *ResponseError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// truncateBody trims body to maxErrorBodyLen bytes, appending "...(truncated)" if needed.
// It backs up to the nearest valid UTF-8 boundary to avoid splitting multi-byte characters.
func truncateBody(body string) string {
	if len(body) <= maxErrorBodyLen {
		return body
	}
	// Walk backwards from maxErrorBodyLen to find a valid rune boundary.
	truncated := body[:maxErrorBodyLen]
	for !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "...(truncated)"
}

// ───────────────────────────────────────────────────────────────────
// Circuit Breaker Errors
// ───────────────────────────────────────────────────────────────────

// IsCircuitOpen ตรวจว่า error เกิดจาก circuit breaker อยู่ในสถานะ open
func IsCircuitOpen(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)
}
