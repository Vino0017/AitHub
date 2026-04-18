package crypto

import (
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if !strings.HasPrefix(token, "sk_") {
		t.Errorf("Expected sk_ prefix, got '%s'", token[:10])
	}
	if len(token) != 67 { // "sk_" + 64 hex chars
		t.Errorf("Expected length 67, got %d", len(token))
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	t1, _ := GenerateToken()
	t2, _ := GenerateToken()
	if t1 == t2 {
		t.Error("Two generated tokens should not be equal")
	}
}

func TestHashToken(t *testing.T) {
	hash := HashToken("test-token")
	if len(hash) != 64 {
		t.Errorf("Expected 64 char SHA-256 hex, got %d chars", len(hash))
	}

	// Same input = same hash
	hash2 := HashToken("test-token")
	if hash != hash2 {
		t.Error("Same input should produce same hash")
	}

	// Different input = different hash
	hash3 := HashToken("other-token")
	if hash == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestRandomHex(t *testing.T) {
	hex := RandomHex(16)
	if len(hex) != 32 {
		t.Errorf("Expected 32 char hex for 16 bytes, got %d", len(hex))
	}
}
