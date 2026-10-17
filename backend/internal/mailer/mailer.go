package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort = 587
	dialTimeout = 10 * time.Second
)

var (
	ErrNotConfigured = errors.New("mailer: SMTP_HOST is not set, so no mail can be sent")
	ErrNoRecipient   = errors.New("mailer: a message needs a recipient")
)

type Sender interface {
	Configured() bool
	Send(ctx context.Context, to, subject, body string) error
}

type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SMTP struct {
	options Options
	now     func() time.Time
}

func NewSMTP(options Options) *SMTP {
	if options.Port == 0 {
		options.Port = DefaultPort
	}
	if options.From == "" {
		options.From = options.Username
	}
	return &SMTP{options: options, now: time.Now}
}

func (s *SMTP) Configured() bool {
	return strings.TrimSpace(s.options.Host) != ""
}

func (s *SMTP) Address() string {
	return net.JoinHostPort(s.options.Host, strconv.Itoa(s.options.Port))
}

func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	if strings.TrimSpace(to) == "" {
		return ErrNoRecipient
	}
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.Address())
	if err != nil {
		return fmt.Errorf("mailer: could not reach the SMTP server: %w", err)
	}
	client, err := smtp.NewClient(conn, s.options.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: could not start the SMTP session: %w", err)
	}
	defer func() { _ = client.Close() }()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.options.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mailer: could not upgrade the SMTP session to TLS: %w", err)
		}
	}
	if s.options.Username != "" {
		auth := smtp.PlainAuth("", s.options.Username, s.options.Password, s.options.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: the SMTP server rejected the credentials: %w", err)
		}
	}
	if err := client.Mail(s.options.From); err != nil {
		return fmt.Errorf("mailer: the SMTP server rejected the sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: the SMTP server rejected the recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: the SMTP server refused the message body: %w", err)
	}
	if _, err := writer.Write([]byte(Message(s.options.From, to, subject, body, s.now()))); err != nil {
		_ = writer.Close()
		return fmt.Errorf("mailer: could not write the message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("mailer: could not finish the message: %w", err)
	}
	return client.Quit()
}

func Message(from, to, subject, body string, sentAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("From: " + from + "\r\n")
	builder.WriteString("To: " + to + "\r\n")
	builder.WriteString("Subject: " + header(subject) + "\r\n")
	builder.WriteString("Date: " + sentAt.Format(time.RFC1123Z) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	builder.WriteString("\r\n")
	return builder.String()
}

func header(value string) string {
	cleaned := strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	return strings.TrimSpace(cleaned)
}
