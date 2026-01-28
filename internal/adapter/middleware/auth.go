package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Protected(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Get Token from Header
		authHeader := c.Get("Authorization") // "Bearer sk_live_..."
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Missing API Key"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Header Format"})
		}
		apiKey := parts[1]

		// 2. Hash the key (We never compare plain text!)
		hash := sha256.Sum256([]byte(apiKey))
		hashedKey := hex.EncodeToString(hash[:])

		// 3. Check DB
		var accountID string
		err := db.QueryRow(c.Context(), "SELECT account_id FROM api_keys WHERE key_hash = $1", hashedKey).Scan(&accountID)
		
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API Key"})
		}

		// 4. Save Account ID to Context (So handler knows who is calling)
		c.Locals("merchant_id", accountID)

		return c.Next()
	}
}