package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account represents a user's wallet or a merchant's vault
type Account struct {
	ID        uuid.UUID
	OwnerName string
	Balance   int64 // Stored in minor units (cents)
	Currency  Currency
	CreatedAt time.Time
}

// Transaction represents a completed movement of money
type Transaction struct {
	ID            uuid.UUID
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        int64
	Currency      Currency
	Status        string // "PENDING", "COMPLETED", "FAILED"
	Description   string
	CreatedAt     time.Time
}