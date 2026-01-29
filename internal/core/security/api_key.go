package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey creates a secure random API key and its SHA256 hash.
//
// Returns:
//   - realKey: The actual API key to show the user (e.g., "sk_live_abc123...")
//   - keyHash: SHA256 hash to store in the database
//   - error: Any error during random byte generation
//
// Example:
//   realKey, keyHash, err := GenerateAPIKey()

func GenerateAPIKey() (string, string, error) {
	// 1. Generate 32 random bytes using crypto/rand (cryptographically secure)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 2. Convert to hexadecimal string (64 characters)
	randomString := hex.EncodeToString(bytes)

	// 3. Add prefix (similar to Stripe's API key format)
// GOOD (Use your own custom prefix, e.g., 'gp' for GoPay)
realKey := fmt.Sprintf("gp_live_%s", randomString)

	// 4. Hash the key with SHA256 - this is what we store in the database
	hash := sha256.Sum256([]byte(realKey))
	keyHash := hex.EncodeToString(hash[:])

	return realKey, keyHash, nil
}

// ValidateKey checks if a provided API key matches the stored hash.
//
// Parameters:
//   - providedKey: The raw API key from the user's request
//   - storedHash: The SHA256 hash stored in the database
//
// Returns:
//   - true if the key is valid (hash matches)
//   - false if the key is invalid or has been tampered with
//
// Example:
//   isValid := ValidateKey("sk_live_abc123...", "b94d27b9934d3e08...")
func ValidateKey(providedKey, storedHash string) bool {
	// Hash the provided key
	hash := sha256.Sum256([]byte(providedKey))
	computedHash := hex.EncodeToString(hash[:])

	// Compare with stored hash (constant-time comparison would be better for production)
	return computedHash == storedHash
}