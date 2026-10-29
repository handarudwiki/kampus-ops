// Package token issues and verifies JWT access tokens and opaque refresh tokens.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/kampusops/backend/internal/platform/appctx"
)

const AccessTTL = 15 * time.Minute
const RefreshTTL = 7 * 24 * time.Hour

type Claims struct {
	TenantID string             `json:"tid"`
	Name     string             `json:"name"`
	Roles    []appctx.UserRole  `json:"roles"`
	jwt.RegisteredClaims
}

var ErrExpired = errors.New("token expired")

type Issuer struct{ secret []byte }

func NewIssuer(secret string) *Issuer { return &Issuer{secret: []byte(secret)} }

func (i *Issuer) Access(p appctx.Principal, now time.Time) (string, error) {
	c := Claims{
		TenantID: p.TenantID, Name: p.Name, Roles: p.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
			Issuer:    "kampusops",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(i.secret)
}

func (i *Issuer) Parse(s string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(s, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	}, jwt.WithIssuer("kampusops"))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, err
	}
	return c, nil
}

// NewRefresh returns a random opaque refresh token and its SHA-256 hash (stored in DB).
func NewRefresh() (raw, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefresh(raw)
}

func HashRefresh(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
