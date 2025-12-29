package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/hesabFun/ledger/gen/go/ledger/v1"
)

// Mock repositories
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, name string) (*repository.Tenant, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, tenantID uuid.UUID) (*repository.Tenant, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByName(ctx context.Context, name string) (*repository.Tenant, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Tenant), args.Error(1)
}

type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) Create(ctx context.Context, tenantID uuid.UUID, params repository.CreateAccountParams) (*repository.Account, error) {
	args := m.Called(ctx, tenantID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Account), args.Error(1)
}

func (m *MockAccountRepository) GetByID(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*repository.Account, error) {
	args := m.Called(ctx, tenantID, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Account), args.Error(1)
}

func (m *MockAccountRepository) List(ctx context.Context, tenantID uuid.UUID, accountTypeID *int32, currencyCode *string, limit, offset int) ([]*repository.Account, int, error) {
	args := m.Called(ctx, tenantID, accountTypeID, currencyCode, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*repository.Account), args.Int(1), args.Error(2)
}

func (m *MockAccountRepository) GetBalance(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) (*repository.AccountBalance, error) {
	args := m.Called(ctx, tenantID, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.AccountBalance), args.Error(1)
}

type MockJournalRepository struct {
	mock.Mock
}

func (m *MockJournalRepository) Create(ctx context.Context, tenantID uuid.UUID, params repository.CreateJournalEntryParams) (*repository.JournalEntry, error) {
	args := m.Called(ctx, tenantID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.JournalEntry), args.Error(1)
}

func (m *MockJournalRepository) GetByID(ctx context.Context, tenantID uuid.UUID, journalEntryID uuid.UUID) (*repository.JournalEntry, error) {
	args := m.Called(ctx, tenantID, journalEntryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.JournalEntry), args.Error(1)
}

func (m *MockJournalRepository) List(ctx context.Context, tenantID uuid.UUID, accountID *uuid.UUID, fromDate, toDate *time.Time, limit, offset int) ([]*repository.JournalEntry, int, error) {
	args := m.Called(ctx, tenantID, accountID, fromDate, toDate, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*repository.JournalEntry), args.Int(1), args.Error(2)
}

type MockReferenceRepository struct {
	mock.Mock
}

func (m *MockReferenceRepository) ListAccountTypes(ctx context.Context) ([]*repository.AccountType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.AccountType), args.Error(1)
}

func (m *MockReferenceRepository) ListCurrencies(ctx context.Context) ([]*repository.Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Currency), args.Error(1)
}

