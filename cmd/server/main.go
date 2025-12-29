package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hesabFun/ledger/internal/config"
	"github.com/hesabFun/ledger/internal/db"
	"github.com/hesabFun/ledger/internal/repository"
	"github.com/hesabFun/ledger/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/hesabFun/ledger/gen/go/ledger/v1"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	ctx := context.Background()
	database, err := db.New(ctx, &cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Successfully connected to database")

	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(database)
	accountRepo := repository.NewAccountRepository(database)
	journalRepo := repository.NewJournalRepository(database)
	referenceRepo := repository.NewReferenceRepository(database)

	// Initialize service
	ledgerService := service.NewLedgerService(
		tenantRepo,
		accountRepo,
		journalRepo,
		referenceRepo,
	)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024), // 10MB
	)

	// Register service
	pb.RegisterLedgerServiceServer(grpcServer, ledgerService)

	// Enable reflection for grpcurl and other tools
	reflection.Register(grpcServer)

	// Create listener
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting gRPC server on %s", address)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Gracefully stop the server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-stopped:
		log.Println("Server stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}
}
