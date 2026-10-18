package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

const (
	PurposeEmailProof = "email_proof"
	ProofTTL          = 15 * time.Minute
)

var (
	ErrInvalidProof = errors.New("auth: the email proof is not valid")
	ErrProofSubject = errors.New("auth: the email proof belongs to another address")
)

type proofClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

type Proofs struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewProofs(secret string, ttl time.Duration) (*Proofs, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrWeakSecret
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	return &Proofs{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

func (p *Proofs) TTL() time.Duration {
	return p.ttl
}

func (p *Proofs) IssueEmailProof(email string) (string, time.Time, error) {
	address := strings.ToLower(strings.TrimSpace(email))
	if address == "" {
		return "", time.Time{}, ErrInvalidProof
	}
	issuedAt := p.now()
	expiresAt := issuedAt.Add(p.ttl)
	claims := proofClaims{
		Purpose: PurposeEmailProof,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   address,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(p.secret)
	if err != nil {
		return "", time.Time{}, ErrInvalidProof
	}
	return signed, expiresAt, nil
}

func (p *Proofs) CheckEmailProof(token, email string) error {
	parsed := &proofClaims{}
	keyFunc := func(*jwt.Token) (any, error) { return p.secret, nil }
	if _, err := jwt.ParseWithClaims(token, parsed, keyFunc,
		jwt.WithValidMethods([]string{signingAlgorithm}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(p.now),
	); err != nil {
		return ErrInvalidProof
	}
	if parsed.Purpose != PurposeEmailProof {
		return ErrInvalidProof
	}
	if parsed.Subject == "" || parsed.Subject != strings.ToLower(strings.TrimSpace(email)) {
		return ErrProofSubject
	}
	return nil
}
