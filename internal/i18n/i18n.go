package i18n

import (
	"go-be/internal/middleware"

	"github.com/gin-gonic/gin"
)

var messages = map[string]map[string]string{
	"en": {
		"auth.invalid_request":       "Invalid request data",
		"auth.invalid_input":         "Invalid input",
		"auth.email_exists":          "Email already exists",
		"auth.username_exists":       "Username already exists",
		"auth.invalid_credentials":   "Invalid email or password",
		"auth.register_success":      "User registered successfully",
		"auth.login_success":         "Login successful",
		"auth.logout_success":        "Logout successful",
		"auth.unauthorized":          "Unauthorized",
		"auth.invalid_refresh_token": "Invalid refresh token",
		"common.internal_error":      "Internal server error",
		"health.running":             "Go Auth API is running",
		"health.db_ok":               "PostgreSQL connected",
		"health.db_unavailable":      "Database unavailable",
	},
	"th": {
		"auth.invalid_request":       "ข้อมูลไม่ถูกต้อง",
		"auth.invalid_input":         "ข้อมูลไม่ถูกต้อง",
		"auth.email_exists":          "อีเมลนี้ถูกใช้งานแล้ว",
		"auth.username_exists":       "ชื่อผู้ใช้นี้ถูกใช้งานแล้ว",
		"auth.invalid_credentials":   "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
		"auth.register_success":      "สมัครสมาชิกสำเร็จ",
		"auth.login_success":         "เข้าสู่ระบบสำเร็จ",
		"auth.logout_success":        "ออกจากระบบสำเร็จ",
		"auth.unauthorized":          "ไม่มีสิทธิ์เข้าถึง",
		"auth.invalid_refresh_token": "รีเฟรชโทเค็นไม่ถูกต้อง",
		"common.internal_error":      "เกิดข้อผิดพลาดภายในระบบ",
		"health.running":             "Go Auth API ทำงานปกติ",
		"health.db_ok":               "เชื่อมต่อ PostgreSQL สำเร็จ",
		"health.db_unavailable":      "ฐานข้อมูลไม่พร้อมใช้งาน",
	},
}

func T(c *gin.Context, key string) string {
	lang := "en"
	if v, ok := c.Get(middleware.LocaleKey); ok {
		if s, ok := v.(string); ok && s != "" {
			lang = s
		}
	}
	if msg, ok := messages[lang][key]; ok {
		return msg
	}
	return messages["en"][key]
}
