package storage

import (
	"context"
	"fmt"
	"log/slog" // Use the new logger!
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectDB initializes the connection pool
// CHANGE: We now pass 'dsn' (databaseUrl) as an argument
func ConnectDB(dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database URL is empty")
	}

	// 1. Parse Config
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// 2. Configure Pool Settings (Optimized for Neon/Serverless)
	config.MaxConns = 10
	config.MinConns = 0
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// 3. Connect
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// 4. Test Connection (Ping)
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

    // Use structured logging
	slog.Info("✅ Successfully connected to Neon Postgres!")
	return pool, nil
}