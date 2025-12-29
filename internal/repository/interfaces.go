package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantRepositoryInterface defines methods for tenant operations
type TenantRepositoryInterface interface {
	Create(ctx context.Context, name string) (*Tenant, error)
	GetByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error)
	GetByName(ctx context.Context, name string) (*Tenant, error)
}

// AccountRepositoryInterface defines methods for account operations
type AccountRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, params CreateAccountParams) (*Account, error)
	GetByID(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*Account, error)
	List(ctx context.Context, tenantID uuid.UUID, accountTypeID *int32, currencyCode *string, limit, offset int) ([]*Account, int, error)
	GetBalance(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*AccountBalance, error)
}

// JournalRepositoryInterface defines methods for journal entry operations
type JournalRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, params CreateJournalEntryParams) (*JournalEntry, error)
	GetByID(ctx context.Context, tenantID uuid.UUID, journalEntryID uuid.UUID) (*JournalEntry, error)
	List(ctx context.Context, tenantID uuid.UUID, accountID *uuid.UUID, fromDate, toDate *time.Time, limit, offset int) ([]*JournalEntry, int, error)
}

// ReferenceRepositoryInterface defines methods for reference data operations
type ReferenceRepositoryInterface interface {
	ListAccountTypes(ctx context.Context) ([]*AccountType, error)
	ListCurrencies(ctx context.Context) ([]*Currency, error)
}
