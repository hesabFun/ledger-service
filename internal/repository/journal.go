package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// JournalEntry represents a journal entry entity
type JournalEntry struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	ReferenceNumber string
	Description     string
	EntryDate       time.Time
	Metadata        map[string]interface{}
	Lines           []*JournalEntryLine
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// JournalEntryLine represents a single line in a journal entry
type JournalEntryLine struct {
	ID             uuid.UUID
	JournalEntryID uuid.UUID
	AccountID      uuid.UUID
	Debit          decimal.Decimal
	Credit         decimal.Decimal
	Description    string
	CreatedAt      time.Time
}

// CreateJournalEntryParams holds parameters for creating a journal entry
type CreateJournalEntryParams struct {
	ReferenceNumber string
	Description     string
	EntryDate       time.Time
	Metadata        map[string]interface{}
	Lines           []*CreateJournalEntryLineParams
}

// CreateJournalEntryLineParams holds parameters for creating a journal entry line
type CreateJournalEntryLineParams struct {
	AccountID   uuid.UUID
	Debit       decimal.Decimal
	Credit      decimal.Decimal
	Description string
}

// JournalRepository handles journal entry database operations
type JournalRepository struct {
	db *db.DB
}

// NewJournalRepository creates a new journal repository
func NewJournalRepository(database *db.DB) *JournalRepository {
	return &JournalRepository{db: database}
}

// Create creates a new journal entry using the database function
func (r *JournalRepository) Create(ctx context.Context, tenantID uuid.UUID, params CreateJournalEntryParams) (*JournalEntry, error) {
	// Start a transaction with tenant context
	tx, err := r.db.BeginTx(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Convert lines to JSONB format expected by the database function
	linesJSON := make([]map[string]interface{}, len(params.Lines))
	for i, line := range params.Lines {
		linesJSON[i] = map[string]interface{}{
			"account_id":  line.AccountID.String(),
			"debit":       line.Debit.String(),
			"credit":      line.Credit.String(),
			"description": line.Description,
		}
	}

	linesBytes, err := json.Marshal(linesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal lines: %w", err)
	}

	var metadataBytes []byte
	if params.Metadata != nil {
		metadataBytes, err = json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	var journalEntryID uuid.UUID
	query := "SELECT create_journal_entry($1, $2, $3, $4, $5)"

	err = tx.QueryRow(ctx, query,
		params.ReferenceNumber,
		params.Description,
		params.EntryDate,
		string(linesBytes),
		string(metadataBytes),
	).Scan(&journalEntryID)

	if err != nil {
		return nil, fmt.Errorf("failed to create journal entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch the created journal entry details
	return r.GetByID(ctx, tenantID, journalEntryID)
}

// GetByID retrieves a journal entry by ID with tenant context
func (r *JournalRepository) GetByID(ctx context.Context, tenantID uuid.UUID, journalEntryID uuid.UUID) (*JournalEntry, error) {
	_, conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	defer conn.Release()

	entry := &JournalEntry{}
	var metadataBytes []byte

	query := `
		SELECT id, tenant_id, reference_number, description, entry_date,
		       metadata, created_at, updated_at
		FROM journal_entries
		WHERE id = $1
	`

	err = conn.QueryRow(ctx, query, journalEntryID).Scan(
		&entry.ID,
		&entry.TenantID,
		&entry.ReferenceNumber,
		&entry.Description,
		&entry.EntryDate,
		&metadataBytes,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("journal entry not found")
		}
		return nil, fmt.Errorf("failed to get journal entry: %w", err)
	}

	// Parse metadata if present
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &entry.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Fetch journal entry lines
	lines, err := r.getLinesByJournalEntryID(ctx, conn, journalEntryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get journal entry lines: %w", err)
	}
	entry.Lines = lines

	return entry, nil
}

// getLinesByJournalEntryID retrieves all lines for a journal entry
func (r *JournalRepository) getLinesByJournalEntryID(ctx context.Context, conn *pgxpool.Conn, journalEntryID uuid.UUID) ([]*JournalEntryLine, error) {
	query := `
		SELECT id, journal_entry_id, account_id, debit, credit, description, created_at
		FROM journal_entry_lines
		WHERE journal_entry_id = $1
		ORDER BY created_at
	`

	rows, err := conn.Query(ctx, query, journalEntryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query journal entry lines: %w", err)
	}
	defer rows.Close()

	lines := make([]*JournalEntryLine, 0)
	for rows.Next() {
		line := &JournalEntryLine{}
		err := rows.Scan(
			&line.ID,
			&line.JournalEntryID,
			&line.AccountID,
			&line.Debit,
			&line.Credit,
			&line.Description,
			&line.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan journal entry line: %w", err)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

// List retrieves journal entries with optional filters
func (r *JournalRepository) List(ctx context.Context, tenantID uuid.UUID, accountID *uuid.UUID, fromDate, toDate *time.Time, limit, offset int) ([]*JournalEntry, int, error) {
	_, conn, err := r.db.WithTenant(ctx, tenantID.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to set tenant context: %w", err)
	}
	defer conn.Release()

	// Build query with filters
	query := `
		SELECT DISTINCT je.id, je.tenant_id, je.reference_number, je.description,
		       je.entry_date, je.metadata, je.created_at, je.updated_at
		FROM journal_entries je
	`
	countQuery := "SELECT COUNT(DISTINCT je.id) FROM journal_entries je"
	args := []interface{}{}
	argCount := 0

	// Add join if filtering by account
	if accountID != nil {
		query += " INNER JOIN journal_entry_lines jel ON je.id = jel.journal_entry_id"
		countQuery += " INNER JOIN journal_entry_lines jel ON je.id = jel.journal_entry_id"
		argCount++
		query += fmt.Sprintf(" WHERE jel.account_id = $%d", argCount)
		countQuery += fmt.Sprintf(" WHERE jel.account_id = $%d", argCount)
		args = append(args, *accountID)
	} else {
		query += " WHERE 1=1"
		countQuery += " WHERE 1=1"
	}

	if fromDate != nil {
		argCount++
		query += fmt.Sprintf(" AND je.entry_date >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND je.entry_date >= $%d", argCount)
		args = append(args, *fromDate)
	}

	if toDate != nil {
		argCount++
		query += fmt.Sprintf(" AND je.entry_date <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND je.entry_date <= $%d", argCount)
		args = append(args, *toDate)
	}

	// Get total count
	var totalCount int
	err = conn.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count journal entries: %w", err)
	}

	// Add pagination
	argCount++
	query += fmt.Sprintf(" ORDER BY je.entry_date DESC, je.created_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list journal entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*JournalEntry, 0)
	for rows.Next() {
		entry := &JournalEntry{}
		var metadataBytes []byte

		err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.ReferenceNumber,
			&entry.Description,
			&entry.EntryDate,
			&metadataBytes,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan journal entry: %w", err)
		}

		// Parse metadata if present
		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &entry.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Fetch lines for this entry
		lines, err := r.getLinesByJournalEntryID(ctx, conn, entry.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get journal entry lines: %w", err)
		}
		entry.Lines = lines

		entries = append(entries, entry)
	}

	return entries, totalCount, nil
}
