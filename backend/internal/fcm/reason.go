package fcm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/raHI-79/250-project/backend/internal/store"
)

func ReasonOf(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrGone):
		return store.PushReasonExpired
	case errors.Is(err, ErrNotConfigured):
		return store.PushReasonInvalidKey
	case errors.Is(err, ErrNoToken):
		return store.PushReasonPayload
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "could not reach the push service"):
		return store.PushReasonNetwork
	case strings.Contains(text, "could not get an access token"):
		return store.PushReasonInvalidKey
	case strings.Contains(text, "status 429"):
		return store.PushReasonRateLimited
	case strings.Contains(text, "status 401"), strings.Contains(text, "status 403"):
		return store.PushReasonInvalidKey
	case strings.Contains(text, "status 400"):
		return store.PushReasonPayload
	default:
		return store.PushReasonOther
	}
}

func Describe(client *Client, err error) string {
	if errors.Is(err, ErrNotConfigured) {
		return "app notifications are off: FIREBASE_SERVICE_ACCOUNT is not set"
	}
	if err != nil {
		return "app notifications are off: " + err.Error()
	}
	if !client.Configured() {
		return "app notifications are off: the service account carries no project"
	}
	return fmt.Sprintf("app notifications are ready for the Firebase project %s", client.projectID)
}
