package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	// A valid case
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	validatedID, err := ValidateJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if validatedID != userID {
		t.Errorf("validatedID (%s) does not match original ID (%s)", validatedID, userID)
	}
}

func TestValidateJWTExpired(t *testing.T) {
	// An expired token
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := -time.Minute // Expired immediately

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	_, err = ValidateJWT(tokenString, secret)
	if err == nil {
		t.Fatalf("Expected an error for an expired token, but got none")
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	// Wrong secret
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Fatalf("Expected an error for wrong secret, but got none")
	}
}
