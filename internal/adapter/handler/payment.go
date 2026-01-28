package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
	"github.com/ibrahimkeyboad/gopay/internal/core/domain"
	"github.com/ibrahimkeyboad/gopay/internal/core/notifications"
)

type PaymentHandler struct {
	Repo *storage.LedgerRepository
}

type ChargeRequest struct {
	CardNumber string `json:"card_number"`
	Expiry     string `json:"expiry"` // MM/YY
	CVC        string `json:"cvc"`
	Amount     int64  `json:"amount"`      // Cents
	MerchantID string `json:"merchant_id"`
}

func (h *PaymentHandler) MakeCharge(c *fiber.Ctx) error {
	var req ChargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	// 1. Validate Card Logic
	isValid, brand := domain.ValidateCard(req.CardNumber)
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Card. We only accept Visa and Mastercard.",
		})
	}

	// 2. Validate Expiry/CVC (Simplified for now)
	if len(req.CVC) < 3 || len(req.Expiry) != 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid CVC or Expiry"})
	}

	// 3. "Process" the Payment (Simulate Bank Approval)
	// Since we are the bank, we just Deposit the money into the Merchant's account.
	merchantUUID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Merchant ID"})
	}

	// We use "Deposit" because money is entering the system from the outside world (The Credit Card)
	err = h.Repo.Deposit(c.Context(), merchantUUID, req.Amount, "Card Payment: "+string(brand))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Payment Processing Failed"})
	}

	// 4. Send Webhook Notification (Asynchronously)
	// In a real app, you would look up the merchant's webhook URL from the database
	go func() {
		webhookPayload := map[string]interface{}{
			"event": "payment.succeeded",
			"data": map[string]interface{}{
				"amount":      req.Amount,
				"currency":    "TZS",
				"merchant_id": req.MerchantID,
				"card_brand":  brand, // Visa/Mastercard
				"timestamp":   time.Now(),
			},
		}
		
		// For testing: Use webhook.site to see webhooks in real-time
		// Replace YOUR-UNIQUE-ID with your actual webhook.site ID
		// In production, fetch this from database: merchant.WebhookURL
		testURL := "https://webhook.site/6ace6744-c47c-4b46-a875-68132e6a65eb"
		
		err := notifications.SendWebhook(testURL, webhookPayload)
		if err != nil {
			fmt.Println("❌ Webhook failed:", err)
		} else {
			fmt.Println("✅ Webhook sent successfully!")
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