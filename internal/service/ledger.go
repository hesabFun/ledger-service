package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hesabFun/ledger/internal/repository"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/hesabFun/ledger/gen/go/ledger/v1"
)

// LedgerService implements the gRPC LedgerService
type LedgerService struct {
	pb.UnimplementedLedgerServiceServer
	tenantRepo    repository.TenantRepositoryInterface
	accountRepo   repository.AccountRepositoryInterface
	journalRepo   repository.JournalRepositoryInterface
	referenceRepo repository.ReferenceRepositoryInterface
}

// NewLedgerService creates a new ledger service
func NewLedgerService(
	tenantRepo repository.TenantRepositoryInterface,
	accountRepo repository.AccountRepositoryInterface,
	journalRepo repository.JournalRepositoryInterface,
	referenceRepo repository.ReferenceRepositoryInterface,
) *LedgerService {
	return &LedgerService{
		tenantRepo:    tenantRepo,
		accountRepo:   accountRepo,
		journalRepo:   journalRepo,
		referenceRepo: referenceRepo,
	}
}

// CreateTenant creates a new tenant
func (s *LedgerService) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.CreateTenantResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant name is required")
	}

	tenant, err := s.tenantRepo.Create(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	return &pb.CreateTenantResponse{
		TenantId:  tenant.ID.String(),
		Name:      tenant.Name,
		CreatedAt: timestamppb.New(tenant.CreatedAt),
	}, nil
}

// GetTenant retrieves a tenant by ID
func (s *LedgerService) GetTenant(ctx context.Context, req *pb.GetTenantRequest) (*pb.GetTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "tenant not found: %v", err)
	}

	return &pb.GetTenantResponse{
		Tenant: &pb.Tenant{
			TenantId:  tenant.ID.String(),
			Name:      tenant.Name,
			CreatedAt: timestamppb.New(tenant.CreatedAt),
			UpdatedAt: timestamppb.New(tenant.UpdatedAt),
		},
	}, nil
}

// CreateAccount creates a new account
func (s *LedgerService) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	if req.AccountNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "account number is required")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "account name is required")
	}

	params := repository.CreateAccountParams{
		AccountNumber: req.AccountNumber,
		Name:          req.Name,
		AccountTypeID: req.AccountTypeId,
		CurrencyCode:  req.CurrencyCode,
	}

	if req.Description != "" {
		params.Description = &req.Description
	}

	if req.ParentAccountId != nil {
		parentID, err := uuid.Parse(*req.ParentAccountId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent account ID")
		}
		params.ParentAccountID = &parentID
	}

	account, err := s.accountRepo.Create(ctx, tenantID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &pb.CreateAccountResponse{
		AccountId:     account.ID.String(),
		TenantId:      account.TenantID.String(),
		AccountNumber: account.AccountNumber,
		Name:          account.Name,
		CreatedAt:     timestamppb.New(account.CreatedAt),
	}, nil
}

// GetAccount retrieves an account by ID
func (s *LedgerService) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	account, err := s.accountRepo.GetByID(ctx, tenantID, accountID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "account not found: %v", err)
	}

	return &pb.GetAccountResponse{
		Account: s.accountToProto(account),
	}, nil
}

// ListAccounts retrieves accounts with optional filters
func (s *LedgerService) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}

	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var accountTypeID *int32
	if req.AccountTypeId != nil {
		accountTypeID = req.AccountTypeId
	}

	var currencyCode *string
	if req.CurrencyCode != nil {
		currencyCode = req.CurrencyCode
	}

	accounts, totalCount, err := s.accountRepo.List(ctx, tenantID, accountTypeID, currencyCode, pageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, account := range accounts {
		pbAccounts[i] = s.accountToProto(account)
	}

	return &pb.ListAccountsResponse{
		Accounts:   pbAccounts,
		TotalCount: int32(totalCount),
	}, nil
}

