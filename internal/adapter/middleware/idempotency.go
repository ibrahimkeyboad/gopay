package middleware

import (
	"log/slog" // Use the new logger

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Idempotency(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Get Key from Header
		key := c.Get("Idempotency-Key")

		// If no key, skip (silently, or you can log at Debug level)
		if key == "" {
			return c.Next()
		}

		// 2. Check if key exists
		var status int
		var body []byte
		err := db.QueryRow(c.Context(),
			"SELECT response_status, response_body FROM idempotency_keys WHERE key_id = $1",
			key).Scan(&status, &body)

		if err == nil {
			slog.Info("🛑 Idempotency Hit! Returning cached response", "key", key)
			c.Set("X-Idempotency-Hit", "true")
			c.Set("Content-Type", "application/json")
			return c.Status(status).Send(body)
		}

		// 3. Run the Handler
		err = c.Next()
		if err != nil {
			return err
		}

		// 4. Save the Result
		resStatus := c.Response().StatusCode()
		resBody := c.Response().Body() // Copy the response body

		_, insertErr := db.Exec(c.Context(),
			"INSERT INTO idempotency_keys (key_id, response_status, response_body) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			key, resStatus, resBody)

		if insertErr != nil {
			slog.Error("❌ Failed to save Idempotency Key", "error", insertErr, "key", key)
		} else {
			slog.Info("💾 Idempotency Key Saved", "key", key)
		}

		return nil
	}
}