package utils

import "strings"

// ===================================================================
// String Helpers
// ===================================================================

// TrimLower trim whitespace แล้วแปลงเป็น lowercase
// ใช้บ่อยมากใน normalization: username, email, search term
//
//	utils.TrimLower("  Admin ")  // => "admin"
func TrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Truncate ตัดข้อความให้ไม่เกิน maxLen — ถ้าเกินจะเติม "..." ต่อท้าย
// เหมาะสำหรับ log message, preview text
//
//	utils.Truncate("Hello World", 5)  // => "Hello..."
//	utils.Truncate("Hi", 10)          // => "Hi"
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// DefaultIfEmpty คืน fallback ถ้า string ว่าง (หลัง trim)
//
//	utils.DefaultIfEmpty("", "N/A")       // => "N/A"
//	utils.DefaultIfEmpty("  ", "N/A")     // => "N/A"
//	utils.DefaultIfEmpty("hello", "N/A")  // => "hello"
func DefaultIfEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
