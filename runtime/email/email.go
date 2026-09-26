package email

import (
	"fmt"
	"net/smtp"
	"strings"
	"sync"
)

// Config holds SMTP configuration for sending emails.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

// Client wraps Go's net/smtp for Germanio.
type Client struct {
	mu     sync.Mutex
	config Config
}

// Novo creates a new email client with the given config.
func Novo(cfg Config) *Client {
	// Default "from" to the user if not set
	if cfg.From == "" {
		cfg.From = cfg.User
	}
	return &Client{config: cfg}
}

// EnviarEmail sends an email via SMTP.
func (c *Client) EnviarEmail(to, subject, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cfg := c.config
	addr := cfg.Host + ":" + cfg.Port

	// Detect HTML content
	contentType := "text/plain"
	if strings.Contains(body, "<") && strings.Contains(body, ">") {
		contentType = "text/html"
	}

	// Build RFC 2822 message
	var msg strings.Builder
	msg.WriteString("From: " + cfg.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: " + contentType + "; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	recipients := []string{to}
	err := smtp.SendMail(addr, auth, cfg.From, recipients, []byte(msg.String()))
	if err != nil {
		return fmt.Errorf("erro ao enviar email para %s: %w", to, err)
	}

	fmt.Printf("[email] Email enviado para %s (assunto: %s)\n", to, subject)
	return nil
}
