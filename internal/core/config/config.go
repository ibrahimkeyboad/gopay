package config

import (
	"log/slog" // Use the new structured logger
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	WebhookURL  string
	Env         string
}

// LoadConfig reads .env file and returns a Config struct
func LoadConfig() *Config {
	// Try loading .env file (it might not exist in Production, which is fine)
	err := godotenv.Load()
	if err != nil {
		// We use Warn because it's not a crash, but it's worth noting
		slog.Warn("No .env file found, relying on System Env Variables")
	}

	return &Config{
		Port:        getEnv("PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		WebhookURL:  getEnv("WEBHOOK_URL", ""),
		Env:         getEnv("ENV", "development"),
	}
}

// Helper to get env with a default fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}