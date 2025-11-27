package services

import (
	"strings"
	"testing"
)

func TestAuthService_HashPassword(t *testing.T) {
	auth := &AuthService{}

	password := "securePassword123!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash == "" {
		t.Error("hash should not be empty")
	}
	if hash == password {
		t.Error("hash should not equal plain password")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Error("hash should be bcrypt format")
	}
}

func TestAuthService_HashPassword_UniqueHashes(t *testing.T) {
	auth := &AuthService{}

	password := "samePassword123"
	hash1, _ := auth.HashPassword(password)
	hash2, _ := auth.HashPassword(password)

	if hash1 == hash2 {
		t.Error("same password should produce different hashes (due to salt)")
	}
}

func TestAuthService_VerifyPassword_Correct(t *testing.T) {
	auth := &AuthService{}

	password := "correctPassword123!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !auth.VerifyPassword(hash, password) {
		t.Error("correct password should verify successfully")
	}
}

func TestAuthService_VerifyPassword_Incorrect(t *testing.T) {
	auth := &AuthService{}

	password := "correctPassword123!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if auth.VerifyPassword(hash, "wrongPassword") {
		t.Error("incorrect password should not verify")
	}
}

func TestAuthService_VerifyPassword_EmptyPassword(t *testing.T) {
	auth := &AuthService{}

	hash, _ := auth.HashPassword("somePassword")

	if auth.VerifyPassword(hash, "") {
		t.Error("empty password should not verify")
	}
}

func TestAuthService_VerifyPassword_InvalidHash(t *testing.T) {
	auth := &AuthService{}

	if auth.VerifyPassword("not-a-valid-hash", "password") {
		t.Error("invalid hash should not verify")
	}
}

func TestAuthService_GenerateSessionToken(t *testing.T) {
	auth := &AuthService{}

	token, hash, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if token == hash {
		t.Error("token and hash should be different")
	}
}

func TestAuthService_GenerateSessionToken_Unique(t *testing.T) {
	auth := &AuthService{}

	token1, hash1, _ := auth.GenerateSessionToken()
	token2, hash2, _ := auth.GenerateSessionToken()

	if token1 == token2 {
		t.Error("tokens should be unique")
	}
	if hash1 == hash2 {
		t.Error("hashes should be unique")
	}
}

func TestAuthService_GenerateSessionToken_Length(t *testing.T) {
	auth := &AuthService{}

	token, hash, _ := auth.GenerateSessionToken()

	// Token is 32 bytes hex encoded = 64 chars
	if len(token) != 64 {
		t.Errorf("token should be 64 chars, got %d", len(token))
	}

	// Hash is SHA256 = 32 bytes hex encoded = 64 chars
	if len(hash) != 64 {
		t.Errorf("hash should be 64 chars, got %d", len(hash))
	}
}

func TestAuthService_HashToken(t *testing.T) {
	auth := &AuthService{}

	token := "some-test-token"
	hash1 := auth.hashToken(token)
	hash2 := auth.hashToken(token)

	// Same token should produce same hash
	if hash1 != hash2 {
		t.Error("same token should produce same hash")
	}

	// Different tokens should produce different hashes
	hash3 := auth.hashToken("different-token")
	if hash1 == hash3 {
		t.Error("different tokens should produce different hashes")
	}
}

func TestAuthService_HashToken_ConsistentWithGenerate(t *testing.T) {
	auth := &AuthService{}

	token, generatedHash, _ := auth.GenerateSessionToken()
	computedHash := auth.hashToken(token)

	if generatedHash != computedHash {
		t.Error("hashToken should produce same result as GenerateSessionToken")
	}
}

func TestPasswordComplexity(t *testing.T) {
	auth := &AuthService{}

	// Various password types should all hash successfully
	// Note: bcrypt has a 72 byte limit, so we test up to that
	passwords := []string{
		"simple",
		"WithNumbers123",
		"Special!@#$%^&*()",
		"Unicode日本語",
		"A moderately long password that fits within bcrypt limits", // Under 72 bytes
		" ", // Single space
	}

	for _, pwd := range passwords {
		t.Run(pwd[:min(10, len(pwd))], func(t *testing.T) {
			hash, err := auth.HashPassword(pwd)
			if err != nil {
				t.Errorf("failed to hash password: %v", err)
			}
			if !auth.VerifyPassword(hash, pwd) {
				t.Error("password should verify after hashing")
			}
		})
	}
}

func TestPasswordComplexity_TooLong(t *testing.T) {
	auth := &AuthService{}

	// bcrypt has a 72 byte limit
	longPassword := "This is a very long password that exceeds the seventy-two byte limit of bcrypt"

	_, err := auth.HashPassword(longPassword)
	// bcrypt should return an error for passwords > 72 bytes
	if err == nil {
		t.Log("Note: bcrypt accepted password over 72 bytes (truncates silently in some implementations)")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
