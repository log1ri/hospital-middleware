package utils_test

import (
	"testing"

	"hospital-middleware/internal/utils"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := utils.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "secret123" {
		t.Fatal("the password was stored in the clear")
	}
	if !utils.CheckPasswordHash("secret123", hash) {
		t.Error("the correct password was rejected")
	}
	if utils.CheckPasswordHash("secret124", hash) {
		t.Error("a wrong password was accepted")
	}
}

// bcrypt salts every hash, so the same password must not produce the same digest.
func TestHashesAreSalted(t *testing.T) {
	first, err := utils.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	second, err := utils.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if first == second {
		t.Error("two hashes of the same password are identical")
	}
}

func TestCheckRejectsGarbageHash(t *testing.T) {
	if utils.CheckPasswordHash("secret123", "not-a-bcrypt-hash") {
		t.Error("a malformed hash was treated as a match")
	}
}
