package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey creates a new key and its hash
// Returns: (realKey, hash)
// Example: ("sk_live_abc123", "a665a4592...")
func GenerateAPIKey() (string, string, error) {
	// 1. Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	// 2. Convert to Hex string
	randomString := hex.EncodeToString(bytes)

	// 3. Add Prefix (Like Stripe)
	realKey := fmt.Sprintf("sk_live_%s", randomString)

	// 4. Hash it (SHA256) - This is what we save to DB
	hash := sha256.Sum256([]byte(realKey))
	hashedKey := hex.EncodeToString(hash[:])

	return realKey, hashedKey, nil
}

// ValidateKey checks if the user provided key matches the hash
func ValidateKey(providedKey, storedHash string) bool {
	hash := sha256.Sum256([]byte(providedKey))
	computedHash := hex.EncodeToString(hash[:])
	return computedHash == storedHash
}