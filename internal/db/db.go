package db

import (
	"context"
	"fmt"
	"time"

	"github.com/hesabFun/ledger/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgxpool connection pool
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(ctx context.Context, cfg *config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Pool returns the underlying connection pool
func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// Close closes the database connection pool
func (d *DB) Close() {
	d.pool.Close()
}

// WithTenant returns a connection with the tenant_id set for RLS
func (d *DB) WithTenant(ctx context.Context, tenantID string) (context.Context, *pgxpool.Conn, error) {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to acquire connection: %w", err)
	}

	// Set the tenant_id for Row-Level Security
	_, err = conn.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		conn.Release()
		return nil, nil, fmt.Errorf("unable to set tenant_id: %w", err)
	}

	return ctx, conn, nil
}

// BeginTx starts a transaction with tenant context
func (d *DB) BeginTx(ctx context.Context, tenantID string) (*TenantTx, error) {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to acquire connection: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("unable to begin transaction: %w", err)
	}

	// Set the tenant_id for Row-Level Security within the transaction
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		_ = tx.Rollback(ctx)
		conn.Release()
		return nil, fmt.Errorf("unable to set tenant_id: %w", err)
	}

	return &TenantTx{
		tx:       tx,
		conn:     conn,
		tenantID: tenantID,
	}, nil
}

// TenantTx wraps a transaction with tenant context
type TenantTx struct {
	tx       pgx.Tx
	conn     *pgxpool.Conn
	tenantID string
}

// Exec executes a query within the tenant transaction
func (t *TenantTx) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}

// Query executes a query and returns rows within the tenant transaction
func (t *TenantTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row within the tenant transaction
func (t *TenantTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// Commit commits the transaction and releases the connection
func (t *TenantTx) Commit(ctx context.Context) error {
	err := t.tx.Commit(ctx)
	t.conn.Release()
	return err
}

// Rollback rolls back the transaction and releases the connection
func (t *TenantTx) Rollback(ctx context.Context) error {
	err := t.tx.Rollback(ctx)
	t.conn.Release()
	return err
}
