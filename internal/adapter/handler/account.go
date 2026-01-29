package handler

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"

	// FIX: This import was missing!
	"github.com/ibrahimkeyboad/gopay/internal/core/security"
)

type AccountHandler struct {
	Repo *storage.AccountRepository
}

// CreateAccountRequest defines what the user sends us
type CreateAccountRequest struct {
	OwnerName string `json:"owner_name"`
	Currency  string `json:"currency"`
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	var req CreateAccountRequest

	// 1. Parse JSON
	if err := c.BodyParser(&req); err != nil {
		slog.Warn("Invalid account body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// 2. Validate Input
	if req.OwnerName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Owner Name is required"})
	}

	validCurrencies := map[string]bool{"USD": true, "TZS": true}
	if !validCurrencies[req.Currency] {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid currency. Use USD or TZS"})
	}

	// 3. Call Storage
	account, err := h.Repo.CreateAccount(c.Context(), req.OwnerName, req.Currency)
	if err != nil {
		slog.Error("Failed to create account", "error", err, "owner", req.OwnerName)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create account"})
	}

	slog.Info("✅ Account Created", "id", account.ID, "owner", req.OwnerName)

	// 4. Return Success
	return c.Status(http.StatusCreated).JSON(account)
}

func (h *AccountHandler) GenerateKey(c *fiber.Ctx) error {
	accountIDParam := c.Params("id")

	// 1. Convert string ID to UUID
	accountUUID, err := uuid.Parse(accountIDParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Account ID format"})
	}

	// 2. Generate Secure Key
	realKey, keyHash, err := security.GenerateAPIKey()
	if err != nil {
		slog.Error("Crypto error generating key", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Crypto error"})
	}

	// 3. Save Hash to DB
	err = h.Repo.SaveAPIKey(c.Context(), accountUUID, keyHash, "sk_live_")
	if err != nil {
		slog.Error("Failed to save API key", "error", err, "account_id", accountUUID)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save key"})
	}

	slog.Info("🔑 API Key Generated", "account_id", accountUUID)

	// 4. Show Key to User (ONCE ONLY)
	return c.JSON(fiber.Map{
		"api_key": realKey,
		"warning": "Save this now! We won't show it again.",
	})
}