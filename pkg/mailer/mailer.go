package mailer

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

var ErrLogOnly = errors.New("email delivery is disabled; message was logged only")

type Message struct {
	To        string
	Subject   string
	TextBody  string
	HTMLBody  string
	ActionURL string
}

type Sender interface {
	Send(context.Context, Message) error
}

type LogSender struct{ Log *slog.Logger }

func (s *LogSender) Send(_ context.Context, m Message) error {
	log := s.Log
	if log == nil {
		log = slog.Default()
	}
	if m.ActionURL != "" {
		// Keep the clickable URL separate from JSON-escaped newlines in the message body.
		log.Info("development email", "to", m.To, "subject", m.Subject, "link", m.ActionURL)
	} else {
		log.Info("development email", "to", m.To, "subject", m.Subject, "body", m.TextBody)
	}
	return ErrLogOnly
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	Timeout  time.Duration
}

type SMTPSender struct{ cfg SMTPConfig }

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid SMTP host or port")
	}
	if strings.TrimSpace(cfg.Username) == "" || cfg.Password == "" {
		return nil, fmt.Errorf("SMTP username and password are required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		cfg.From = cfg.Username
	}
	addr, err := mail.ParseAddress(cfg.From)
	if err != nil || !strings.EqualFold(addr.Address, strings.TrimSpace(cfg.From)) {
		return nil, fmt.Errorf("invalid SMTP from address")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPSender{cfg: cfg}, nil
}

func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	toAddr, err := mail.ParseAddress(strings.TrimSpace(m.To))
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	dialer := net.Dialer{Timeout: s.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.cfg.Timeout))

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	} else {
		return fmt.Errorf("smtp server does not advertise STARTTLS")
	}

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(toAddr.Address); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := io.WriteString(wc, buildMIME(s.cfg, toAddr.Address, m)); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func buildMIME(cfg SMTPConfig, to string, m Message) string {
	subject := sanitizeHeader(m.Subject)
	from := (&mail.Address{Name: sanitizeHeader(cfg.FromName), Address: cfg.From}).String()
	boundary := "sfedu-boundary-7caa1d57"
	var b strings.Builder
	w := bufio.NewWriter(&b)
	fmt.Fprintf(w, "From: %s\r\n", from)
	fmt.Fprintf(w, "To: %s\r\n", to)
	fmt.Fprintf(w, "Subject: %s\r\n", subject)
	fmt.Fprint(w, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(w, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(w, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n", boundary, m.TextBody)
	fmt.Fprintf(w, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n", boundary, m.HTMLBody)
	fmt.Fprintf(w, "--%s--\r\n", boundary)
	_ = w.Flush()
	return b.String()
}

func sanitizeHeader(v string) string {
	v = strings.ReplaceAll(v, "\r", " ")
	v = strings.ReplaceAll(v, "\n", " ")
	return strings.TrimSpace(v)
}
