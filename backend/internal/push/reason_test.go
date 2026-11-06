package push

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/store"
)

func TestTheReasonNamesWhyANotificationDidNotArrive(t *testing.T) {
	tests := []struct {
		name   string
		status int
		err    error
		want   string
	}{
		{name: "delivered", status: http.StatusCreated, want: ""},
		{name: "accepted", status: http.StatusOK, want: ""},
		{name: "the service could not be reached", err: errors.New("dial tcp: timeout"),
			want: store.PushReasonNetwork},
		{name: "no answer at all", status: 0, want: store.PushReasonNetwork},
		{name: "gone", status: http.StatusGone, want: store.PushReasonExpired},
		{name: "not found", status: http.StatusNotFound, want: store.PushReasonExpired},
		{name: "refused", status: http.StatusForbidden, want: store.PushReasonExpired},
		{name: "the key was refused", status: http.StatusUnauthorized,
			want: store.PushReasonInvalidKey},
		{name: "the payload was too large", status: http.StatusRequestEntityTooLarge,
			want: store.PushReasonPayload},
		{name: "the payload was refused", status: http.StatusBadRequest,
			want: store.PushReasonPayload},
		{name: "rate limited", status: http.StatusTooManyRequests,
			want: store.PushReasonRateLimited},
		{name: "the service broke", status: http.StatusInternalServerError,
			want: store.PushReasonOther},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ReasonOf(test.status, test.err); got != test.want {
				t.Errorf("ReasonOf(%d, %v) = %q, want %q", test.status, test.err, got, test.want)
			}
		})
	}
}

// A subscription the push service has thrown away has to go on the spot. Leaving it in
// place means the same refusal on every later send, which keeps the failure share of the
// whole portal above its alert threshold for good.
func TestExpiredCoversEveryAnswerThatMeansTheSubscriptionIsGone(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusGone, http.StatusForbidden} {
		if !Expired(status) {
			t.Errorf("Expired(%d) = false, want true", status)
		}
	}
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusUnauthorized,
		http.StatusTooManyRequests, http.StatusInternalServerError, 0} {
		if Expired(status) {
			t.Errorf("Expired(%d) = true, want false", status)
		}
	}
}

const (
	// A real uncompressed P-256 point and scalar, in base64url, with no padding.
	goodPublic  = "BFs6tKJYkDr4hGzEqhiJYQUNr3UcLDvZwLRt0wVyB6c0UNEGfp8Ow0U4m0brLZyD9AnBsYQTDsOyUJoyTeUkT6g"
	goodPrivate = "Yk0kVxl3ZtK0kqzFk0TmcHAEaGlZ9wzLQmRTJ5dFYQk"
)

func TestTheVapidKeysAreCheckedOnce(t *testing.T) {
	tests := []struct {
		name    string
		public  string
		private string
		subject string
		want    error
	}{
		{
			name:    "a good pair",
			public:  goodPublic,
			private: goodPrivate,
			subject: "mailto:cse@sust.edu",
		},
		{name: "no keys", subject: "mailto:cse@sust.edu", want: ErrNoKeys},
		{
			name:    "only a public key",
			public:  goodPublic,
			subject: "mailto:cse@sust.edu",
			want:    ErrNoKeys,
		},
		{
			name:    "no subject",
			public:  goodPublic,
			private: goodPrivate,
			want:    ErrNoSubject,
		},
		{
			name:    "a public key that is not base64url",
			public:  "not a key at all!!",
			private: goodPrivate,
			subject: "mailto:cse@sust.edu",
			want:    ErrBadPublicKey,
		},
		{
			name:    "a public key of the wrong length",
			public:  "BFs6tKJYkDr4hGzEqhiJYQ",
			private: goodPrivate,
			subject: "mailto:cse@sust.edu",
			want:    ErrBadPublicKey,
		},
		{
			name:    "a private key of the wrong length",
			public:  goodPublic,
			private: "Yk0kVxl3ZtK0kqzFk0Tm",
			subject: "mailto:cse@sust.edu",
			want:    ErrBadPrivateKey,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateKeys(test.public, test.private, test.subject)
			if !errors.Is(err, test.want) {
				t.Errorf("ValidateKeys() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestTheStartupLineNeverPrintsAKey(t *testing.T) {
	ready := Describe(goodPublic, goodPrivate, "mailto:cse@sust.edu")
	if !strings.Contains(ready, "web push is ready") {
		t.Errorf("the line = %q, want it to say web push is ready", ready)
	}
	for _, secret := range []string{goodPublic, goodPrivate} {
		if strings.Contains(ready, secret) {
			t.Error("the startup line printed a key")
		}
	}
	off := Describe("", "", "")
	if !strings.Contains(off, "web push is off") {
		t.Errorf("the line = %q, want it to say web push is off", off)
	}
	broken := Describe("not a key", goodPrivate, "mailto:cse@sust.edu")
	if !strings.Contains(broken, ErrBadPublicKey.Error()) {
		t.Errorf("the line = %q, want it to name the broken key", broken)
	}
}
