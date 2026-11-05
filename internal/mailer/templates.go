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
	AppName     string
	Username    string
	ResetURL    string
	ExpiresIn   string
	Subject     string
	Locale      string
}

type renderedEmail struct {
	Subject  string
	TextBody string
	HTMLBody string
}

func renderPasswordResetEmail(data PasswordResetEmailData) (*renderedEmail, error) {
	locale := strings.ToLower(strings.TrimSpace(data.Locale))
	if locale != "th" {
		locale = "en"
	}

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

	textName := fmt.Sprintf("templates/password_reset_%s.txt", locale)
	htmlName := fmt.Sprintf("templates/password_reset_%s.html", locale)

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
		Subject:  data.Subject,
		TextBody: textBuf.String(),
		HTMLBody: htmlBuf.String(),
	}, nil
}
