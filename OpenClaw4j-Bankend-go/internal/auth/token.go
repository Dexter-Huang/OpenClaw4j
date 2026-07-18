package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TokenID(token string) string {
	hash := HashToken(token)
	return hash[:16]
}
