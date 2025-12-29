//go:build integration
// +build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/config"
	"github.com/hesabFun/ledger/internal/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite is the test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	db            *db.DB
	tenantRepo    *TenantRepository
	accountRepo   *AccountRepository
	journalRepo   *JournalRepository
	referenceRepo *ReferenceRepository
	testTenantID  uuid.UUID
}

// SetupSuite runs once before all tests
func (s *IntegrationTestSuite) SetupSuite() {
	// Load configuration from environment
	cfg := &config.DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("DB_NAME", "ledger"),
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

	// Connect to database
	ctx := context.Background()
	database, err := db.New(ctx, cfg)
	require.NoError(s.T(), err, "Failed to connect to database")

	s.db = database

	// Initialize repositories
	s.tenantRepo = NewTenantRepository(database)
	s.accountRepo = NewAccountRepository(database)
	s.journalRepo = NewJournalRepository(database)
	s.referenceRepo = NewReferenceRepository(database)
}

// TearDownSuite runs once after all tests
func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// SetupTest runs before each test
func (s *IntegrationTestSuite) SetupTest() {
	// Create a test tenant for each test
	tenant, err := s.tenantRepo.Create(context.Background(), "test-tenant-"+uuid.New().String())
	require.NoError(s.T(), err)
	s.testTenantID = tenant.ID
}

// TearDownTest runs after each test
func (s *IntegrationTestSuite) TearDownTest() {
	// Clean up: delete the test tenant (cascade will delete related data)
	if s.testTenantID != uuid.Nil {
		_, err := s.db.Pool().Exec(context.Background(), "DELETE FROM tenants WHERE id = $1", s.testTenantID)
		require.NoError(s.T(), err)
	}
}

// TestTenantRepository_Create tests creating a tenant
func (s *IntegrationTestSuite) TestTenantRepository_Create() {
	ctx := context.Background()

	tenant, err := s.tenantRepo.Create(ctx, "integration-test-tenant")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), tenant)

	assert.NotEqual(s.T(), uuid.Nil, tenant.ID)
	assert.Equal(s.T(), "integration-test-tenant", tenant.Name)
	assert.False(s.T(), tenant.CreatedAt.IsZero())
	assert.False(s.T(), tenant.UpdatedAt.IsZero())

	// Clean up
	_, err = s.db.Pool().Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenant.ID)
	require.NoError(s.T(), err)
}

// TestTenantRepository_GetByID tests retrieving a tenant by ID
func (s *IntegrationTestSuite) TestTenantRepository_GetByID() {
	ctx := context.Background()

	tenant, err := s.tenantRepo.GetByID(ctx, s.testTenantID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), tenant)

	assert.Equal(s.T(), s.testTenantID, tenant.ID)
	assert.NotEmpty(s.T(), tenant.Name)
}

// TestAccountRepository_Create tests creating an account
func (s *IntegrationTestSuite) TestAccountRepository_Create() {
	ctx := context.Background()

	params := CreateAccountParams{
		AccountNumber: "1000",
		Name:          "Cash",
		AccountTypeID: 1, // Assuming ASSET type exists
		CurrencyCode:  "USD",
	}

	account, err := s.accountRepo.Create(ctx, s.testTenantID, params)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), account)

	assert.NotEqual(s.T(), uuid.Nil, account.ID)
	assert.Equal(s.T(), s.testTenantID, account.TenantID)
	assert.Equal(s.T(), "1000", account.AccountNumber)
	assert.Equal(s.T(), "Cash", account.Name)
	assert.True(s.T(), account.IsActive)
}

// TestAccountRepository_GetByID tests retrieving an account by ID
func (s *IntegrationTestSuite) TestAccountRepository_GetByID() {
	ctx := context.Background()

	// Create an account first
	params := CreateAccountParams{
		AccountNumber: "2000",
		Name:          "Bank Account",
		AccountTypeID: 1,
		CurrencyCode:  "USD",
	}

	created, err := s.accountRepo.Create(ctx, s.testTenantID, params)
	require.NoError(s.T(), err)

	// Retrieve the account
	account, err := s.accountRepo.GetByID(ctx, s.testTenantID, created.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), account)

	assert.Equal(s.T(), created.ID, account.ID)
	assert.Equal(s.T(), "2000", account.AccountNumber)
}

// TestAccountRepository_List tests listing accounts
func (s *IntegrationTestSuite) TestAccountRepository_List() {
	ctx := context.Background()

	// Create multiple accounts
	for i := 1; i <= 3; i++ {
		params := CreateAccountParams{
			AccountNumber: uuid.New().String()[:8],
			Name:          "Test Account",
			AccountTypeID: 1,
			CurrencyCode:  "USD",
		}
		_, err := s.accountRepo.Create(ctx, s.testTenantID, params)
		require.NoError(s.T(), err)
	}

	// List accounts
	accounts, totalCount, err := s.accountRepo.List(ctx, s.testTenantID, nil, nil, 10, 0)
	require.NoError(s.T(), err)

	assert.GreaterOrEqual(s.T(), len(accounts), 3)
	assert.GreaterOrEqual(s.T(), totalCount, 3)
}

