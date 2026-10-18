package googleauth

import (
	"context"
	"errors"
	"google.golang.org/api/idtoken"
	"strings"
)

var (
	ErrNotConfigured = errors.New("googleauth: GOOGLE_CLIENT_ID is not set, so no credential can be checked")
	ErrInvalidToken  = errors.New("googleauth: the Google credential is not valid")
)

type Identity struct {
	Email         string
	EmailVerified bool
	HostedDomain  string
}

type Validator interface {
	Validate(ctx context.Context, credential, audience string) (Identity, error)
}

type IDTokens struct{}

func New() *IDTokens {
	return &IDTokens{}
}

func (IDTokens) Validate(ctx context.Context, credential, audience string) (Identity, error) {
	if strings.TrimSpace(audience) == "" {
		return Identity{}, ErrNotConfigured
	}
	payload, err := idtoken.Validate(ctx, credential, audience)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	return Identity{
		Email:         strings.ToLower(claimText(payload.Claims, "email")),
		EmailVerified: claimBool(payload.Claims, "email_verified"),
		HostedDomain:  strings.ToLower(claimText(payload.Claims, "hd")),
	}, nil
}

func claimText(claims map[string]any, name string) string {
	value, ok := claims[name].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func claimBool(claims map[string]any, name string) bool {
	switch value := claims[name].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}
