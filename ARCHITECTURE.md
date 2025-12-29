# Ledger Microservice Architecture

## Overview

This document describes the architecture of the Ledger microservice, a high-performance, multi-tenant double-entry accounting system built with Go and gRPC.

## Design Principles

1. **Separation of Concerns**: Clear separation between layers (API, Service, Repository, Database)
2. **Multi-Tenancy**: Complete data isolation using PostgreSQL Row-Level Security (RLS)
3. **Database-First**: Business logic encapsulated in PostgreSQL functions
4. **Performance**: Connection pooling, denormalized balances, optimized queries
5. **Testability**: Interface-based design for easy mocking and testing

## Architecture Layers

```
┌─────────────────────────────────────────────────────┐
│                   gRPC Clients                      │
└────────────────────┬────────────────────────────────┘
                     │ Protocol Buffers
┌────────────────────▼────────────────────────────────┐
│               Service Layer (gRPC)                  │
│  - Input validation                                 │
│  - Request/Response mapping                         │
│  - Error handling                                   │
└────────────────────┬────────────────────────────────┘
                     │ Repository Interfaces
┌────────────────────▼────────────────────────────────┐
│              Repository Layer                       │
│  - Data access abstraction                          │
│  - Tenant context management                        │
│  - Query construction                               │
└────────────────────┬────────────────────────────────┘
                     │ pgx/pgxpool
┌────────────────────▼────────────────────────────────┐
│            Database Layer (PostgreSQL 18)           │
│  - Row-Level Security (RLS)                         │
│  - Database functions                               │
│  - Triggers & constraints                           │
│  - ACID transactions                                │
└─────────────────────────────────────────────────────┘
```

## Multi-Tenancy Strategy

### Row-Level Security (RLS)

The system uses PostgreSQL's Row-Level Security for complete tenant isolation:

1. **Session Context**: Each database connection sets `app.current_tenant_id`
   ```sql
   SET LOCAL app.current_tenant_id = '<tenant-uuid>';
   ```

2. **RLS Policies**: Automatic filtering based on tenant_id
   ```sql
   CREATE POLICY tenant_isolation_policy ON accounts
       USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
   ```

3. **Advantages**:
   - Security enforced at database level
   - No application-level filtering needed
   - Impossible to accidentally access other tenant's data
   - Simplified application code

### Implementation Details

```go
// Database connection with tenant context
func (d *DB) WithTenant(ctx context.Context, tenantID string) (context.Context, *pgxpool.Conn, error) {
    conn, err := d.pool.Acquire(ctx)
    if err != nil {
        return nil, nil, err
    }

    // Set tenant context - enforced by RLS
    _, err = conn.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
    if err != nil {
        conn.Release()
        return nil, nil, err
    }

    return ctx, conn, nil
}
```

## Database Schema

### Core Tables

#### tenants
- Multi-tenant organization data
- No RLS (global access needed for tenant creation)

#### accounts
- Chart of accounts for each tenant
- RLS enabled with tenant_id isolation
- Single currency per account
- Hierarchical structure support (parent_account_id)

#### journal_entries
- Double-entry journal transactions
- RLS enabled with tenant_id isolation
- JSONB metadata for flexible tax/custom data

#### journal_entry_lines
- Individual debit/credit entries
- RLS inherited through journal_entries relationship
- Constraint: Either debit OR credit (never both)

#### account_balances
- Denormalized balance cache for performance
- RLS inherited through accounts relationship
- Updated automatically by triggers/functions

### Database Functions

#### create_tenant(name)
Creates a new tenant with proper initialization.

#### create_account(...)
Creates an account with:
- Automatic balance initialization
- Tenant context enforcement
- Parent-child relationship validation

#### create_journal_entry(...)
Creates a balanced journal entry with:
- Balance validation (debits = credits)
- Minimum 2 lines requirement
- Automatic balance updates
- Atomic transaction

## Service Layer

### LedgerService (gRPC)

Implements the Protocol Buffer service definition:

```protobuf
service LedgerService {
  // Tenant Management
  rpc CreateTenant(CreateTenantRequest) returns (CreateTenantResponse);
  rpc GetTenant(GetTenantRequest) returns (GetTenantResponse);

  // Account Management
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc GetAccountBalance(GetAccountBalanceRequest) returns (GetAccountBalanceResponse);

  // Journal Entry Management
  rpc CreateJournalEntry(CreateJournalEntryRequest) returns (CreateJournalEntryResponse);
  rpc GetJournalEntry(GetJournalEntryRequest) returns (GetJournalEntryResponse);
  rpc ListJournalEntries(ListJournalEntriesRequest) returns (ListJournalEntriesResponse);

  // Reference Data
  rpc ListAccountTypes(ListAccountTypesRequest) returns (ListAccountTypesResponse);
  rpc ListCurrencies(ListCurrenciesRequest) returns (ListCurrenciesResponse);
}
```

### Responsibilities

- **Input Validation**: UUID parsing, required fields, format checking
- **Data Mapping**: Convert between Protocol Buffer messages and domain models
- **Error Handling**: Convert errors to appropriate gRPC status codes
- **Tenant Context**: Pass tenant_id to repository layer

## Repository Layer

### Design Pattern

Uses interface-based design for testability:

```go
type TenantRepositoryInterface interface {
    Create(ctx context.Context, name string) (*Tenant, error)
    GetByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error)
    GetByName(ctx context.Context, name string) (*Tenant, error)
}
```

### Repositories

