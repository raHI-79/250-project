package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	testSecret      = "0123456789abcdef0123456789abcdef"
	testOtherSecret = "fedcba9876543210fedcba9876543210"
	testTTL         = time.Hour
)

func newTestTokens(t *testing.T) *Tokens {
	t.Helper()
	tokens, err := NewTokens(testSecret, testTTL)
	if err != nil {
		t.Fatalf("NewTokens() returned an unexpected error: %v", err)
	}
	return tokens
}

func signRaw(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("could not sign the test token: %v", err)
	}
	return signed
}

func registeredClaims(subject string, issuedAt, expiresAt time.Time) jwt.RegisteredClaims {
	claims := jwt.RegisteredClaims{Subject: subject, IssuedAt: jwt.NewNumericDate(issuedAt)}
	if !expiresAt.IsZero() {
		claims.ExpiresAt = jwt.NewNumericDate(expiresAt)
	}
	return claims
}

func TestNewTokensValidatesItsInput(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		ttl     time.Duration
		wantErr error
	}{
		{name: "valid", secret: testSecret, ttl: testTTL},
		{name: "secret one character too short", secret: strings.Repeat("x", 31), ttl: testTTL, wantErr: ErrWeakSecret},
		{name: "empty secret", secret: "", ttl: testTTL, wantErr: ErrWeakSecret},
		{name: "zero lifetime", secret: testSecret, ttl: 0, wantErr: ErrInvalidTTL},
		{name: "negative lifetime", secret: testSecret, ttl: -time.Hour, wantErr: ErrInvalidTTL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := NewTokens(tc.secret, tc.ttl)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("NewTokens() error = %v, want %v", err, tc.wantErr)
				}
				if tokens != nil {
					t.Error("NewTokens() returned a value alongside the error, want nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewTokens() returned an unexpected error: %v", err)
			}
			if tokens.TTL() != tc.ttl {
				t.Errorf("TTL() = %v, want %v", tokens.TTL(), tc.ttl)
			}
		})
	}
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	tokens := newTestTokens(t)
	tests := []struct {
		name         string
		userID       int64
		role         string
		tokenVersion int
	}{
		{name: "student", userID: 1, role: RoleStudent, tokenVersion: 0},
		{name: "class representative", userID: 42, role: RoleCR, tokenVersion: 3},
		{name: "teacher", userID: 7, role: RoleTeacher, tokenVersion: 1},
		{name: "admin", userID: 9007199254740993, role: RoleAdmin, tokenVersion: 128},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tokens.IssueToken(tc.userID, tc.role, tc.tokenVersion)
			if err != nil {
				t.Fatalf("IssueToken() returned an unexpected error: %v", err)
			}
			claims, err := tokens.ParseToken(token)
			if err != nil {
				t.Fatalf("ParseToken() returned an unexpected error: %v", err)
			}
			if claims.UserID != tc.userID {
				t.Errorf("UserID = %d, want %d", claims.UserID, tc.userID)
			}
			if claims.Role != tc.role {
				t.Errorf("Role = %q, want %q", claims.Role, tc.role)
			}
			if claims.TokenVersion != tc.tokenVersion {
				t.Errorf("TokenVersion = %d, want %d", claims.TokenVersion, tc.tokenVersion)
			}
			if got := claims.ExpiresAt.Sub(claims.IssuedAt); got != testTTL {
				t.Errorf("token lifetime = %v, want %v", got, testTTL)
			}
			if !claims.ExpiresAt.After(time.Now()) {
				t.Error("a freshly issued token is already expired")
			}
		})
	}
}

