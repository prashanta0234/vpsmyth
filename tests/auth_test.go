package tests

import (
	"testing"
	"github.com/prashanta0234/vpsmyth/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	password := "supersecret123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	match, err := auth.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}
	if !match {
		t.Fatal("Password should match hash")
	}

	match, err = auth.VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}
	if match {
		t.Fatal("Wrong password should not match hash")
	}
}

func TestJWTToken(t *testing.T) {
	username := "admin"
	token, err := auth.GenerateToken(username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("Token is empty")
	}

	validatedUsername, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validatedUsername != username {
		t.Fatalf("Expected username %s, got %s", username, validatedUsername)
	}
}
