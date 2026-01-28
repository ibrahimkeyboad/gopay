package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectDB initializes the connection pool to Neon
func ConnectDB() (*pgxpool.Pool, error) {
	// 1. Get URL from .env
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	// 2. Parse Config
	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// 3. Configure Pool Settings (Crucial for Serverless)
	// Neon creates connections fast, but we don't want to hold too many idle ones.
	config.MaxConns = 10           // Max 10 simultaneous connections
	config.MinConns = 0            // Allow scaling to zero
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// 4. Connect
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// 5. Test Connection (Ping)
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	fmt.Println("✅ Successfully connected to Neon Postgres!")
	return pool, nil
}