func TestIssueTokenRejectsBadInput(t *testing.T) {
	tokens := newTestTokens(t)
	tests := []struct {
		name         string
		userID       int64
		role         string
		tokenVersion int
	}{
		{name: "zero user id", userID: 0, role: RoleStudent},
		{name: "negative user id", userID: -1, role: RoleStudent},
		{name: "unknown role", userID: 1, role: "principal"},
		{name: "empty role", userID: 1, role: ""},
		{name: "negative token version", userID: 1, role: RoleStudent, tokenVersion: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tokens.IssueToken(tc.userID, tc.role, tc.tokenVersion)
			if err == nil {
				t.Fatal("IssueToken() returned no error, want one")
			}
			if token != "" {
				t.Error("IssueToken() returned a token alongside the error, want an empty string")
			}
		})
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	tokens := newTestTokens(t)
	valid, err := tokens.IssueToken(5, RoleCR, 2)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	expiredTokens := newTestTokens(t)
	expiredTokens.now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	expired, err := expiredTokens.IssueToken(5, RoleCR, 2)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	otherTokens, err := NewTokens(testOtherSecret, testTTL)
	if err != nil {
		t.Fatalf("NewTokens() returned an unexpected error: %v", err)
	}
	wrongSecret, err := otherTokens.IssueToken(5, RoleCR, 2)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	now := time.Now()
	future := now.Add(testTTL)
	hs512 := signRaw(t, jwt.SigningMethodHS512, []byte(testSecret), tokenClaims{
		Role:             RoleCR,
		TokenVersion:     2,
		RegisteredClaims: registeredClaims("5", now, future),
	})
	unsigned := signRaw(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, tokenClaims{
		Role:             RoleAdmin,
		TokenVersion:     0,
		RegisteredClaims: registeredClaims("5", now, future),
	})
	noExpiry := signRaw(t, jwt.SigningMethodHS256, []byte(testSecret), tokenClaims{
		Role:             RoleCR,
		TokenVersion:     2,
		RegisteredClaims: registeredClaims("5", now, time.Time{}),
	})
	badSubject := signRaw(t, jwt.SigningMethodHS256, []byte(testSecret), tokenClaims{
		Role:             RoleCR,
		TokenVersion:     2,
		RegisteredClaims: registeredClaims("not-a-number", now, future),
	})
	zeroSubject := signRaw(t, jwt.SigningMethodHS256, []byte(testSecret), tokenClaims{
		Role:             RoleCR,
		TokenVersion:     2,
		RegisteredClaims: registeredClaims("0", now, future),
	})
	unknownRole := signRaw(t, jwt.SigningMethodHS256, []byte(testSecret), tokenClaims{
		Role:             "principal",
		TokenVersion:     2,
		RegisteredClaims: registeredClaims("5", now, future),
	})
	negativeVersion := signRaw(t, jwt.SigningMethodHS256, []byte(testSecret), tokenClaims{
		Role:             RoleCR,
		TokenVersion:     -1,
		RegisteredClaims: registeredClaims("5", now, future),
	})
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "not a token at all", token: "nonsense"},
		{name: "only two segments", token: strings.Join(strings.Split(valid, ".")[:2], ".")},
		{name: "tampered payload", token: valid[:len(valid)-4] + "AAAA"},
		{name: "expired", token: expired},
		{name: "signed with another secret", token: wrongSecret},
		{name: "signed with HS512", token: hs512},
		{name: "unsigned with algorithm none", token: unsigned},
		{name: "without an expiry claim", token: noExpiry},
		{name: "subject is not a number", token: badSubject},
		{name: "subject is zero", token: zeroSubject},
		{name: "unknown role", token: unknownRole},
		{name: "negative token version", token: negativeVersion},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := tokens.ParseToken(tc.token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("ParseToken() error = %v, want ErrInvalidToken", err)
			}
			if claims != (Claims{}) {
				t.Error("ParseToken() returned claims alongside the error, want the zero value")
			}
		})
	}
}

func TestParseTokenAcceptsOnlyItsOwnAlgorithm(t *testing.T) {
	tokens := newTestTokens(t)
	token, err := tokens.IssueToken(11, RoleTeacher, 0)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(token, &tokenClaims{})
	if err != nil {
		t.Fatalf("could not inspect the token header: %v", err)
	}
	if got := parsed.Method.Alg(); got != signingAlgorithm {
		t.Errorf("token algorithm = %q, want %q", got, signingAlgorithm)
	}
	claims, ok := parsed.Claims.(*tokenClaims)
	if !ok {
		t.Fatal("the token claims have an unexpected type")
	}
	if claims.Subject != strconv.FormatInt(11, 10) {
		t.Errorf("sub = %q, want %q", claims.Subject, "11")
	}
}
