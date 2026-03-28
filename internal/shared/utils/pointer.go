package utils

// ===================================================================
// Pointer Helpers — สร้าง pointer จาก value (Go ไม่มี &literal syntax)
// ===================================================================
//
// Go ไม่อนุญาตให้ทำ &"hello" หรือ &42 ตรง ๆ
// ต้องสร้างตัวแปรก่อนแล้วค่อย & เอา — functions เหล่านี้ช่วยให้กระชับขึ้น
//
// ใช้เยอะมากใน:
//   - Test setup (mock data ที่มี optional fields เป็น *string, *int)
//   - DTO mapping ที่ field เป็น pointer (nullable JSON)
//   - Database scan ที่ column nullable
//
// ตัวอย่าง:
//
//	user := domain.User{
//	    Name:  "admin",
//	    Email: utils.Ptr("admin@example.com"),  // *string
//	    Age:   utils.Ptr(30),                    // *int
//	}

// Ptr คืน pointer ของ value ใด ๆ (generic — ใช้ได้ทุก type)
//
//	name := utils.Ptr("admin")     // *string
//	age  := utils.Ptr(30)          // *int
//	flag := utils.Ptr(true)        // *bool
//	amt  := utils.Ptr(99.50)       // *float64
func Ptr[T any](v T) *T {
	return &v
}

// Deref คืนค่าจาก pointer — ถ้าเป็น nil จะ return zero value ของ type นั้น
//
//	var name *string = nil
//	utils.Deref(name)  // => ""
//
//	s := utils.Ptr("hello")
//	utils.Deref(s)     // => "hello"
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// DerefOr คืนค่าจาก pointer — ถ้าเป็น nil จะ return fallback value ที่กำหนด
//
//	var name *string = nil
//	utils.DerefOr(name, "unknown")  // => "unknown"
//
//	s := utils.Ptr("hello")
//	utils.DerefOr(s, "unknown")     // => "hello"
func DerefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
