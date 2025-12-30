# Ledger Microservice

A high-performance, multi-tenant double-entry accounting ledger microservice built with Go and gRPC.

## Features

- **Double-Entry Accounting**: Full implementation of double-entry bookkeeping principles
- **Multi-Tenancy**: Row-Level Security (RLS) at the database level for complete tenant isolation
- **gRPC API**: High-performance Protocol Buffer-based API
- **PostgreSQL 18**: Leverages advanced PostgreSQL features including:
  - Row-Level Security (RLS) for multi-tenancy
  - Database functions for business logic
  - JSONB for flexible metadata storage
  - Atomic transactions
- **Microservice Architecture**: Designed to be deployed as an independent service
- **Comprehensive Testing**: Unit tests and integration tests included

## Architecture

### Database Schema

The system uses PostgreSQL 18 with the following core tables:

- `tenants`: Multi-tenant organization data
- `account_types`: Chart of account types (Asset, Liability, Equity, Revenue, Expense)
- `currencies`: Multi-currency support
- `accounts`: Chart of accounts for each tenant
- `journal_entries`: Double-entry journal transactions
- `journal_entry_lines`: Individual debit/credit entries
- `account_balances`: Denormalized balances for performance

### Multi-Tenancy Strategy

Row-Level Security (RLS) is implemented at the database level using PostgreSQL's native RLS feature. Each connection sets `app.current_tenant_id` which is enforced by RLS policies, ensuring complete data isolation between tenants.

### Service Layer

The gRPC service provides the following operations:

- **Tenant Management**: Create and retrieve tenants
- **Account Management**: Create accounts, list accounts, retrieve balances
- **Journal Entries**: Create double-entry transactions, list entries with filters
- **Reference Data**: List account types and currencies

## Prerequisites

- Go 1.25.5 or higher
- PostgreSQL 18
- buf CLI (for Protocol Buffer generation)
- Docker (optional, for running database and tests)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/hesabFun/ledger.git
cd ledger
```

2. Install dependencies:
```bash
go mod download
```

3. Install development tools:
```bash
make install-tools
```

4. Generate Protocol Buffer code:
```bash
make proto
```

## Configuration

Copy `.env.example` to `.env` and configure your environment:

```bash
cp .env.example .env
```

Configuration options:

- `SERVER_HOST`: gRPC server host (default: 0.0.0.0)
- `SERVER_PORT`: gRPC server port (default: 9090)
- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (default: postgres)
- `DB_NAME`: Database name (default: ledger)
- `DB_SSL_MODE`: SSL mode (default: disable)
- `DB_MAX_CONNS`: Maximum database connections (default: 25)
- `DB_MIN_CONNS`: Minimum database connections (default: 5)

## Running the Service

### Using Make

```bash
make run
```

### Using Go directly

```bash
go run cmd/server/main.go
```

### Building

```bash
make build
./bin/ledger
```

## Testing

### Run all tests

```bash
make test
```

### Run tests with coverage

```bash
make coverage
```

### Run integration tests

Integration tests require a running PostgreSQL instance with the schema applied:

```bash
go test -v -tags=integration ./internal/repository/
```

## API Documentation

The service exposes a gRPC API defined in `proto/ledger/v1/ledger.proto`.

### Example: Creating a Tenant

```bash
grpcurl -plaintext -d '{"name": "Acme Corp"}' \
  localhost:9090 ledger.v1.LedgerService/CreateTenant
```

### Example: Creating an Account

```bash
grpcurl -plaintext -d '{
  "tenant_id": "uuid-here",
  "account_number": "1000",
  "name": "Cash",
  "account_type_id": 1,
  "currency_code": "USD"
}' localhost:9090 ledger.v1.LedgerService/CreateAccount
```

### Example: Creating a Journal Entry

```bash
grpcurl -plaintext -d '{
  "tenant_id": "uuid-here",
  "reference_number": "INV-001",
  "description": "Sale transaction",
  "entry_date": "2024-01-15T10:00:00Z",
  "lines": [
    {
      "account_id": "debit-account-uuid",
      "debit": "100.00",
      "credit": "0",
      "description": "Cash received"
    },
    {
      "account_id": "credit-account-uuid",
      "debit": "0",
      "credit": "100.00",
      "description": "Revenue"
    }
  ]
}' localhost:9090 ledger.v1.LedgerService/CreateJournalEntry
```

## Project Structure

```
.
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── db/              # Database connection and utilities
│   ├── repository/      # Data access layer
│   └── service/         # gRPC service implementation
├── proto/
│   └── ledger/v1/       # Protocol Buffer definitions
├── gen/                 # Generated code (gitignored)
├── db-schema/           # Database schema and migrations (submodule)
├── Makefile            # Build automation
├── buf.yaml            # Buf configuration
└── buf.gen.yaml        # Buf code generation config
```

## Development

### Code Generation

After modifying `.proto` files:

```bash
make proto
```

### Linting

```bash
make lint
```

### Formatting

```bash
make fmt
```

## Database Functions

The service leverages PostgreSQL functions for critical operations:

- `create_tenant(name)`: Creates a new tenant
- `create_account(...)`: Creates a new account with automatic balance initialization
- `create_journal_entry(...)`: Creates a balanced journal entry with automatic validation and balance updates

These functions ensure data integrity and encapsulate business logic at the database level.

## Performance Considerations

- Connection pooling with configurable min/max connections
- Denormalized `account_balances` table for fast balance queries
- Database indexes on foreign keys and frequently queried columns
- RLS policies optimized with proper indexing

## Security

- Row-Level Security (RLS) ensures tenant data isolation
- All tenant operations require tenant context
- Database functions validate business rules
- Prepared statements prevent SQL injection

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write tests
5. Run tests and linting
6. Submit a pull request

## License

MIT License - See LICENSE file for details

## Support

For issues and questions, please open an issue on GitHub.