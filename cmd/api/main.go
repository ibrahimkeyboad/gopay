package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"

	"github.com/ibrahimkeyboad/gopay/internal/adapter/handler"
	"github.com/ibrahimkeyboad/gopay/internal/adapter/middleware"
	"github.com/ibrahimkeyboad/gopay/internal/adapter/storage"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	dbPool, err := storage.ConnectDB()
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer dbPool.Close()

	// Repositories
	accountRepo := storage.NewAccountRepository(dbPool)
	ledgerRepo := storage.NewLedgerRepository(dbPool) // NEW

	// Handlers
	accountHandler := &handler.AccountHandler{Repo: accountRepo}
	transactionHandler := &handler.TransactionHandler{Repo: ledgerRepo} // NEW

	app := fiber.New()
// 1. Enable CORS (So your HTML can talk to your Go API)
    app.Use(cors.New())

    // 2. Serve Static Files (The Checkout Page) <-- ADD THIS
    // This tells Go: "Serve any file inside the 'public' folder at the root URL"
    app.Static("/", "./public")

paymentHandler := &handler.PaymentHandler{Repo: ledgerRepo}

	api := app.Group("/v1")

	
	api.Post("/accounts", accountHandler.CreateAccount)
	api.Post("/accounts/:id/keys", accountHandler.GenerateKey)
	api.Post("/charges", paymentHandler.MakeCharge)


	// Protected Routes (The "Guard" is active here)
	// Only requests with valid API keys can pass
	private := api.Use(middleware.Protected(dbPool))


private.Post("/deposit", transactionHandler.Deposit)
	private.Post("/transfer", transactionHandler.Transfer)
	private.Get("/accounts/:id/transactions", transactionHandler.GetHistory)

	log.Println("🚀 Server running on port 3000")
	log.Fatal(app.Listen(":3000"))
}