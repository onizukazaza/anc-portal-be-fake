package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ===================================================================
// ID Generators — สร้าง unique ID สำหรับ entity ต่าง ๆ
// ===================================================================

// NewID สร้าง unique ID แบบ prefixed: "{prefix}-{timestamp}-{random}"
// เหมาะสำหรับ primary key ที่ต้องการ human-readable + sortable by time
//
//	utils.NewID("usr")  // => "usr-20260326150405-a1b2c3d4"
//	utils.NewID("qt")   // => "qt-20260326150405-e5f6a7b8"
//	utils.NewID("ord")  // => "ord-20260326150405-c9d0e1f2"
func NewID(prefix string) string {
	ts := time.Now().Format("20060102150405")
	rnd := randomHex(4)
	return fmt.Sprintf("%s-%s-%s", prefix, ts, rnd)
}

// NewShortID สร้าง short random hex ID (8 chars)
// เหมาะสำหรับ correlation ID, request ID, temp key
//
//	utils.NewShortID()  // => "a1b2c3d4"
func NewShortID() string {
	return randomHex(4)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback ถ้า crypto/rand ล้มเหลว (แทบไม่มีทาง)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
