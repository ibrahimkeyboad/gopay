package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
)

type MobileMoneyHandler struct {
	Repo *storage.LedgerRepository
}

type MobilePayRequest struct {
	PhoneNumber string `json:"phone_number"`
	Provider    string `json:"provider"`
	Amount      int64  `json:"amount"`
	MerchantID  string `json:"merchant_id"`
}

func (h *MobileMoneyHandler) InitializePayment(c *fiber.Ctx) error {
	var req MobilePayRequest
	if err := c.BodyParser(&req); err != nil {
		slog.Warn("Invalid payment body received", "error", err) // <--- Warn Log
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}


	// 1. SAFETY CHECK: Minimum Transaction Amount
    // We set the minimum to 500 TZS (which is 50,000 cents)
    const MinAmount = 500 * 100

		slog.Info("Payment initialization request",
			"sms_phone", req.PhoneNumber,
			"provider", req.Provider,
			"amount", req.Amount,
			"merchant_id", req.MerchantID,
		)

		if req.Amount < MinAmount {
        slog.Warn("❌ Payment rejected: Amount too low", "amount", req.Amount)
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Amount too low. Minimum is 500 TZS. Did you forget to multiply by 100?",
        })
    }


	// Validate Phone Number
	if len(req.PhoneNumber) < 10 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Phone Number"})
	}

	// Validate Merchant ID
	merchantUUID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Merchant UUID"})
	}

	// Start the Background Process
	go func() {
	// Log Context: We can attach data to the log
		logAttrs := []any{
			slog.String("phone", req.PhoneNumber),
			slog.String("merchant_id", req.MerchantID),
			slog.Int64("amount", req.Amount),
			slog.String("provider", req.Provider),
		}

		slog.Info("📲 [M-PESA] USSD Push initiated", logAttrs...)

		// Simulate User Delay (waiting for PIN entry)
		time.Sleep(5 * time.Second)

		// 80% Success Rate simulation
		if rand.Float32() < 0.8 {
		slog.Info("✅ [M-PESA] User entered PIN. Processing deposit...", logAttrs...)

			// 1. Update Ledger
			err := h.Repo.Deposit(context.Background(), merchantUUID, req.Amount, "M-Pesa Payment: "+req.PhoneNumber)
			if err != nil {
	slog.Error("❌ [M-PESA] Ledger Deposit Failed", "error", err, "phone", req.PhoneNumber)
				return
			}
			slog.Info("💰 [M-PESA] Money deposited in DB!", logAttrs...)

			// 2. Queue Webhook for Background Worker
			webhookPayload := map[string]interface{}{
				"event": "payment.succeeded",
				"data": map[string]interface{}{
					"amount":       req.Amount,
					"currency":     "TZS",
					"merchant_id":  req.MerchantID,
					"phone_number": req.PhoneNumber,
					"provider":     req.Provider,
					"status":       "COMPLETED",
					"timestamp":    time.Now(),
				},
			}

			// Convert payload to JSON
			payloadJSON, err := json.Marshal(webhookPayload)
			if err != nil {
				slog.Error("❌ [M-PESA] Failed to marshal webhook payload", "error", err)
				return
			}

			// Queue the webhook job
			// REPLACE YOUR-UNIQUE-ID with your webhook.site ID
			webhookURL := "https://webhook.site/6ace6744-c47c-4b46-a875-68132e6a65eb"

			jobQuery := `INSERT INTO webhook_jobs (url, payload) VALUES ($1, $2)`
			
			// --- FIX IS HERE: Changed 'db' to 'Db' ---
			_, err = h.Repo.Db.Exec(context.Background(), jobQuery, webhookURL, payloadJSON)
			
			if err != nil {
			slog.Error("❌ [M-PESA] Webhook Queue Error", "error", err)
			} else {
			slog.Info("✅ [M-PESA] Webhook queued for Worker!")
			}

		} else {
		slog.Warn("⚠️ [M-PESA] User cancelled or timed out", "phone", req.PhoneNumber)

			// Optional: Queue a webhook for failed payment
			failedPayload := map[string]interface{}{
				"event": "payment.failed",
				"data": map[string]interface{}{
					"amount":       req.Amount,
					"merchant_id":  req.MerchantID,
					"phone_number": req.PhoneNumber,
					"provider":     req.Provider,
					"reason":       "User cancelled or timeout",
					"timestamp":    time.Now(),
				},
			}

			payloadJSON, _ := json.Marshal(failedPayload)
			webhookURL := "https://webhook.site/6ace6744-c47c-4b46-a875-68132e6a65eb"

			// This one was already correct, but just to be safe:
			_, _ = h.Repo.Db.Exec(context.Background(),
				`INSERT INTO webhook_jobs (url, payload) VALUES ($1, $2)`,
				webhookURL, payloadJSON)
		}
	}()

	// Return immediately with pending status
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status":   "pending",
		"message":  "USSD Push sent. Check your phone.",
		"provider": req.Provider,
	})
}