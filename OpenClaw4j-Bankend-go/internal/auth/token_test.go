package auth

import "testing"

func TestHashTokenIsStableAndDoesNotExposePlaintext(t *testing.T) {
	token := "oc_access_secret"
	first := HashToken(token)
	second := HashToken(token)
	if first != second {
		t.Fatal("expected stable token hash")
	}
	if first == token || len(first) != 64 {
		t.Fatalf("unexpected hash: %q", first)
	}
}

func TestTokenIDUsesHashPrefix(t *testing.T) {
	id := TokenID("oc_access_secret")
	if len(id) != 16 {
		t.Fatalf("expected 16-char token id, got %q", id)
	}
}
