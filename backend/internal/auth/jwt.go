package auth

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"time"
)

const (
	MinSecretLength  = 32
	signingAlgorithm = "HS256"
)

var (
	ErrWeakSecret   = fmt.Errorf("auth: the JWT secret must be at least %d characters long", MinSecretLength)
	ErrInvalidTTL   = errors.New("auth: the token lifetime must be a positive duration")
	ErrInvalidToken = errors.New("auth: the token is not valid")
)

type Claims struct {
	UserID       int64
	Role         string
	TokenVersion int
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

type tokenClaims struct {
	Role         string `json:"role"`
	TokenVersion int    `json:"ver"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokens(secret string, ttl time.Duration) (*Tokens, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrWeakSecret
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	return &Tokens{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

func (t *Tokens) TTL() time.Duration {
	return t.ttl
}

func (t *Tokens) IssueToken(userID int64, effectiveRole string, tokenVersion int) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("auth: user id must be positive, got %d", userID)
	}
	if !IsEffectiveRole(effectiveRole) {
		return "", fmt.Errorf("auth: %q is not an effective role", effectiveRole)
	}
	if tokenVersion < 0 {
		return "", fmt.Errorf("auth: token version must not be negative, got %d", tokenVersion)
	}
	issuedAt := t.now()
	claims := tokenClaims{
		Role:         effectiveRole,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(t.ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("auth: could not sign the token: %w", err)
	}
	return signed, nil
}

func (t *Tokens) ParseToken(token string) (Claims, error) {
	parsed := &tokenClaims{}
	keyFunc := func(*jwt.Token) (any, error) { return t.secret, nil }
	if _, err := jwt.ParseWithClaims(token, parsed, keyFunc,
		jwt.WithValidMethods([]string{signingAlgorithm}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(t.now),
	); err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	userID, err := strconv.ParseInt(parsed.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return Claims{}, ErrInvalidToken
	}
	if !IsEffectiveRole(parsed.Role) {
		return Claims{}, ErrInvalidToken
	}
	if parsed.TokenVersion < 0 {
		return Claims{}, ErrInvalidToken
	}
	if parsed.IssuedAt == nil || parsed.ExpiresAt == nil {
		return Claims{}, ErrInvalidToken
	}
	return Claims{
		UserID:       userID,
		Role:         parsed.Role,
		TokenVersion: parsed.TokenVersion,
		IssuedAt:     parsed.IssuedAt.Time,
		ExpiresAt:    parsed.ExpiresAt.Time,
	}, nil
}
