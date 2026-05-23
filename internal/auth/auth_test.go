package auth

import (
	"testing"
	"time"
)

// ── JWTService tests ──────────────────────────────────────────────────────────

func TestGenerateAndValidateToken_RoundTrip(t *testing.T) {
	svc := NewJWTService("test-secret")
	token, err := svc.GenerateAccessToken("user-1", "tenant-1", "alice", string(RoleAdmin))
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want alice", claims.Username)
	}
	if claims.Role != string(RoleAdmin) {
		t.Errorf("Role = %q, want admin", claims.Role)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-a")
	svc2 := NewJWTService("secret-b")

	token, _ := svc1.GenerateAccessToken("u", "t", "u", string(RoleViewer))
	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	svc := NewJWTService("secret")
	_, err := svc.ValidateToken("this.is.not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestValidateToken_EmptyString(t *testing.T) {
	svc := NewJWTService("secret")
	_, err := svc.ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestGenerateToken_AllRoles(t *testing.T) {
	svc := NewJWTService("secret")
	for _, role := range []Role{RoleAdmin, RoleOperator, RoleViewer} {
		tok, err := svc.GenerateAccessToken("u", "t", "u", string(role))
		if err != nil {
			t.Fatalf("role %q: GenerateAccessToken error: %v", role, err)
		}
		claims, err := svc.ValidateToken(tok)
		if err != nil {
			t.Fatalf("role %q: ValidateToken error: %v", role, err)
		}
		if Role(claims.Role) != role {
			t.Errorf("role round-trip: got %q, want %q", claims.Role, role)
		}
	}
}

func TestAccessTokenTTL(t *testing.T) {
	if AccessTokenTTL() != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", AccessTokenTTL())
	}
}

func TestRefreshTokenTTL(t *testing.T) {
	if RefreshTokenTTL() != 7*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 168h", RefreshTokenTTL())
	}
}

// ── ValidRole tests ───────────────────────────────────────────────────────────

func TestValidRole(t *testing.T) {
	valid := []string{"super_admin", "admin", "operator", "viewer"}
	for _, r := range valid {
		if !ValidRole(r) {
			t.Errorf("ValidRole(%q) = false, want true", r)
		}
	}
	invalid := []string{"", "superuser", "root", "Admin"}
	for _, r := range invalid {
		if ValidRole(r) {
			t.Errorf("ValidRole(%q) = true, want false", r)
		}
	}
}

// ── Password tests ────────────────────────────────────────────────────────────

func TestHashAndCheckPassword_Correct(t *testing.T) {
	hash, err := HashPassword("my-secret-pass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := CheckPassword(hash, "my-secret-pass"); err != nil {
		t.Errorf("CheckPassword correct: %v", err)
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, _ := HashPassword("correct-pass")
	if err := CheckPassword(hash, "wrong-pass"); err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestHashPassword_DifferentHashesSameInput(t *testing.T) {
	// bcrypt generates different salts each time.
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Error("expected different hashes for same input (bcrypt should use different salts)")
	}
}

func TestHashPassword_EmptyString(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword empty: %v", err)
	}
	if err := CheckPassword(hash, ""); err != nil {
		t.Errorf("CheckPassword empty correct: %v", err)
	}
}

// ── GenerateRefreshToken tests ────────────────────────────────────────────────

func TestGenerateRefreshToken_UniqueTokens(t *testing.T) {
	raw1, hash1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	raw2, hash2, _ := GenerateRefreshToken()

	if raw1 == raw2 {
		t.Error("expected unique raw tokens")
	}
	if hash1 == hash2 {
		t.Error("expected unique token hashes")
	}
}

func TestGenerateRefreshToken_HashMatchesRaw(t *testing.T) {
	raw, storedHash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	// Re-hash the raw token and check it matches the stored hash.
	computedHash := HashToken(raw)
	if computedHash != storedHash {
		t.Errorf("hash mismatch: computed %q, stored %q", computedHash, storedHash)
	}
}

func TestGenerateRefreshToken_Length(t *testing.T) {
	// 32 random bytes → 64 hex chars.
	raw, _, _ := GenerateRefreshToken()
	if len(raw) != 64 {
		t.Errorf("raw token length = %d, want 64 hex chars", len(raw))
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := HashToken("some-token")
	h2 := HashToken("some-token")
	if h1 != h2 {
		t.Error("HashToken is not deterministic")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	if HashToken("a") == HashToken("b") {
		t.Error("different inputs produced the same hash")
	}
}
