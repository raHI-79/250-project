package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	VerifyEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	timeout        = 10 * time.Second
	maxBody        = 1 << 13
)

var (
	ErrNotConfigured = errors.New("turnstile: no secret key is set")
	ErrNoToken       = errors.New("turnstile: the browser sent no token")
	ErrRefused       = errors.New("turnstile: Cloudflare refused the token")
)

type Checker interface {
	Verify(ctx context.Context, token, remoteIP string) error
}

type answer struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

type Client struct {
	secret   string
	endpoint string
	client   *http.Client
}

func New(secret, endpoint string) *Client {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = VerifyEndpoint
	}
	return &Client{
		secret:   strings.TrimSpace(secret),
		endpoint: endpoint,
		client:   &http.Client{Timeout: timeout},
	}
}

func (c *Client) Configured() bool {
	return c.secret != ""
}

func (c *Client) Verify(ctx context.Context, token, remoteIP string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrNoToken
	}
	form := url.Values{}
	form.Set("secret", c.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile: could not build the request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("turnstile: could not reach Cloudflare: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("turnstile: Cloudflare answered with status %d", response.StatusCode)
	}
	var decoded answer
	if err := json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&decoded); err != nil {
		return fmt.Errorf("turnstile: Cloudflare did not answer with JSON: %w", err)
	}
	if !decoded.Success {
		return fmt.Errorf("%w: %s", ErrRefused, strings.Join(decoded.ErrorCodes, ", "))
	}
	return nil
}
