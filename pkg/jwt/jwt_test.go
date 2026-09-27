package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestRoundTripAndAlgorithmPinning(t *testing.T) {
	m := NewManager("01234567890123456789012345678901")
	token, err := m.Generate(Claims{UserID: 42, Role: "client", TokenVersion: 7}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.TokenVersion != 7 {
		t.Fatalf("claims=(user=%d, version=%d), want (42,7)", claims.UserID, claims.TokenVersion)
	}

	other := jwtlib.NewWithClaims(jwtlib.SigningMethodHS384, Claims{UserID: 42, Role: "admin"})
	otherToken, err := other.SignedString([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Parse(otherToken); err == nil {
		t.Fatal("HS384 token must be rejected")
	}
}
