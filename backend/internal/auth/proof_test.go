package auth

import (
	"errors"
	"testing"
	"time"
)

const proofSecret = "0123456789abcdef0123456789abcdef"

func TestNewProofs(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		ttl     time.Duration
		wantErr error
	}{
		{name: "valid", secret: proofSecret, ttl: ProofTTL},
		{name: "short secret", secret: "too-short", ttl: ProofTTL, wantErr: ErrWeakSecret},
		{name: "zero lifetime", secret: proofSecret, ttl: 0, wantErr: ErrInvalidTTL},
		{name: "negative lifetime", secret: proofSecret, ttl: -time.Minute, wantErr: ErrInvalidTTL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proofs, err := NewProofs(tc.secret, tc.ttl)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("NewProofs() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProofs() returned an unexpected error: %v", err)
			}
			if proofs.TTL() != tc.ttl {
				t.Errorf("TTL() = %v, want %v", proofs.TTL(), tc.ttl)
			}
		})
	}
}

func TestEmailProof(t *testing.T) {
	const email = "2023331079@student.sust.edu"
	tests := []struct {
		name     string
		issueFor string
		checkFor string
		ttl      time.Duration
		age      time.Duration
		secret   string
		wantErr  error
	}{
		{name: "fresh proof", issueFor: email, checkFor: email, ttl: ProofTTL},
		{name: "case does not matter", issueFor: email, checkFor: "2023331079@Student.SUST.edu", ttl: ProofTTL},
		{name: "spaces are trimmed", issueFor: email, checkFor: " " + email + " ", ttl: ProofTTL},
		{
			name:     "just inside the lifetime",
			issueFor: email,
			checkFor: email,
			ttl:      ProofTTL,
			age:      ProofTTL - time.Minute,
		},
		{
			name:     "expired proof",
			issueFor: email,
			checkFor: email,
			ttl:      ProofTTL,
			age:      ProofTTL + time.Minute,
			wantErr:  ErrInvalidProof,
		},
		{
			name:     "another email",
			issueFor: email,
			checkFor: "2023331080@student.sust.edu",
			ttl:      ProofTTL,
			wantErr:  ErrProofSubject,
		},
		{
			name:     "empty email",
			issueFor: email,
			checkFor: "",
			ttl:      ProofTTL,
			wantErr:  ErrProofSubject,
		},
		{
			name:     "another secret",
			issueFor: email,
			checkFor: email,
			ttl:      ProofTTL,
			secret:   "fedcba9876543210fedcba9876543210",
			wantErr:  ErrInvalidProof,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issuedAt := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
			issuer, err := NewProofs(proofSecret, tc.ttl)
			if err != nil {
				t.Fatalf("NewProofs() returned an unexpected error: %v", err)
			}
			issuer.now = func() time.Time { return issuedAt }
			token, expiresAt, err := issuer.IssueEmailProof(tc.issueFor)
			if err != nil {
				t.Fatalf("IssueEmailProof() returned an unexpected error: %v", err)
			}
			if token == "" {
				t.Fatal("IssueEmailProof() returned an empty token")
			}
			if want := issuedAt.Add(tc.ttl); !expiresAt.Equal(want) {
				t.Errorf("expiry = %v, want %v", expiresAt, want)
			}
			secret := proofSecret
			if tc.secret != "" {
				secret = tc.secret
			}
			checker, err := NewProofs(secret, tc.ttl)
			if err != nil {
				t.Fatalf("NewProofs() returned an unexpected error: %v", err)
			}
			checker.now = func() time.Time { return issuedAt.Add(tc.age) }
			err = checker.CheckEmailProof(token, tc.checkFor)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("CheckEmailProof() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckEmailProof() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestEmailProofRejectsOtherTokens(t *testing.T) {
	proofs, err := NewProofs(proofSecret, ProofTTL)
	if err != nil {
		t.Fatalf("NewProofs() returned an unexpected error: %v", err)
	}
	tokens, err := NewTokens(proofSecret, time.Hour)
	if err != nil {
		t.Fatalf("NewTokens() returned an unexpected error: %v", err)
	}
	session, err := tokens.IssueToken(11, RoleStudent, 0)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name  string
		token string
	}{
		{name: "a session token", token: session},
		{name: "nonsense", token: "not-a-token"},
		{name: "empty", token: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := proofs.CheckEmailProof(tc.token, "2023331079@student.sust.edu"); !errors.Is(err, ErrInvalidProof) {
				t.Errorf("CheckEmailProof() error = %v, want ErrInvalidProof", err)
			}
		})
	}
}

func TestIssueEmailProofNeedsAnEmail(t *testing.T) {
	proofs, err := NewProofs(proofSecret, ProofTTL)
	if err != nil {
		t.Fatalf("NewProofs() returned an unexpected error: %v", err)
	}
	if _, _, err := proofs.IssueEmailProof("  "); !errors.Is(err, ErrInvalidProof) {
		t.Errorf("IssueEmailProof() error = %v, want ErrInvalidProof", err)
	}
}
