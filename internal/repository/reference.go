package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hesabFun/ledger/internal/db"
)

// AccountType represents an account type entity
type AccountType struct {
	ID            int32
	Code          string
	Name          string
	NormalBalance string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Currency represents a currency entity
type Currency struct {
	ID        int32
	Code      string
	Name      string
	Symbol    string
	Precision int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReferenceRepository handles reference data database operations
type ReferenceRepository struct {
	db *db.DB
}

// NewReferenceRepository creates a new reference repository
func NewReferenceRepository(database *db.DB) *ReferenceRepository {
	return &ReferenceRepository{db: database}
}

// ListAccountTypes retrieves all account types
func (r *ReferenceRepository) ListAccountTypes(ctx context.Context) ([]*AccountType, error) {
	query := `
		SELECT id, code, name, normal_balance, created_at, updated_at
		FROM account_types
		ORDER BY id
	`

	rows, err := r.db.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list account types: %w", err)
	}
	defer rows.Close()

	accountTypes := make([]*AccountType, 0)
	for rows.Next() {
		accountType := &AccountType{}
		err := rows.Scan(
			&accountType.ID,
			&accountType.Code,
			&accountType.Name,
			&accountType.NormalBalance,
			&accountType.CreatedAt,
			&accountType.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account type: %w", err)
		}
		accountTypes = append(accountTypes, accountType)
	}

	return accountTypes, nil
}

// ListCurrencies retrieves all currencies
func (r *ReferenceRepository) ListCurrencies(ctx context.Context) ([]*Currency, error) {
	query := `
		SELECT id, code, name, symbol, precision, created_at, updated_at
		FROM currencies
		ORDER BY code
	`

	rows, err := r.db.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}
	defer rows.Close()

	currencies := make([]*Currency, 0)
	for rows.Next() {
		currency := &Currency{}
		err := rows.Scan(
			&currency.ID,
			&currency.Code,
			&currency.Name,
			&currency.Symbol,
			&currency.Precision,
			&currency.CreatedAt,
			&currency.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan currency: %w", err)
		}
		currencies = append(currencies, currency)
	}

	return currencies, nil
}
