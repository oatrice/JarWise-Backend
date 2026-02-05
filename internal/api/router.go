package api

import (
	"jarwise-backend/internal/api/handlers"
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/repository"
	"jarwise-backend/internal/service"
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// Infrastructure
	dbConn, err := db.InitDB("transactions.db")
	if err != nil {
		// In a real app we might panic or handle differently
		panic(err)
	}

	// Dependencies
	migrationSvc := service.NewMigrationService()
	migrationHandler := handlers.NewMigrationHandler(migrationSvc)

	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
	txService := service.NewTransactionService(txRepo)
	txHandler := handlers.NewTransactionHandler(txService)
	reportService := service.NewReportService(txRepo)
	reportHandler := handlers.NewReportHandler(reportService)

	// Routes
	mux.HandleFunc("/api/v1/migrations/money-manager", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		migrationHandler.HandleUpload(w, r)
	})

	mux.HandleFunc("/api/v1/transfers", txHandler.CreateTransfer)
	mux.HandleFunc("/api/v1/reports", reportHandler.GetReport)

	// Wallets (Mock for Manual Verification)
	mux.HandleFunc("/api/wallets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"1","name":"Cash","balance":100.0},{"id":"2","name":"Bank","balance":5000.0}]`))
	})

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}
