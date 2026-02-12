package jwtManager

import "testing"

func TestGenerateAndVerifyToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")

	token, err := GenerateToken("u-1", "tester")
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	parsed, claims, err := VerificateToken(token)
	if err != nil {
		t.Fatalf("token parse failed: %v", err)
	}
	if !parsed.Valid {
		t.Fatalf("expected token to be valid")
	}
	if claims.UserID != "u-1" {
		t.Fatalf("expected user id u-1 got %s", claims.UserID)
	}
}