// Test CreateTenant
func TestLedgerService_CreateTenant(t *testing.T) {
	ctx := context.Background()
	mockTenantRepo := new(MockTenantRepository)
	service := NewLedgerService(mockTenantRepo, nil, nil, nil)

	t.Run("successfully creates tenant", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now()

		mockTenantRepo.On("Create", ctx, "Test Tenant").Return(&repository.Tenant{
			ID:        tenantID,
			Name:      "Test Tenant",
			CreatedAt: now,
			UpdatedAt: now,
		}, nil).Once()

		req := &pb.CreateTenantRequest{Name: "Test Tenant"}
		resp, err := service.CreateTenant(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, tenantID.String(), resp.TenantId)
		assert.Equal(t, "Test Tenant", resp.Name)
		mockTenantRepo.AssertExpectations(t)
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		req := &pb.CreateTenantRequest{Name: ""}
		resp, err := service.CreateTenant(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// Test CreateAccount
func TestLedgerService_CreateAccount(t *testing.T) {
	ctx := context.Background()
	mockAccountRepo := new(MockAccountRepository)
	service := NewLedgerService(nil, mockAccountRepo, nil, nil)

	t.Run("successfully creates account", func(t *testing.T) {
		tenantID := uuid.New()
		accountID := uuid.New()
		now := time.Now()

		params := repository.CreateAccountParams{
			AccountNumber: "1000",
			Name:          "Cash",
			AccountTypeID: 1,
			CurrencyCode:  "USD",
		}

		mockAccountRepo.On("Create", ctx, tenantID, params).Return(&repository.Account{
			ID:            accountID,
			TenantID:      tenantID,
			AccountNumber: "1000",
			Name:          "Cash",
			AccountTypeID: 1,
			CurrencyCode:  "USD",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}, nil).Once()

		req := &pb.CreateAccountRequest{
			TenantId:      tenantID.String(),
			AccountNumber: "1000",
			Name:          "Cash",
			AccountTypeId: 1,
			CurrencyCode:  "USD",
		}
		resp, err := service.CreateAccount(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, accountID.String(), resp.AccountId)
		assert.Equal(t, "1000", resp.AccountNumber)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("returns error when tenant ID is invalid", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			TenantId:      "invalid-uuid",
			AccountNumber: "1000",
			Name:          "Cash",
		}
		resp, err := service.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("returns error when account number is empty", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			TenantId:      uuid.New().String(),
			AccountNumber: "",
			Name:          "Cash",
		}
		resp, err := service.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// Test CreateJournalEntry
func TestLedgerService_CreateJournalEntry(t *testing.T) {
	ctx := context.Background()
	mockJournalRepo := new(MockJournalRepository)
	service := NewLedgerService(nil, nil, mockJournalRepo, nil)

	t.Run("successfully creates journal entry", func(t *testing.T) {
		tenantID := uuid.New()
		journalID := uuid.New()
		account1ID := uuid.New()
		account2ID := uuid.New()
		now := time.Now()

		lines := []*repository.CreateJournalEntryLineParams{
			{
				AccountID:   account1ID,
				Debit:       decimal.NewFromInt(100),
				Credit:      decimal.Zero,
				Description: "Debit line",
			},
			{
				AccountID:   account2ID,
				Debit:       decimal.Zero,
				Credit:      decimal.NewFromInt(100),
				Description: "Credit line",
			},
		}

		params := repository.CreateJournalEntryParams{
			ReferenceNumber: "REF001",
			Description:     "Test entry",
			EntryDate:       now,
			Lines:           lines,
		}

		mockJournalRepo.On("Create", ctx, tenantID, mock.MatchedBy(func(p repository.CreateJournalEntryParams) bool {
			return p.ReferenceNumber == params.ReferenceNumber &&
				len(p.Lines) == 2
		})).Return(&repository.JournalEntry{
			ID:              journalID,
			TenantID:        tenantID,
			ReferenceNumber: "REF001",
			Description:     "Test entry",
			EntryDate:       now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}, nil).Once()

		req := &pb.CreateJournalEntryRequest{
			TenantId:        tenantID.String(),
			ReferenceNumber: "REF001",
			Description:     "Test entry",
			EntryDate:       timestamppb.New(now),
			Lines: []*pb.JournalEntryLine{
				{
					AccountId:   account1ID.String(),
					Debit:       "100",
					Credit:      "0",
					Description: "Debit line",
				},
				{
					AccountId:   account2ID.String(),
					Debit:       "0",
					Credit:      "100",
					Description: "Credit line",
				},
			},
		}
		resp, err := service.CreateJournalEntry(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, journalID.String(), resp.JournalEntryId)
		mockJournalRepo.AssertExpectations(t)
	})

	t.Run("returns error when less than 2 lines", func(t *testing.T) {
		req := &pb.CreateJournalEntryRequest{
			TenantId:        uuid.New().String(),
			ReferenceNumber: "REF001",
			Lines: []*pb.JournalEntryLine{
				{
					AccountId: uuid.New().String(),
					Debit:     "100",
					Credit:    "0",
				},
			},
		}
		resp, err := service.CreateJournalEntry(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// Test GetAccountBalance
func TestLedgerService_GetAccountBalance(t *testing.T) {
	ctx := context.Background()
	mockAccountRepo := new(MockAccountRepository)
	service := NewLedgerService(nil, mockAccountRepo, nil, nil)

	t.Run("successfully retrieves account balance", func(t *testing.T) {
		tenantID := uuid.New()
		accountID := uuid.New()
		now := time.Now()

		mockAccountRepo.On("GetBalance", ctx, tenantID, accountID).Return(&repository.AccountBalance{
			AccountID:     accountID,
			DebitBalance:  decimal.NewFromInt(1000),
			CreditBalance: decimal.NewFromInt(500),
			UpdatedAt:     now,
		}, nil).Once()

		req := &pb.GetAccountBalanceRequest{
			TenantId:  tenantID.String(),
			AccountId: accountID.String(),
		}
		resp, err := service.GetAccountBalance(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, accountID.String(), resp.AccountId)
		assert.Equal(t, "1000", resp.DebitBalance)
		assert.Equal(t, "500", resp.CreditBalance)
		assert.Equal(t, "500", resp.NetBalance) // 1000 - 500
		mockAccountRepo.AssertExpectations(t)
	})
}

// Test ListAccountTypes
func TestLedgerService_ListAccountTypes(t *testing.T) {
	ctx := context.Background()
	mockReferenceRepo := new(MockReferenceRepository)
	service := NewLedgerService(nil, nil, nil, mockReferenceRepo)

	t.Run("successfully lists account types", func(t *testing.T) {
		accountTypes := []*repository.AccountType{
			{ID: 1, Code: "ASSET", Name: "Asset", NormalBalance: "DEBIT"},
			{ID: 2, Code: "LIABILITY", Name: "Liability", NormalBalance: "CREDIT"},
		}

		mockReferenceRepo.On("ListAccountTypes", ctx).Return(accountTypes, nil).Once()

		req := &pb.ListAccountTypesRequest{}
		resp, err := service.ListAccountTypes(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.AccountTypes, 2)
		assert.Equal(t, "ASSET", resp.AccountTypes[0].Code)
		mockReferenceRepo.AssertExpectations(t)
	})
}

// Test ListCurrencies
func TestLedgerService_ListCurrencies(t *testing.T) {
	ctx := context.Background()
	mockReferenceRepo := new(MockReferenceRepository)
	service := NewLedgerService(nil, nil, nil, mockReferenceRepo)

	t.Run("successfully lists currencies", func(t *testing.T) {
		currencies := []*repository.Currency{
			{ID: 1, Code: "USD", Name: "US Dollar", Symbol: "$", Precision: 2},
			{ID: 2, Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2},
		}

		mockReferenceRepo.On("ListCurrencies", ctx).Return(currencies, nil).Once()

		req := &pb.ListCurrenciesRequest{}
		resp, err := service.ListCurrencies(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Currencies, 2)
		assert.Equal(t, "USD", resp.Currencies[0].Code)
		mockReferenceRepo.AssertExpectations(t)
	})
}
