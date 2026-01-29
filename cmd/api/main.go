package main

import (
	"log/slog"
	"os"
	"os/signal" // <--- NEW: To listen for Ctrl+C
	"syscall"   // <--- NEW: System calls

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/handler"
	"github.com/ibrahimkeyboad/gopay/internal/adapter/middleware"
	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
	"github.com/ibrahimkeyboad/gopay/internal/core/config"
	"github.com/ibrahimkeyboad/gopay/internal/core/worker"
)

func main() {
	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Setup Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 3. Connect to Database
	dbPool, err := storage.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("❌ Database connection failed", "error", err)
		os.Exit(1)
	}
	// We do NOT defer dbPool.Close() here anymore. We close it manually on shutdown.

	// 4. Setup Repos & Handlers
	accountRepo := storage.NewAccountRepository(dbPool)
	ledgerRepo := storage.NewLedgerRepository(dbPool)

	accountHandler := &handler.AccountHandler{Repo: accountRepo}
	transactionHandler := &handler.TransactionHandler{Repo: ledgerRepo}
	mobileHandler := &handler.MobileMoneyHandler{Repo: ledgerRepo}
	paymentHandler := &handler.PaymentHandler{Repo: ledgerRepo}

	// 5. Setup Fiber
	app := fiber.New(fiber.Config{
		// This prevents the server from shutting down instantly
		DisableStartupMessage: true, 
	})
	
	app.Use(cors.New())
	app.Static("/", "./public")

	// 6. Routes
	api := app.Group("/v1")

	// Public
	api.Post("/accounts", accountHandler.CreateAccount)
	api.Post("/accounts/:id/keys", accountHandler.GenerateKey)
	api.Post("/charges", paymentHandler.MakeCharge)

	// Protected
	private := api.Use(middleware.Protected(dbPool))
	private.Post("/deposit", transactionHandler.Deposit)
	private.Post("/transfer", middleware.Idempotency(dbPool), transactionHandler.Transfer)
	private.Post("/mobile-money", middleware.Idempotency(dbPool), mobileHandler.InitializePayment)
	private.Get("/accounts/:id/transactions", transactionHandler.GetHistory)

	// 7. Start Worker
	worker.StartWebhookWorker(dbPool)

	// ==========================================
	// 🚀 GRACEFUL SHUTDOWN LOGIC STARTS HERE
	// ==========================================

	// Create a channel to listen for OS signals (Ctrl+C, Docker Stop)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run Server in a separate Goroutine so it doesn't block
	go func() {
		slog.Info("🚀 Server starting", "env", cfg.Env, "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("Server forced to shutdown", "error", err)
		}
	}()

	// Block here until we receive a stop signal
	<-stop
	slog.Info("🛑 Shutting down server...")

	// Close the Database Connection nicely
	dbPool.Close()
	slog.Info("✅ Database connection closed")

	// Tell Fiber to stop accepting new requests and finish active ones
	if err := app.Shutdown(); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	slog.Info("👋 Server exited successfully")
}