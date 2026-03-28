package utils

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ===================================================================
// JSON Helpers — สำหรับ debug, logging, testing
// ===================================================================

// PrettyJSON แปลง value เป็น JSON แบบ indent สวย ๆ สำหรับ debug/log
//
//	fmt.Println(utils.PrettyJSON(user))
//	// {
//	//   "id": "u-001",
//	//   "username": "admin"
//	// }
func PrettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// CompactJSON แปลง value เป็น JSON แบบบรรทัดเดียว (ไม่มี indent)
// เหมาะสำหรับ logging ที่ต้องการ single-line
//
//	log.Info().Str("payload", utils.CompactJSON(req)).Msg("received")
func CompactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// PrettyJSONBytes เหมือน PrettyJSON แต่รับ []byte แล้ว format ใหม่ให้สวย
// เหมาะสำหรับ Kafka message payload, raw HTTP body
//
//	formatted := utils.PrettyJSONBytes(msg.Payload)
func PrettyJSONBytes(data []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return string(data)
	}
	return buf.String()
}

// MustMarshal แปลง value เป็น []byte JSON — ถ้า marshal ไม่ได้จะ return []byte("{}")
// สะดวกในกรณีที่มั่นใจว่า struct ต้อง marshal ได้
//
//	body := utils.MustMarshal(event)
func MustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// SafeUnmarshal unmarshal JSON bytes เข้า dest — return false ถ้า error
// ลดโค้ด if err != nil ซ้ำ ๆ
//
//	var user domain.User
//	if !utils.SafeUnmarshal(data, &user) {
//	    // handle error
//	}
func SafeUnmarshal(data []byte, dest any) bool {
	return json.Unmarshal(data, dest) == nil
}

// MaskJSON ซ่อนค่า sensitive fields ใน JSON string (เช่น password, token)
// เหมาะสำหรับ audit log ที่ไม่ต้องการเปิดเผยข้อมูลลับ
//
//	masked := utils.MaskJSON(rawJSON, "password", "token", "secret")
//	// {"username":"admin","password":"***","token":"***"}
func MaskJSON(jsonStr string, fields ...string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	maskFields(data, fields)

	b, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}
	return string(b)
}

func maskFields(data map[string]any, fields []string) {
	for k, v := range data {
		for _, f := range fields {
			if strings.EqualFold(k, f) {
				data[k] = "***"
				break
			}
		}
		// ถ้า value เป็น nested object ให้ mask ลึกลงไปด้วย
		if nested, ok := v.(map[string]any); ok {
			maskFields(nested, fields)
		}
	}
}