1. **TenantRepository**: Tenant CRUD operations
2. **AccountRepository**: Account management with tenant context
3. **JournalRepository**: Journal entry operations with balance updates
4. **ReferenceRepository**: Account types and currencies (global data)

### Transaction Management

```go
// Start transaction with tenant context
tx, err := db.BeginTx(ctx, tenantID.String())

// Execute operations
// ...

// Commit or rollback
tx.Commit(ctx)
```

## Double-Entry Accounting Logic

### Principles

1. **Balanced Entries**: Total debits must equal total credits
2. **Minimum Two Lines**: Every entry must have at least 2 lines
3. **Atomic Updates**: Entry creation and balance updates in single transaction
4. **Account Types**: Normal balance determines debit/credit side

### Account Types

| Type | Normal Balance | Increases With | Decreases With |
|------|----------------|----------------|----------------|
| Asset | Debit | Debit | Credit |
| Liability | Credit | Credit | Debit |
| Equity | Credit | Credit | Debit |
| Revenue | Credit | Credit | Debit |
| Expense | Debit | Debit | Credit |

### Example Transaction

Sale of goods for cash:

```json
{
  "reference_number": "INV-001",
  "description": "Sale of goods",
  "lines": [
    {
      "account_id": "cash-account-uuid",
      "debit": "1000.00",
      "credit": "0",
      "description": "Cash received"
    },
    {
      "account_id": "revenue-account-uuid",
      "debit": "0",
      "credit": "1000.00",
      "description": "Revenue from sale"
    }
  ]
}
```

## Performance Optimizations

### Connection Pooling

```go
poolConfig.MaxConns = 25
poolConfig.MinConns = 5
poolConfig.MaxConnLifetime = time.Hour
poolConfig.MaxConnIdleTime = 30 * time.Minute
```

### Denormalized Balances

- `account_balances` table caches current balances
- Updated atomically with journal entries
- Avoids expensive SUM queries on journal_entry_lines

### Database Indexes

```sql
-- Implicit indexes on primary keys and unique constraints
-- Foreign key indexes for joins
CREATE INDEX idx_accounts_tenant_id ON accounts(tenant_id);
CREATE INDEX idx_journal_entries_tenant_id ON journal_entries(tenant_id);
CREATE INDEX idx_journal_entry_lines_journal_entry_id ON journal_entry_lines(journal_entry_id);
CREATE INDEX idx_journal_entry_lines_account_id ON journal_entry_lines(account_id);
```

### Query Optimization

- Use of `EXISTS` in RLS policies instead of JOINs
- Pagination support for list operations
- Selective column retrieval

## Security Considerations

### Data Isolation

- RLS enforced at database level
- Tenant context required for all operations
- No cross-tenant queries possible

### Input Validation

- UUID format validation
- Numeric precision validation
- Required field checks
- Balance validation

### SQL Injection Prevention

- Parameterized queries only
- No string concatenation in SQL
- Database functions with typed parameters

### Prepared Statements

```go
// Always use parameterized queries
conn.QueryRow(ctx, "SELECT * FROM accounts WHERE id = $1", accountID)
```

## Testing Strategy

### Unit Tests

- Service layer with mocked repositories
- Configuration loading
- Business logic validation
- Located in `*_test.go` files alongside code

### Integration Tests

- Real database operations
- Transaction testing
- RLS policy verification
- Tag-based execution: `-tags=integration`

### Test Coverage

- Config package: 100%
- Service package: 31.7% (focused on critical paths)
- Integration tests for database operations

## Deployment

### Docker

Multi-stage build for minimal image size:

```dockerfile
# Build stage
FROM golang:1.25.5-alpine AS builder
# ... build steps

# Final stage
FROM alpine:latest
COPY --from=builder /app/ledger .
```

### Docker Compose

Complete stack with database:

```yaml
services:
  ledger-service:
    build: .
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:18-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
```

### Configuration

Environment-based configuration:
- `SERVER_HOST`, `SERVER_PORT`: gRPC server
- `DB_*`: Database connection parameters
- `DB_MAX_CONNS`, `DB_MIN_CONNS`: Connection pool

## Monitoring & Observability

### Logging

- Structured logging with context
- Database connection status
- Transaction boundaries
- Error conditions

### Metrics (Future)

- Request latency
- Transaction throughput
- Connection pool usage
- Error rates

### Tracing (Future)

- Distributed tracing with OpenTelemetry
- Request flow visualization
- Database query tracing

## Scalability

### Horizontal Scaling

- Stateless service design
- Database connection pooling
- No in-memory state

### Database Scaling

- Read replicas for queries
- Connection pooling prevents connection exhaustion
- Denormalized balances reduce query complexity

### Caching (Future)

- Account type/currency caching (rarely changes)
- Balance caching with invalidation strategy
- Redis for distributed cache

## Future Enhancements

1. **Audit Trail**: Immutable history of all changes
2. **Soft Deletes**: Mark records as deleted instead of removing
3. **Multi-Currency Transactions**: Exchange rate support
4. **Recurring Entries**: Scheduled journal entries
5. **Reporting**: Balance sheets, income statements, trial balance
6. **Bulk Operations**: Batch journal entry creation
7. **Webhooks**: Event notifications for integrations
8. **GraphQL API**: Alternative to gRPC for web clients

## References

- [PostgreSQL Row-Level Security](https://www.postgresql.org/docs/18/ddl-rowsecurity.html)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Double-Entry Accounting](https://en.wikipedia.org/wiki/Double-entry_bookkeeping)
- [Protocol Buffers Style Guide](https://protobuf.dev/programming-guides/style/)