// TestAccountRepository_GetBalance tests retrieving account balance
func (s *IntegrationTestSuite) TestAccountRepository_GetBalance() {
	ctx := context.Background()

	// Create an account
	params := CreateAccountParams{
		AccountNumber: "3000",
		Name:          "Revenue Account",
		AccountTypeID: 1,
		CurrencyCode:  "USD",
	}

	account, err := s.accountRepo.Create(ctx, s.testTenantID, params)
	require.NoError(s.T(), err)

	// Get balance (should be zero initially)
	balance, err := s.accountRepo.GetBalance(ctx, s.testTenantID, account.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), balance)

	assert.Equal(s.T(), decimal.Zero, balance.DebitBalance)
	assert.Equal(s.T(), decimal.Zero, balance.CreditBalance)
}

// TestJournalRepository_Create tests creating a journal entry
func (s *IntegrationTestSuite) TestJournalRepository_Create() {
	ctx := context.Background()

	// Create two accounts
	account1, err := s.accountRepo.Create(ctx, s.testTenantID, CreateAccountParams{
		AccountNumber: "4000",
		Name:          "Debit Account",
		AccountTypeID: 1,
		CurrencyCode:  "USD",
	})
	require.NoError(s.T(), err)

	account2, err := s.accountRepo.Create(ctx, s.testTenantID, CreateAccountParams{
		AccountNumber: "5000",
		Name:          "Credit Account",
		AccountTypeID: 2,
		CurrencyCode:  "USD",
	})
	require.NoError(s.T(), err)

	// Create journal entry
	params := CreateJournalEntryParams{
		ReferenceNumber: "TEST-001",
		Description:     "Test transaction",
		EntryDate:       time.Now(),
		Lines: []*CreateJournalEntryLineParams{
			{
				AccountID:   account1.ID,
				Debit:       decimal.NewFromInt(100),
				Credit:      decimal.Zero,
				Description: "Debit line",
			},
			{
				AccountID:   account2.ID,
				Debit:       decimal.Zero,
				Credit:      decimal.NewFromInt(100),
				Description: "Credit line",
			},
		},
	}

	entry, err := s.journalRepo.Create(ctx, s.testTenantID, params)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), entry)

	assert.NotEqual(s.T(), uuid.Nil, entry.ID)
	assert.Equal(s.T(), "TEST-001", entry.ReferenceNumber)
	assert.Len(s.T(), entry.Lines, 2)

	// Verify balances were updated
	balance1, err := s.accountRepo.GetBalance(ctx, s.testTenantID, account1.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "100", balance1.DebitBalance.String())

	balance2, err := s.accountRepo.GetBalance(ctx, s.testTenantID, account2.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "100", balance2.CreditBalance.String())
}

// TestJournalRepository_GetByID tests retrieving a journal entry by ID
func (s *IntegrationTestSuite) TestJournalRepository_GetByID() {
	ctx := context.Background()

	// Create accounts and journal entry
	account1, _ := s.accountRepo.Create(ctx, s.testTenantID, CreateAccountParams{
		AccountNumber: "6000",
		Name:          "Account 1",
		AccountTypeID: 1,
		CurrencyCode:  "USD",
	})

	account2, _ := s.accountRepo.Create(ctx, s.testTenantID, CreateAccountParams{
		AccountNumber: "7000",
		Name:          "Account 2",
		AccountTypeID: 2,
		CurrencyCode:  "USD",
	})

	created, err := s.journalRepo.Create(ctx, s.testTenantID, CreateJournalEntryParams{
		ReferenceNumber: "TEST-002",
		Description:     "Test entry",
		EntryDate:       time.Now(),
		Lines: []*CreateJournalEntryLineParams{
			{AccountID: account1.ID, Debit: decimal.NewFromInt(50), Credit: decimal.Zero, Description: "Line 1"},
			{AccountID: account2.ID, Debit: decimal.Zero, Credit: decimal.NewFromInt(50), Description: "Line 2"},
		},
	})
	require.NoError(s.T(), err)

	// Retrieve the journal entry
	entry, err := s.journalRepo.GetByID(ctx, s.testTenantID, created.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), entry)

	assert.Equal(s.T(), created.ID, entry.ID)
	assert.Len(s.T(), entry.Lines, 2)
}

// TestReferenceRepository_ListAccountTypes tests listing account types
func (s *IntegrationTestSuite) TestReferenceRepository_ListAccountTypes() {
	ctx := context.Background()

	accountTypes, err := s.referenceRepo.ListAccountTypes(ctx)
	require.NoError(s.T(), err)

	// Assuming there are some account types in the database
	assert.NotEmpty(s.T(), accountTypes)
}

// TestReferenceRepository_ListCurrencies tests listing currencies
func (s *IntegrationTestSuite) TestReferenceRepository_ListCurrencies() {
	ctx := context.Background()

	currencies, err := s.referenceRepo.ListCurrencies(ctx)
	require.NoError(s.T(), err)

	// Assuming there are some currencies in the database
	assert.NotEmpty(s.T(), currencies)
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(IntegrationTestSuite))
}

// Helper function to get environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
