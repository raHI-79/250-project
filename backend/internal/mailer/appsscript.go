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
	appsScriptTimeout   = 20 * time.Second
	appsScriptMaxError  = 400
	appsScriptRedirects = 5
)

type appsScriptMessage struct {
	Secret  string `json:"secret"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Name    string `json:"name"`
}

type appsScriptAnswer struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type AppsScript struct {
	options Options
	client  *http.Client
}

func NewAppsScript(options Options) *AppsScript {
	if strings.TrimSpace(options.FromName) == "" {
		options.FromName = defaultFromName
	}
	return &AppsScript{
		options: options,
		client: &http.Client{
			Timeout: appsScriptTimeout,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= appsScriptRedirects {
					return fmt.Errorf("mailer: the mail script redirected more than %d times", appsScriptRedirects)
				}
				return nil
			},
		},
	}
}

func (a *AppsScript) Configured() bool {
	return strings.TrimSpace(a.options.Endpoint) != "" && strings.TrimSpace(a.options.Secret) != ""
}

func (a *AppsScript) Send(ctx context.Context, to, subject, body string) error {
	if !a.Configured() {
		return ErrNotConfigured
	}
	if strings.TrimSpace(to) == "" {
		return ErrNoRecipient
	}
	payload, err := json.Marshal(appsScriptMessage{
		Secret:  a.options.Secret,
		To:      to,
		Subject: header(subject),
		Body:    body,
		Name:    a.options.FromName,
	})
	if err != nil {
		return fmt.Errorf("mailer: could not build the message: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(a.options.Endpoint),
		bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("mailer: could not build the request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("mailer: could not reach the mail script: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= appsScriptMaxError {
		return fmt.Errorf("mailer: the mail script refused the message with status %d", response.StatusCode)
	}
	var answer appsScriptAnswer
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<12)).Decode(&answer); err != nil {
		return fmt.Errorf("mailer: the mail script did not answer with JSON: %w", err)
	}
	if !answer.OK {
		detail := strings.TrimSpace(answer.Error)
		if detail == "" {
			detail = "no reason given"
		}
		return fmt.Errorf("mailer: the mail script did not send the message: %s", detail)
	}
	return nil
}
