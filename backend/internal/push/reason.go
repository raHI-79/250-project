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
	// publicKeyBytes is the length of an uncompressed P-256 point, which is what the
	// browser expects in applicationServerKey.
	publicKeyBytes = 65
	// privateKeyBytes is the length of a P-256 scalar.
	privateKeyBytes = 32
)

var (
	ErrNoKeys        = errors.New("push: VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are both required")
	ErrNoSubject     = errors.New("push: VAPID_SUBJECT is required, for example mailto:cse@sust.edu")
	ErrBadPublicKey  = errors.New("push: VAPID_PUBLIC_KEY is not a 65 byte base64url key")
	ErrBadPrivateKey = errors.New("push: VAPID_PRIVATE_KEY is not a 32 byte base64url key")
)

// Expired tells whether the push service has thrown the subscription away. A browser
// that cleared its site data, a reinstalled app or an unsubscribed tab all end here, and
// the row has to go: keeping it means every later send fails for good and drags the
// failure share of the whole portal above the alert threshold.
func Expired(status int) bool {
	switch status {
	case http.StatusNotFound, http.StatusGone, http.StatusForbidden:
		return true
	default:
		return false
	}
}

// ReasonOf names why a send did not arrive, or returns an empty string when it did. The
// names are the ones System health groups the failures by.
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

// ValidateKeys reads the VAPID pair once, at startup, so that a mistyped key is a clear
// line in the log instead of a quiet stream of refusals hours later.
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

// Describe writes the one startup line about web push. It never prints a key.
func Describe(publicKey, privateKey, subject string) string {
	if err := ValidateKeys(publicKey, privateKey, subject); err != nil {
		return "web push is off: " + err.Error()
	}
	return fmt.Sprintf("web push is ready, a VAPID key of %d characters for %s",
		len(strings.TrimSpace(publicKey)), strings.TrimSpace(subject))
}
