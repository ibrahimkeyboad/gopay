package domain

import (
	"errors"
	"fmt"
)

type Currency string

const (
	USD Currency = "USD"
	TZS Currency = "TZS"
)

// Money struct holds amount in "minor units" (cents)
// Example: 1000 TZS is stored as 1000. $10.50 is stored as 1050.
type Money struct {
	Amount   int64
	Currency Currency
}

// NewMoney creates a new Money instance
func NewMoney(amount int64, currency Currency) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// Add adds two Money instances safely
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: cannot add %s to %s", other.Currency, m.Currency)
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: m.Currency,
	}, nil
}

// Subtract subtracts Money safely
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, errors.New("currency mismatch")
	}
	if m.Amount < other.Amount {
		return Money{}, errors.New("insufficient balance")
	}
	return Money{
		Amount:   m.Amount - other.Amount,
		Currency: m.Currency,
	}, nil
}