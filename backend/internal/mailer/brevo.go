package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	BrevoEndpoint   = "https://api.brevo.com/v3/smtp/email"
	BrevoKeyHeader  = "api-key"
	brevoTimeout    = 15 * time.Second
	brevoMaxError   = 400
	defaultFromName = "SUST CSE Portal"
)

type brevoAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoMessage struct {
	Sender      brevoAddress   `json:"sender"`
	To          []brevoAddress `json:"to"`
	Subject     string         `json:"subject"`
	TextContent string         `json:"textContent"`
}

type Brevo struct {
	options  Options
	endpoint string
	client   *http.Client
}

func NewBrevo(options Options) *Brevo {
	if strings.TrimSpace(options.FromName) == "" {
		options.FromName = defaultFromName
	}
	endpoint := strings.TrimSpace(options.Endpoint)
	if endpoint == "" {
		endpoint = BrevoEndpoint
	}
	return &Brevo{
		options:  options,
		endpoint: endpoint,
		client:   &http.Client{Timeout: brevoTimeout},
	}
}

func (b *Brevo) Configured() bool {
	return strings.TrimSpace(b.options.APIKey) != "" && strings.TrimSpace(b.options.From) != ""
}

func (b *Brevo) Send(ctx context.Context, to, subject, body string) error {
	if !b.Configured() {
		return ErrNotConfigured
	}
	if strings.TrimSpace(to) == "" {
		return ErrNoRecipient
	}
	payload, err := json.Marshal(brevoMessage{
		Sender:      brevoAddress{Email: b.options.From, Name: b.options.FromName},
		To:          []brevoAddress{{Email: to}},
		Subject:     header(subject),
		TextContent: body,
	})
	if err != nil {
		return fmt.Errorf("mailer: could not build the message: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("mailer: could not build the request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set(BrevoKeyHeader, b.options.APIKey)
	response, err := b.client.Do(request)
	if err != nil {
		return fmt.Errorf("mailer: could not reach the mail service: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= brevoMaxError {
		return fmt.Errorf("mailer: the mail service refused the message with status %d: %s",
			response.StatusCode, reason(response.Body))
	}
	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}

func reason(body io.Reader) string {
	var answer struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 1<<12)).Decode(&answer); err != nil {
		return "no reason given"
	}
	text := strings.TrimSpace(answer.Code + " " + answer.Message)
	if text == "" {
		return "no reason given"
	}
	return text
}
