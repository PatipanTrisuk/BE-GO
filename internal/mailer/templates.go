package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	texttemplate "text/template"
)

//go:embed templates/*
var templateFS embed.FS

type PasswordResetEmailData struct {
	AppName   string
	Username  string
	ResetURL  string
	ExpiresIn string
	Subject   string
	Locale    string
}

type EmailVerificationData struct {
	AppName   string
	Username  string
	VerifyURL string
	ExpiresIn string
	Subject   string
	Locale    string
}

type renderedEmail struct {
	Subject  string
	TextBody string
	HTMLBody string
}

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale != "th" {
		return "en"
	}
	return locale
}

func renderEmail(textName, htmlName string, data any, subject string) (*renderedEmail, error) {
	textTmpl, err := texttemplate.ParseFS(templateFS, textName)
	if err != nil {
		return nil, fmt.Errorf("parse text template: %w", err)
	}
	htmlTmpl, err := template.ParseFS(templateFS, htmlName)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("render text template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("render html template: %w", err)
	}

	return &renderedEmail{
		Subject:  subject,
		TextBody: textBuf.String(),
		HTMLBody: htmlBuf.String(),
	}, nil
}

func renderPasswordResetEmail(data PasswordResetEmailData) (*renderedEmail, error) {
	locale := normalizeLocale(data.Locale)

	if data.AppName == "" {
		data.AppName = "Auth App"
	}
	if data.ExpiresIn == "" {
		if locale == "th" {
			data.ExpiresIn = "1 ชั่วโมง"
		} else {
			data.ExpiresIn = "1 hour"
		}
	}
	if data.Subject == "" {
		if locale == "th" {
			data.Subject = "ตั้งค่ารหัสผ่านของคุณ"
		} else {
			data.Subject = "Set your password"
		}
	}

	return renderEmail(
		fmt.Sprintf("templates/password_reset_%s.txt", locale),
		fmt.Sprintf("templates/password_reset_%s.html", locale),
		data,
		data.Subject,
	)
}

func renderEmailVerificationEmail(data EmailVerificationData) (*renderedEmail, error) {
	locale := normalizeLocale(data.Locale)

	if data.AppName == "" {
		data.AppName = "Auth App"
	}
	if data.ExpiresIn == "" {
		if locale == "th" {
			data.ExpiresIn = "24 ชั่วโมง"
		} else {
			data.ExpiresIn = "24 hours"
		}
	}
	if data.Subject == "" {
		if locale == "th" {
			data.Subject = "ยืนยันอีเมลของคุณ"
		} else {
			data.Subject = "Verify your email"
		}
	}

	return renderEmail(
		fmt.Sprintf("templates/email_verify_%s.txt", locale),
		fmt.Sprintf("templates/email_verify_%s.html", locale),
		data,
		data.Subject,
	)
}
