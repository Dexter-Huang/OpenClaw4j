package auth

import "testing"

func TestHashPasswordProducesVerifiableJavaCompatibleArgon2ID(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatalf("generated password hash was not verifiable: %q", hash)
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("wrong password unexpectedly verified")
	}
}
