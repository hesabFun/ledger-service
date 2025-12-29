package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Account represents an account entity
type Account struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	AccountNumber   string
	Name            string
	Description     *string
	AccountTypeID   int32
	CurrencyCode    string
	ParentAccountID *uuid.UUID
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AccountBalance represents account balance entity
type AccountBalance struct {
	AccountID     uuid.UUID
	DebitBalance  decimal.Decimal
	CreditBalance decimal.Decimal
	UpdatedAt     time.Time
}

// CreateAccountParams holds parameters for creating an account
type CreateAccountParams struct {
	AccountNumber   string
	Name            string
	Description     *string
	AccountTypeID   int32
	CurrencyCode    string
	ParentAccountID *uuid.UUID
}

// AccountRepository handles account database operations
type AccountRepository struct {
	db *db.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(database *db.DB) *AccountRepository {
	return &AccountRepository{db: database}
}

// Create creates a new account using the database function
func (r *AccountRepository) Create(ctx context.Context, tenantID uuid.UUID, params CreateAccountParams) (*Account, error) {
	// Start a transaction with tenant context
	tx, err := r.db.BeginTx(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var accountID uuid.UUID
	query := "SELECT create_account($1, $2, $3, $4, $5, $6)"

	err = tx.QueryRow(ctx, query,
		params.AccountNumber,
		params.Name,
		params.AccountTypeID,
		params.CurrencyCode,
		params.Description,
		params.ParentAccountID,
	).Scan(&accountID)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch the created account details
	return r.GetByID(ctx, tenantID, accountID)
}

// GetByID retrieves an account by ID with tenant context
func (r *AccountRepository) GetByID(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*Account, error) {
	_, conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	defer conn.Release()

	account := &Account{}
	query := `
		SELECT id, tenant_id, account_number, name, description, account_type_id,
		       currency_code, parent_account_id, is_active, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	err = conn.QueryRow(ctx, query, accountID).Scan(
		&account.ID,
		&account.TenantID,
		&account.AccountNumber,
		&account.Name,
		&account.Description,
		&account.AccountTypeID,
		&account.CurrencyCode,
		&account.ParentAccountID,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// List retrieves accounts with optional filters
func (r *AccountRepository) List(ctx context.Context, tenantID uuid.UUID, accountTypeID *int32, currencyCode *string, limit, offset int) ([]*Account, int, error) {
	_, conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to set tenant context: %w", err)
	}
	defer conn.Release()

	// Build query with filters
	query := `
		SELECT id, tenant_id, account_number, name, description, account_type_id,
		       currency_code, parent_account_id, is_active, created_at, updated_at
		FROM accounts
		WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM accounts WHERE 1=1"
	var args []interface{}
	argCount := 0

	if accountTypeID != nil {
		argCount++
		query += fmt.Sprintf(" AND account_type_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND account_type_id = $%d", argCount)
		args = append(args, *accountTypeID)
	}

	if currencyCode != nil {
		argCount++
		query += fmt.Sprintf(" AND currency_code = $%d", argCount)
		countQuery += fmt.Sprintf(" AND currency_code = $%d", argCount)
		args = append(args, *currencyCode)
	}

	// Get total count
	var totalCount int
	err = conn.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	// Add pagination
	argCount++
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*Account, 0)
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.ID,
			&account.TenantID,
			&account.AccountNumber,
			&account.Name,
			&account.Description,
			&account.AccountTypeID,
			&account.CurrencyCode,
			&account.ParentAccountID,
			&account.IsActive,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, totalCount, nil
}

// GetBalance retrieves the balance for an account
func (r *AccountRepository) GetBalance(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*AccountBalance, error) {
	_, conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	defer conn.Release()

	balance := &AccountBalance{AccountID: accountID}
	query := `
		SELECT debit_balance, credit_balance, updated_at
		FROM account_balances
		WHERE account_id = $1
	`

	err = conn.QueryRow(ctx, query, accountID).Scan(
		&balance.DebitBalance,
		&balance.CreditBalance,
		&balance.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("balance not found for account")
		}
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	return balance, nil
}
