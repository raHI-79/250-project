package mailer

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestConfigured(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "host set", host: "smtp.example.com", want: true},
		{name: "host empty", host: "", want: false},
		{name: "host blank", host: "   ", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sender := NewSMTP(Options{Host: tc.host})
			if got := sender.Configured(); got != tc.want {
				t.Errorf("Configured() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestNewSMTPDefaults(t *testing.T) {
	sender := NewSMTP(Options{Host: "smtp.example.com", Username: "robot@example.com"})
	if sender.options.Port != DefaultPort {
		t.Errorf("port = %d, want %d", sender.options.Port, DefaultPort)
	}
	if sender.options.From != "robot@example.com" {
		t.Errorf("from = %q, want the username", sender.options.From)
	}
	if got := sender.Address(); got != "smtp.example.com:587" {
		t.Errorf("Address() = %q, want %q", got, "smtp.example.com:587")
	}
}

func TestSendWithoutAHostIsNotConfigured(t *testing.T) {
	sender := NewSMTP(Options{})
	err := sender.Send(context.Background(), "student@student.sust.edu", "Subject", "Body")
	if err != ErrNotConfigured {
		t.Errorf("Send() error = %v, want %v", err, ErrNotConfigured)
	}
}

func TestSendNeedsARecipient(t *testing.T) {
	sender := NewSMTP(Options{Host: "smtp.example.com"})
	err := sender.Send(context.Background(), "  ", "Subject", "Body")
	if err != ErrNoRecipient {
		t.Errorf("Send() error = %v, want %v", err, ErrNoRecipient)
	}
}

func TestMessageHeadersAndBody(t *testing.T) {
	sentAt := time.Date(2026, time.March, 4, 9, 30, 0, 0, time.UTC)
	message := Message("noreply@sust.edu", "student@student.sust.edu", "Password reset code", "First line\nSecond line", sentAt)
	wants := []string{
		"From: noreply@sust.edu\r\n",
		"To: student@student.sust.edu\r\n",
		"Subject: Password reset code\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=UTF-8\r\n",
		"\r\n\r\nFirst line\r\nSecond line\r\n",
	}
	for _, want := range wants {
		if !strings.Contains(message, want) {
			t.Errorf("the message is missing %q\ngot:\n%s", want, message)
		}
	}
	if !strings.Contains(message, sentAt.Format(time.RFC1123Z)) {
		t.Error("the message has no Date header")
	}
}

func TestMessageSubjectCannotInjectHeaders(t *testing.T) {
	message := Message("noreply@sust.edu", "student@student.sust.edu", "Code\r\nBcc: attacker@example.com", "Body", time.Now())
	if strings.Contains(message, "\r\nBcc:") {
		t.Errorf("the subject started a new header line\ngot:\n%s", message)
	}
	if !strings.Contains(message, "Subject: Code  Bcc: attacker@example.com\r\n") {
		t.Errorf("the subject was not folded onto one line\ngot:\n%s", message)
	}
}