// GetAccountBalance retrieves the balance for an account
func (s *LedgerService) GetAccountBalance(ctx context.Context, req *pb.GetAccountBalanceRequest) (*pb.GetAccountBalanceResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	balance, err := s.accountRepo.GetBalance(ctx, tenantID, accountID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "balance not found: %v", err)
	}

	netBalance := balance.DebitBalance.Sub(balance.CreditBalance)

	return &pb.GetAccountBalanceResponse{
		AccountId:     balance.AccountID.String(),
		DebitBalance:  balance.DebitBalance.String(),
		CreditBalance: balance.CreditBalance.String(),
		NetBalance:    netBalance.String(),
		UpdatedAt:     timestamppb.New(balance.UpdatedAt),
	}, nil
}

// CreateJournalEntry creates a new journal entry
func (s *LedgerService) CreateJournalEntry(ctx context.Context, req *pb.CreateJournalEntryRequest) (*pb.CreateJournalEntryResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	if len(req.Lines) < 2 {
		return nil, status.Error(codes.InvalidArgument, "journal entry must have at least two lines")
	}

	lines := make([]*repository.CreateJournalEntryLineParams, len(req.Lines))
	for i, line := range req.Lines {
		accountID, err := uuid.Parse(line.AccountId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid account ID at line %d", i)
		}

		debit, err := decimal.NewFromString(line.Debit)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid debit amount at line %d", i)
		}

		credit, err := decimal.NewFromString(line.Credit)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid credit amount at line %d", i)
		}

		lines[i] = &repository.CreateJournalEntryLineParams{
			AccountID:   accountID,
			Debit:       debit,
			Credit:      credit,
			Description: line.Description,
		}
	}

	var metadata map[string]interface{}
	if req.Metadata != nil && *req.Metadata != "" {
		if err := json.Unmarshal([]byte(*req.Metadata), &metadata); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid metadata JSON")
		}
	}

	params := repository.CreateJournalEntryParams{
		ReferenceNumber: req.ReferenceNumber,
		Description:     req.Description,
		EntryDate:       req.EntryDate.AsTime(),
		Metadata:        metadata,
		Lines:           lines,
	}

	entry, err := s.journalRepo.Create(ctx, tenantID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create journal entry: %v", err)
	}

	return &pb.CreateJournalEntryResponse{
		JournalEntryId:  entry.ID.String(),
		TenantId:        entry.TenantID.String(),
		ReferenceNumber: entry.ReferenceNumber,
		EntryDate:       timestamppb.New(entry.EntryDate),
		CreatedAt:       timestamppb.New(entry.CreatedAt),
	}, nil
}

// GetJournalEntry retrieves a journal entry by ID
func (s *LedgerService) GetJournalEntry(ctx context.Context, req *pb.GetJournalEntryRequest) (*pb.GetJournalEntryResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	journalEntryID, err := uuid.Parse(req.JournalEntryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid journal entry ID")
	}

	entry, err := s.journalRepo.GetByID(ctx, tenantID, journalEntryID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "journal entry not found: %v", err)
	}

	return &pb.GetJournalEntryResponse{
		JournalEntry: s.journalEntryToProto(entry),
	}, nil
}

// ListJournalEntries retrieves journal entries with optional filters
func (s *LedgerService) ListJournalEntries(ctx context.Context, req *pb.ListJournalEntriesRequest) (*pb.ListJournalEntriesResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant ID")
	}

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}

	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var accountID *uuid.UUID
	if req.AccountId != nil {
		aid, err := uuid.Parse(*req.AccountId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid account ID")
		}
		accountID = &aid
	}

	var fromTime, toTime *time.Time
	if req.FromDate != nil {
		t := req.FromDate.AsTime()
		fromTime = &t
	}
	if req.ToDate != nil {
		t := req.ToDate.AsTime()
		toTime = &t
	}

	entries, totalCount, err := s.journalRepo.List(ctx, tenantID, accountID, fromTime, toTime, pageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list journal entries: %v", err)
	}

	pbEntries := make([]*pb.JournalEntry, len(entries))
	for i, entry := range entries {
		pbEntries[i] = s.journalEntryToProto(entry)
	}

	return &pb.ListJournalEntriesResponse{
		JournalEntries: pbEntries,
		TotalCount:     int32(totalCount),
	}, nil
}

