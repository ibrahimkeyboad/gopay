package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	// Ensure this path matches your project structure
	"github.com/ibrahimkeyboad/gopay/internal/core/notifications"
)

func StartWebhookWorker(db *pgxpool.Pool) {
	go func() {
		slog.Info("👷 Webhook Worker started")
		for {
			processJobs(db)
			time.Sleep(5 * time.Second)
		}
	}()
}

func processJobs(db *pgxpool.Pool) {
	ctx := context.Background()

	query := `
		SELECT id, url, payload, attempts 
		FROM webhook_jobs 
		WHERE status = 'PENDING' AND next_run_at <= NOW() 
		ORDER BY created_at ASC 
		LIMIT 1 
		FOR UPDATE SKIP LOCKED
	`

	var id string
	var url string
	var payloadBytes []byte
	var attempts int

	err := db.QueryRow(ctx, query).Scan(&id, &url, &payloadBytes, &attempts)
	if err != nil {
		return
	}

	var payload interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		slog.Error("Worker: Failed to parse payload", "error", err, "job_id", id)
		db.Exec(ctx, "UPDATE webhook_jobs SET status = 'FAILED' WHERE id = $1", id)
		return
	}

	slog.Info("Worker: Processing job", "url", url, "job_id", id)

	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		slog.Warn("⚠️ WEBHOOK_SECRET is missing in .env, using default insecure key")
		secret = "default_insecure_key"
	}

	sendErr := notifications.SendWebhook(url, payload, secret)

	if sendErr != nil {
		slog.Error("Worker: Webhook failed", "error", sendErr, "attempts", attempts)
		nextRun := time.Now().Add(time.Duration(attempts*10+10) * time.Second)

		if attempts >= 5 {
			db.Exec(ctx, "UPDATE webhook_jobs SET status = 'FAILED' WHERE id = $1", id)
			slog.Error("Worker: Job marked as FAILED (Max attempts reached)", "job_id", id)
		} else {
			db.Exec(ctx, "UPDATE webhook_jobs SET status = 'PENDING', attempts = attempts + 1, next_run_at = $2 WHERE id = $1", id, nextRun)
			slog.Info("Worker: Scheduled retry", "next_run", nextRun)
		}
	} else {
		slog.Info("✅ Worker: Webhook Sent Successfully!", "job_id", id)
		db.Exec(ctx, "UPDATE webhook_jobs SET status = 'COMPLETED' WHERE id = $1", id)
	}
}