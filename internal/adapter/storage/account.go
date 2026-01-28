package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository struct {
	db *pgxpool.Pool
}

func NewAccountRepository(db *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{db: db}
}

// Account Model
type Account struct {
	ID        uuid.UUID `json:"id"`
	OwnerName string    `json:"owner_name"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAccount
func (r *AccountRepository) CreateAccount(ctx context.Context, ownerName string, currency string) (*Account, error) {
	query := `
		INSERT INTO accounts (owner_name, currency, balance)
		VALUES ($1, $2, 0)
		RETURNING id, owner_name, balance, currency, created_at
	`
	var acc Account
	err := r.db.QueryRow(ctx, query, ownerName, currency).Scan(
		&acc.ID, &acc.OwnerName, &acc.Balance, &acc.Currency, &acc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}
	return &acc, nil
}

// GetAccountByID
func (r *AccountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	query := `SELECT id, owner_name, balance, currency, created_at FROM accounts WHERE id = $1`
	var acc Account
	err := r.db.QueryRow(ctx, query, id).Scan(
		&acc.ID, &acc.OwnerName, &acc.Balance, &acc.Currency, &acc.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// --- THIS IS THE MISSING PART ---
// SaveAPIKey stores the hashed key for the user
func (r *AccountRepository) SaveAPIKey(ctx context.Context, accountID uuid.UUID, keyHash string, keyPrefix string) error {
	query := `INSERT INTO api_keys (account_id, key_hash, key_prefix) VALUES ($1, $2, $3)`
	
	_, err := r.db.Exec(ctx, query, accountID, keyHash, keyPrefix)
	if err != nil {
		return fmt.Errorf("failed to save api key: %w", err)
	}
	return nil
}