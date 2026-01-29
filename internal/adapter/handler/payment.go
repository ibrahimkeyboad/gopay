package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os" // Added to read environment variables
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
	"github.com/ibrahimkeyboad/gopay/internal/core/domain"
)

type PaymentHandler struct {
	Repo *storage.LedgerRepository
}

type ChargeRequest struct {
	CardNumber string `json:"card_number"`
	Expiry     string `json:"expiry"` // MM/YY
	CVC        string `json:"cvc"`
	Amount     int64  `json:"amount"` // Cents
	MerchantID string `json:"merchant_id"`
}

func (h *PaymentHandler) MakeCharge(c *fiber.Ctx) error {
	var req ChargeRequest
	if err := c.BodyParser(&req); err != nil {
		slog.Warn("Invalid card body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	// --- SECURITY CHECK START ---
	// Minimum 500 TZS (50,000 cents)
	const MinAmount = 500 * 100

	if req.Amount < MinAmount {
		slog.Warn("❌ Card Payment rejected: Amount too low", "amount", req.Amount)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Amount too low. Minimum is 500 TZS. Did you forget to multiply by 100?",
		})
	}
	// --- SECURITY CHECK END ---

	// 1. Validate Card Logic
	isValid, brand := domain.ValidateCard(req.CardNumber)
	if !isValid {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Card. We only accept Visa and Mastercard.",
		})
	}

	// 2. Validate Expiry/CVC (Simplified for now)
	if len(req.CVC) < 3 || len(req.Expiry) != 5 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid CVC or Expiry"})
	}

	// 3. "Process" the Payment (Simulate Bank Approval)
	merchantUUID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Merchant ID"})
	}

	// Call Deposit
	err = h.Repo.Deposit(c.Context(), merchantUUID, req.Amount, "Card Payment: "+string(brand))
	if err != nil {
		slog.Error("❌ Payment Processing Failed", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Payment Processing Failed"})
	}

	// 4. Queue Webhook Notification
	go func() {
		webhookPayload := map[string]interface{}{
			"event": "payment.succeeded",
			"data": map[string]interface{}{
				"amount":      req.Amount,
				"currency":    "TZS",
				"merchant_id": req.MerchantID,
				"card_brand":  brand,
				"status":      "COMPLETED",
				"timestamp":   time.Now(),
			},
		}

		payloadJSON, err := json.Marshal(webhookPayload)
		if err != nil {
			slog.Error("❌ Failed to marshal webhook payload", "error", err)
			return
		}

		// Get Webhook URL from Environment
		webhookURL := os.Getenv("WEBHOOK_URL")
		if webhookURL == "" {
			slog.Warn("⚠️ No WEBHOOK_URL found in .env, skipping webhook queue")
			return
		}

		// Queue the Webhook
		_, err = h.Repo.Db.Exec(context.Background(),
			"INSERT INTO webhook_jobs (url, payload) VALUES ($1, $2)",
			webhookURL, payloadJSON)

		if err != nil {
			slog.Error("❌ Webhook Queue Error", "error", err)
		} else {
			slog.Info("✅ Webhook queued for Worker!", "url", webhookURL)
		}
	}()

	// 5. Return Success Response
	return c.JSON(fiber.Map{
		"status":         "success",
		"message":        "Payment Approved",
		"brand":          brand,
		"amount_charged": req.Amount,
	})
}