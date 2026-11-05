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
		"auth.account_locked":        "Too many failed login attempts. Try again in 60 minutes.",
		"auth.register_success":      "User registered successfully",
		"auth.login_success":         "Login successful",
		"auth.logout_success":        "Logout successful",
		"auth.unauthorized":          "Unauthorized",
		"auth.forbidden":             "Forbidden",
		"auth.invalid_refresh_token":   "Invalid refresh token",
		"auth.password_reset_sent":     "If that email exists, a reset link has been sent",
		"auth.password_reset_success":  "Password updated successfully",
		"auth.invalid_reset_token":     "This reset link is invalid or has expired",
		"auth.discord_not_configured":  "Discord login is not configured",
		"auth.discord_email_missing":   "Discord account must have a verified email",
		"auth.discord_failed":          "Discord login failed",
		"auth.discord_email_conflict":  "This email is already linked to another Discord account",
		"common.internal_error":        "Internal server error",
		"common.too_many_requests":   "Too many requests. Please try again later.",
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
		"auth.account_locked":        "เข้าสู่ระบบผิดหลายครั้ง บัญชีถูกบล็อก 60 นาที",
		"auth.register_success":      "สมัครสมาชิกสำเร็จ",
		"auth.login_success":         "เข้าสู่ระบบสำเร็จ",
		"auth.logout_success":        "ออกจากระบบสำเร็จ",
		"auth.unauthorized":          "ไม่มีสิทธิ์เข้าถึง",
		"auth.forbidden":             "ไม่มีสิทธิ์ผู้ดูแลระบบ",
		"auth.invalid_refresh_token":  "รีเฟรชโทเค็นไม่ถูกต้อง",
		"auth.password_reset_sent":    "หากอีเมลมีในระบบ เราได้ส่งลิงก์ตั้งค่ารหัสผ่านแล้ว",
		"auth.password_reset_success": "อัปเดตรหัสผ่านสำเร็จ",
		"auth.invalid_reset_token":    "ลิงก์ตั้งค่ารหัสผ่านไม่ถูกต้องหรือหมดอายุแล้ว",
		"auth.discord_not_configured": "ยังไม่ได้ตั้งค่า Discord login",
		"auth.discord_email_missing":  "บัญชี Discord ต้องมีอีเมลที่ยืนยันแล้ว",
		"auth.discord_failed":         "เข้าสู่ระบบด้วย Discord ไม่สำเร็จ",
		"auth.discord_email_conflict": "อีเมลนี้ถูกเชื่อมกับบัญชี Discord อื่นแล้ว",
		"common.internal_error":       "เกิดข้อผิดพลาดภายในระบบ",
		"common.too_many_requests":   "คำขอมากเกินไป กรุณาลองใหม่ภายหลัง",
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
