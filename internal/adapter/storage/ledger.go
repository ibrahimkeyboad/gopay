package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerRepository struct {
	// CHANGE IS HERE: We changed 'db' to 'Db' (Capital D makes it public)
	Db *pgxpool.Pool 
}

func NewLedgerRepository(db *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{Db: db}
}

// Deposit adds money to an account
func (r *LedgerRepository) Deposit(ctx context.Context, accountID uuid.UUID, amount int64, description string) error {
	// UPDATE HERE: Use r.Db instead of r.db
	tx, err := r.Db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var transactionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (amount, currency, description, status)
		VALUES ($1, 'TZS', $2, 'COMPLETED') RETURNING id`, amount, description).Scan(&transactionID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, accountID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount)
		VALUES ($1, $2, 'CREDIT', $3)`, transactionID, accountID, amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Transfer moves money safely
func (r *LedgerRepository) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64) error {
	// UPDATE HERE: Use r.Db instead of r.db
	tx, err := r.Db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var balance int64
	err = tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, fromID).Scan(&balance)
	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("insufficient funds: you have %d but tried to send %d", balance, amount)
	}

	var transactionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (amount, currency, description, status)
		VALUES ($1, 'TZS', 'P2P Transfer', 'COMPLETED') RETURNING id`, amount).Scan(&transactionID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID); err != nil {
		return err
	}
	
	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, toID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO entries (transaction_id, account_id, direction, amount) VALUES ($1, $2, 'DEBIT', $3)`, transactionID, fromID, amount); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO entries (transaction_id, account_id, direction, amount) VALUES ($1, $2, 'CREDIT', $3)`, transactionID, toID, amount); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetHistory fetches the last 10 transactions
func (r *LedgerRepository) GetHistory(ctx context.Context, accountID uuid.UUID) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			t.id, t.amount, t.currency, t.description, t.status, t.created_at, e.direction
		FROM entries e
		JOIN transactions t ON e.transaction_id = t.id
		WHERE e.account_id = $1
		ORDER BY t.created_at DESC
		LIMIT 10
	`

	// UPDATE HERE: Use r.Db instead of r.db
	rows, err := r.Db.Query(ctx, query, accountID)
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
			"direction":   direction,
			"date":        createdAt,
		})
	}

	return history, nil
}