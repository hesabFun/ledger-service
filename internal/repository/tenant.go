package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/db"
)

// Tenant represents a tenant entity
type Tenant struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TenantRepository handles tenant database operations
type TenantRepository struct {
	db *db.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(database *db.DB) *TenantRepository {
	return &TenantRepository{db: database}
}

// Create creates a new tenant using the database function
func (r *TenantRepository) Create(ctx context.Context, name string) (*Tenant, error) {
	var tenantID uuid.UUID

	query := "SELECT create_tenant($1)"
	err := r.db.Pool().QueryRow(ctx, query, name).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Fetch the created tenant details
	return r.GetByID(ctx, tenantID)
}

// GetByID retrieves a tenant by ID
func (r *TenantRepository) GetByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error) {
	tenant := &Tenant{}

	query := `
		SELECT id, name, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	err := r.db.Pool().QueryRow(ctx, query, tenantID).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return tenant, nil
}

// GetByName retrieves a tenant by name
func (r *TenantRepository) GetByName(ctx context.Context, name string) (*Tenant, error) {
	tenant := &Tenant{}

	query := `
		SELECT id, name, created_at, updated_at
		FROM tenants
		WHERE name = $1
	`

	err := r.db.Pool().QueryRow(ctx, query, name).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by name: %w", err)
	}

	return tenant, nil
}
