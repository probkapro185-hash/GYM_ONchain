package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const issuer = "sfedu-crm"

type Claims struct {
	UserID       int64  `json:"user_id"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
	jwtlib.RegisteredClaims
}

type Manager struct{ secret []byte }

func NewManager(secret string) *Manager { return &Manager{secret: []byte(secret)} }

func (m *Manager) Generate(claims Claims, ttl time.Duration) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwtlib.RegisteredClaims{
		Issuer:    issuer,
		Subject:   fmt.Sprintf("%d", claims.UserID),
		IssuedAt:  jwtlib.NewNumericDate(now),
		NotBefore: jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwtlib.Token) (any, error) {
			if token.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
			}
			return m.secret, nil
		},
		jwtlib.WithIssuer(issuer),
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}),
		jwtlib.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || claims.UserID <= 0 {
		if err == nil {
			err = errors.New("invalid token")
		}
		return nil, err
	}
	return claims, nil
}
