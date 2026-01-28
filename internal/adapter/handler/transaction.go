package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
)

type TransactionHandler struct {
	Repo *storage.LedgerRepository
}

// Request Models
type DepositRequest struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"` // Cents!
}

type TransferRequest struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
	Amount int64  `json:"amount"`
}

// Deposit API
func (h *TransactionHandler) Deposit(c *fiber.Ctx) error {
	var req DepositRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}

	accUUID, _ := uuid.Parse(req.AccountID)
	err := h.Repo.Deposit(c.Context(), accUUID, req.Amount, "Manual Deposit")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success", "message": "Money Deposited!"})
}

// Transfer API
func (h *TransactionHandler) Transfer(c *fiber.Ctx) error {
	var req TransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}

	fromUUID, _ := uuid.Parse(req.FromID)
	toUUID, _ := uuid.Parse(req.ToID)

	err := h.Repo.Transfer(c.Context(), fromUUID, toUUID, req.Amount)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success", "message": "Transfer Complete!"})
}