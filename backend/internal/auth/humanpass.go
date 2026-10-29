package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	PurposeHumanPass = "human_pass"
	HumanPassBrowser = 24 * time.Hour
	HumanPassApp     = 30 * 24 * time.Hour
)

var ErrInvalidHumanPass = errors.New("auth: the human pass is not valid")

type humanClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

type HumanPasses struct {
	secret []byte
	now    func() time.Time
}

func NewHumanPasses(secret string) (*HumanPasses, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrWeakSecret
	}
	return &HumanPasses{secret: []byte(secret), now: time.Now}, nil
}

func (h *HumanPasses) Issue(ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		return "", time.Time{}, ErrInvalidTTL
	}
	issuedAt := h.now()
	expiresAt := issuedAt.Add(ttl)
	claims := humanClaims{
		Purpose: PurposeHumanPass,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (h *HumanPasses) Check(token string) error {
	var claims humanClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidHumanPass
		}
		return h.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(h.now))
	if err != nil || !parsed.Valid || claims.Purpose != PurposeHumanPass {
		return ErrInvalidHumanPass
	}
	return nil
}
