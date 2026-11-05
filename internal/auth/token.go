package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// tokenBytes is the session token length: 32 bytes from crypto/rand
// (Web-App Development Stack v2.1, Section 10.2).
const tokenBytes = 32

// NewToken returns a random URL-safe token and the SHA-256 hash that is stored
// in the database. The token itself is never stored.
func NewToken() (token string, hash []byte, err error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

// HashToken returns the SHA-256 hash of token.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
