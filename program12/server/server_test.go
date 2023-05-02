package server

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var validToken string

var testKey = []byte("secret_key")

func init() {
	// Set custom claims
	claims := &jwtCustomClaims{
		"username",
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	validToken, _ = token.SignedString(testKey)
}

func TestEmptyToken(t *testing.T) {
	token := ""
	ret := IsValidateToken(token)
	if ret != false {
		t.Fatal("Empty token test failed")
	}
}

func TestInvalidToken(t *testing.T) {
	token := "Tests"
	ret := IsValidateToken(token)
	if ret != false {
		t.Fatalf("Token %s test failed", token)
	}
}

func TestValidToken(t *testing.T) {
	token := validToken
	ret := IsValidateToken(token)
	if ret != true {
		t.Fatalf("Token %s test failed", token)
	}
}
