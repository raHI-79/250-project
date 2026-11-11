package push

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	publicKeyBytes  = 65
	privateKeyBytes = 32
)

var (
	ErrNoKeys        = errors.New("push: VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are both required")
	ErrNoSubject     = errors.New("push: VAPID_SUBJECT is required, for example mailto:cse@sust.edu")
	ErrBadPublicKey  = errors.New("push: VAPID_PUBLIC_KEY is not a 65 byte base64url key")
	ErrBadPrivateKey = errors.New("push: VAPID_PRIVATE_KEY is not a 32 byte base64url key")
)

func Expired(status int) bool {
	switch status {
	case http.StatusNotFound, http.StatusGone, http.StatusForbidden:
		return true
	default:
		return false
	}
}

func ReasonOf(status int, err error) string {
	if err != nil {
		return store.PushReasonNetwork
	}
	switch {
	case status == 0:
		return store.PushReasonNetwork
	case status < http.StatusBadRequest:
		return ""
	case Expired(status):
		return store.PushReasonExpired
	case status == http.StatusUnauthorized:
		return store.PushReasonInvalidKey
	case status == http.StatusRequestEntityTooLarge || status == http.StatusBadRequest:
		return store.PushReasonPayload
	case status == http.StatusTooManyRequests:
		return store.PushReasonRateLimited
	default:
		return store.PushReasonOther
	}
}

func decodeKey(value string, want int) bool {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "=")
	if trimmed == "" {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return false
	}
	return len(raw) == want
}

func ValidateKeys(publicKey, privateKey, subject string) error {
	public := strings.TrimSpace(publicKey)
	private := strings.TrimSpace(privateKey)
	if public == "" || private == "" {
		return ErrNoKeys
	}
	if strings.TrimSpace(subject) == "" {
		return ErrNoSubject
	}
	if !decodeKey(public, publicKeyBytes) {
		return ErrBadPublicKey
	}
	if !decodeKey(private, privateKeyBytes) {
		return ErrBadPrivateKey
	}
	return nil
}

func Describe(publicKey, privateKey, subject string) string {
	if err := ValidateKeys(publicKey, privateKey, subject); err != nil {
		return "web push is off: " + err.Error()
	}
	return fmt.Sprintf("web push is ready, a VAPID key of %d characters for %s",
		len(strings.TrimSpace(publicKey)), strings.TrimSpace(subject))
}
