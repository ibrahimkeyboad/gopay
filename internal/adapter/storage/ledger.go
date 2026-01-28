package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerRepository struct {
	db *pgxpool.Pool
}

func NewLedgerRepository(db *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{db: db}
}

// Deposit adds money to an account (Simulating a top-up from a Bank/Stripe)
func (r *LedgerRepository) Deposit(ctx context.Context, accountID uuid.UUID, amount int64, description string) error {
	tx, err := r.db.Begin(ctx) // Start Transaction
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // Safety: Undo everything if we don't commit

	// 1. Create Transaction Record
	var transactionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (amount, currency, description, status)
		VALUES ($1, 'TZS', $2, 'COMPLETED') RETURNING id`, amount, description).Scan(&transactionID)
	if err != nil {
		return err
	}

	// 2. Add Money to Account
	_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, accountID)
	if err != nil {
		return err
	}

	// 3. Create Ledger Entry (Credit)
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount)
		VALUES ($1, $2, 'CREDIT', $3)`, transactionID, accountID, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx) // Save changes
}

// Transfer moves money safely between two accounts
func (r *LedgerRepository) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Lock Sender Account & Check Balance
	var balance int64
	err = tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, fromID).Scan(&balance)
	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("insufficient funds: you have %d but tried to send %d", balance, amount)
	}

	// 2. Create Transaction Record
	var transactionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (amount, currency, description, status)
		VALUES ($1, 'TZS', 'P2P Transfer', 'COMPLETED') RETURNING id`, amount).Scan(&transactionID)
	if err != nil {
		return err
	}

	// 3. Deduct from Sender
	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID); err != nil {
		return err
	}
	
	// 4. Add to Receiver
	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, toID); err != nil {
		return err
	}

	// 5. Create Entries (Debit Sender, Credit Receiver)
	if _, err := tx.Exec(ctx, `INSERT INTO entries (transaction_id, account_id, direction, amount) VALUES ($1, $2, 'DEBIT', $3)`, transactionID, fromID, amount); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO entries (transaction_id, account_id, direction, amount) VALUES ($1, $2, 'CREDIT', $3)`, transactionID, toID, amount); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetHistory fetches the last 10 transactions for an account
func (r *LedgerRepository) GetHistory(ctx context.Context, accountID uuid.UUID) ([]map[string]interface{}, error) {
	// We join 'entries' with 'transactions' to get the full details
	query := `
		SELECT 
			t.id, 
			t.amount, 
			t.currency, 
			t.description, 
			t.status, 
			t.created_at,
			e.direction -- Was this a CREDIT (In) or DEBIT (Out)?
		FROM entries e
		JOIN transactions t ON e.transaction_id = t.id
		WHERE e.account_id = $1
		ORDER BY t.created_at DESC
		LIMIT 10
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}

	for rows.Next() {
		var id uuid.UUID
		var amount int64
		var currency, description, status, direction string
		var createdAt time.Time

		rows.Scan(&id, &amount, &currency, &description, &status, &createdAt, &direction)

		history = append(history, map[string]interface{}{
			"id":          id,
			"amount":      amount,
			"currency":    currency,
			"description": description,
			"status":      status,
			"direction":   direction, // "CREDIT" or "DEBIT"
			"date":        createdAt,
		})
	}

	return history, nil
}