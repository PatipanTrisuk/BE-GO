package mailer

import (
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"os"
	"strings"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
	appName  string
}

func NewFromEnv() *Mailer {
	m := &Mailer{
		host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		port:     firstNonEmpty(os.Getenv("SMTP_PORT"), "587"),
		username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		from:     firstNonEmpty(os.Getenv("SMTP_FROM"), os.Getenv("SMTP_USERNAME"), "noreply@localhost"),
		appName:  firstNonEmpty(os.Getenv("APP_NAME"), "Auth App"),
	}

	// Incomplete SMTP config stays in log-only mode so local dev keeps working.
	if m.host != "" && (m.username == "" || m.password == "") {
		log.Printf("[mailer] SMTP host set but username/password incomplete — logging emails only")
		m.host = ""
	}

	if m.host != "" {
		log.Printf("[mailer] SMTP enabled host=%s port=%s from=%s", m.host, m.port, m.from)
	} else {
		log.Printf("[mailer] SMTP not configured — password-reset emails will be logged only")
	}

	return m
}

func (m *Mailer) SendPasswordReset(to string, data PasswordResetEmailData) error {
	if data.AppName == "" {
		data.AppName = m.appName
	}

	rendered, err := renderPasswordResetEmail(data)
	if err != nil {
		return err
	}

	return m.send(to, rendered.Subject, rendered.TextBody, rendered.HTMLBody)
}

func (m *Mailer) send(to, subject, textBody, htmlBody string) error {
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	boundary := "auth-app-boundary"

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("From: %s\r\n", m.from))
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msgBuilder.WriteString("\r\n")

	msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msgBuilder.WriteString(textBody)
	msgBuilder.WriteString("\r\n")

	msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msgBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msgBuilder.WriteString(htmlBody)
	msgBuilder.WriteString("\r\n")

	msgBuilder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	msg := msgBuilder.String()

	if m.host == "" {
		log.Printf(
			"[mailer] SMTP not configured — email to=%s subject=%q\n%s\n--- html ---\n%s",
			to,
			subject,
			textBody,
			htmlBody,
		)
		return nil
	}

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	if err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	log.Printf("[mailer] sent email to=%s subject=%q", to, subject)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
