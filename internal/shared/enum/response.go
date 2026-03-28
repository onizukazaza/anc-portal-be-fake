// Package enum เก็บค่าคงที่ (constants) ที่ใช้ร่วมกันระหว่าง modules
// ป้องกัน hardcode string กระจัดกระจาย — เปลี่ยนที่เดียวจบ
package enum

// ===================================================================
// Response Status — สถานะของ API response
// ===================================================================

const (
	// ResponseOK ใช้ใน API response เมื่อสำเร็จ
	ResponseOK = "OK"

	// ResponseError ใช้ใน API response เมื่อเกิด error
	ResponseError = "ERROR"
)
