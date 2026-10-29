package fcm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	Scope        = "https://www.googleapis.com/auth/firebase.messaging"
	EndpointHost = "https://fcm.googleapis.com"
	Timeout      = 15 * time.Second
	TTL          = 24 * time.Hour
	PriorityHigh = "high"
	maxBody      = 1 << 14
)

var (
	ErrNotConfigured = errors.New("fcm: FIREBASE_SERVICE_ACCOUNT is not set")
	ErrNoToken       = errors.New("fcm: a message needs a device token")
	ErrGone          = errors.New("fcm: the device token is no longer registered")
)

type Message struct {
	Token string
	Title string
	Body  string
	URL   string
	Tag   string
}

type account struct {
	ProjectID string `json:"project_id"`
}

type androidNotification struct {
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Sound       string `json:"sound,omitempty"`
	ChannelID   string `json:"channel_id,omitempty"`
	Tag         string `json:"tag,omitempty"`
	ClickAction string `json:"click_action,omitempty"`
}

type androidConfig struct {
	Priority     string              `json:"priority"`
	TTL          string              `json:"ttl"`
	Notification androidNotification `json:"notification"`
}

type notification struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

type payload struct {
	Token        string            `json:"token"`
	Notification notification      `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Android      androidConfig     `json:"android"`
}

type envelope struct {
	Message payload `json:"message"`
}

type failure struct {
	Error struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

type Sender interface {
	Send(ctx context.Context, message Message) error
	Configured() bool
}

type Client struct {
	projectID string
	endpoint  string
	tokens    oauth2.TokenSource
	client    *http.Client
	channelID string
}

func New(encoded, endpoint, channelID string) (*Client, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, ErrNotConfigured
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		raw = []byte(trimmed)
	}
	var details account
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("fcm: the service account is not valid JSON: %w", err)
	}
	if details.ProjectID == "" {
		return nil, errors.New("fcm: the service account has no project_id")
	}
	credentials, err := google.CredentialsFromJSON(context.Background(), raw, Scope)
	if err != nil {
		return nil, fmt.Errorf("fcm: the service account was refused: %w", err)
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = EndpointHost
	}
	return &Client{
		projectID: details.ProjectID,
		endpoint:  strings.TrimRight(endpoint, "/"),
		tokens:    credentials.TokenSource,
		client:    &http.Client{Timeout: Timeout},
		channelID: channelID,
	}, nil
}

func (c *Client) Configured() bool {
	return c != nil && c.projectID != ""
}

func (c *Client) URL() string {
	return fmt.Sprintf("%s/v1/projects/%s/messages:send", c.endpoint, c.projectID)
}

func (c *Client) Send(ctx context.Context, message Message) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	if strings.TrimSpace(message.Token) == "" {
		return ErrNoToken
	}
	body, err := json.Marshal(envelope{Message: payload{
		Token:        message.Token,
		Notification: notification{Title: message.Title, Body: message.Body},
		Data: map[string]string{
			"url":   message.URL,
			"tag":   message.Tag,
			"title": message.Title,
			"body":  message.Body,
		},
		Android: androidConfig{
			Priority: PriorityHigh,
			TTL:      fmt.Sprintf("%ds", int(TTL.Seconds())),
			Notification: androidNotification{
				Icon:      "ic_stat_notify",
				Color:     "#7A1F2B",
				Sound:     "default",
				ChannelID: c.channelID,
				Tag:       message.Tag,
			},
		},
	}})
	if err != nil {
		return fmt.Errorf("fcm: could not build the message: %w", err)
	}
	token, err := c.tokens.Token()
	if err != nil {
		return fmt.Errorf("fcm: could not get an access token: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("fcm: could not build the request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("fcm: could not reach the push service: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusBadRequest {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	var reason failure
	_ = json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&reason)
	if Unregistered(response.StatusCode, reason.Error.Status) {
		return ErrGone
	}
	return fmt.Errorf("fcm: the push service answered with status %d", response.StatusCode)
}

func Unregistered(status int, reason string) bool {
	if status == http.StatusNotFound || status == http.StatusGone {
		return true
	}
	if status != http.StatusBadRequest && status != http.StatusForbidden {
		return false
	}
	switch reason {
	case "UNREGISTERED", "NOT_FOUND", "INVALID_ARGUMENT":
		return true
	default:
		return false
	}
}
