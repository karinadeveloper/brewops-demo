// Package token generates and validates the JWTs BrewOps uses for auth.
// It has no dependency on domain, service, or middleware, so both the auth
// service (which issues tokens) and the auth middleware (which validates
// them) can depend on it without a cycle.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Kind distinguishes an access token from a refresh token so one can never
// be accepted where the other is expected.
type Kind string

const (
	KindAccess  Kind = "access"
	KindRefresh Kind = "refresh"
)

var ErrInvalid = errors.New("invalid token")

// Claims is the JWT payload: the standard registered claims (subject =
// user ID, expiry, issued-at) plus the token Kind.
type Claims struct {
	Kind Kind `json:"kind"`
	jwt.RegisteredClaims
}

// Generate signs a new JWT of the given kind for userID, valid for ttl.
func Generate(secret []byte, userID uuid.UUID, kind Kind, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Kind: kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

// Parse validates a JWT's signature, expiry, and kind, returning its claims.
// Any failure — bad signature, expired, wrong kind, malformed subject — is
// reported as ErrInvalid so callers never need to distinguish the reason.
func Parse(secret []byte, tokenString string, wantKind Kind) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalid
	}
	if claims.Kind != wantKind {
		return nil, ErrInvalid
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, ErrInvalid
	}
	return claims, nil
}
