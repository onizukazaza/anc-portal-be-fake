package utils

// ===================================================================
// Slice Helpers — ใช้กับ slice ทั่วไป (generic Go 1.18+)
// ===================================================================

// Contains ตรวจว่า slice มี element ที่ต้องการหรือไม่
//
//	utils.Contains([]string{"admin","user"}, "admin")  // => true
//	utils.Contains([]int{1,2,3}, 5)                    // => false
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Unique คืน slice ที่ไม่มี element ซ้ำ (รักษาลำดับเดิม)
//
//	utils.Unique([]string{"a","b","a","c"})  // => ["a","b","c"]
//	utils.Unique([]int{1,2,2,3,1})           // => [1,2,3]
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Map แปลง slice จาก type A เป็น type B ด้วย transform function
//
//	ids := utils.Map(users, func(u User) string { return u.ID })
func Map[A any, B any](slice []A, fn func(A) B) []B {
	result := make([]B, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Filter คืนเฉพาะ element ที่ผ่านเงื่อนไข
//
//	active := utils.Filter(users, func(u User) bool { return u.Status == "active" })
func Filter[T any](slice []T, fn func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}
