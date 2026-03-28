package cache

import "errors"

// ErrCacheMiss คือ error ที่บอกว่า key ไม่พบใน cache
// ใช้ errors.Is(err, cache.ErrCacheMiss) เพื่อตรวจสอบ
var ErrCacheMiss = errors.New("cache: key not found")