// ListAccountTypes retrieves all account types
func (s *LedgerService) ListAccountTypes(ctx context.Context, req *pb.ListAccountTypesRequest) (*pb.ListAccountTypesResponse, error) {
	accountTypes, err := s.referenceRepo.ListAccountTypes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list account types: %v", err)
	}

	pbAccountTypes := make([]*pb.AccountType, len(accountTypes))
	for i, at := range accountTypes {
		pbAccountTypes[i] = &pb.AccountType{
			Id:            at.ID,
			Code:          at.Code,
			Name:          at.Name,
			NormalBalance: at.NormalBalance,
		}
	}

	return &pb.ListAccountTypesResponse{
		AccountTypes: pbAccountTypes,
	}, nil
}

// ListCurrencies retrieves all currencies
func (s *LedgerService) ListCurrencies(ctx context.Context, req *pb.ListCurrenciesRequest) (*pb.ListCurrenciesResponse, error) {
	currencies, err := s.referenceRepo.ListCurrencies(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list currencies: %v", err)
	}

	pbCurrencies := make([]*pb.Currency, len(currencies))
	for i, c := range currencies {
		pbCurrencies[i] = &pb.Currency{
			Id:        c.ID,
			Code:      c.Code,
			Name:      c.Name,
			Symbol:    c.Symbol,
			Precision: c.Precision,
		}
	}

	return &pb.ListCurrenciesResponse{
		Currencies: pbCurrencies,
	}, nil
}

// Helper functions to convert domain models to protobuf messages

func (s *LedgerService) accountToProto(account *repository.Account) *pb.Account {
	pbAccount := &pb.Account{
		AccountId:     account.ID.String(),
		TenantId:      account.TenantID.String(),
		AccountNumber: account.AccountNumber,
		Name:          account.Name,
		Description:   "",
		AccountTypeId: account.AccountTypeID,
		CurrencyCode:  account.CurrencyCode,
		IsActive:      account.IsActive,
		CreatedAt:     timestamppb.New(account.CreatedAt),
		UpdatedAt:     timestamppb.New(account.UpdatedAt),
	}

	if account.Description != nil {
		pbAccount.Description = *account.Description
	}

	if account.ParentAccountID != nil {
		parentID := account.ParentAccountID.String()
		pbAccount.ParentAccountId = &parentID
	}

	return pbAccount
}

func (s *LedgerService) journalEntryToProto(entry *repository.JournalEntry) *pb.JournalEntry {
	lines := make([]*pb.JournalEntryLine, len(entry.Lines))
	for i, line := range entry.Lines {
		lineID := line.ID.String()
		createdAt := timestamppb.New(line.CreatedAt)

		lines[i] = &pb.JournalEntryLine{
			LineId:      &lineID,
			AccountId:   line.AccountID.String(),
			Debit:       line.Debit.String(),
			Credit:      line.Credit.String(),
			Description: line.Description,
			CreatedAt:   createdAt,
		}
	}

	pbEntry := &pb.JournalEntry{
		JournalEntryId:  entry.ID.String(),
		TenantId:        entry.TenantID.String(),
		ReferenceNumber: entry.ReferenceNumber,
		Description:     entry.Description,
		EntryDate:       timestamppb.New(entry.EntryDate),
		Lines:           lines,
		CreatedAt:       timestamppb.New(entry.CreatedAt),
		UpdatedAt:       timestamppb.New(entry.UpdatedAt),
	}

	if entry.Metadata != nil {
		metadataBytes, err := json.Marshal(entry.Metadata)
		if err == nil {
			metadataStr := string(metadataBytes)
			pbEntry.Metadata = &metadataStr
		}
	}

	return pbEntry
}
