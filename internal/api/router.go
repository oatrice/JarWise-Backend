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

	walletRepo := repository.NewSQLiteWalletRepository(dbConn)
	walletHandler := handlers.NewWalletHandler(walletRepo)

	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
	txService := service.NewTransactionService(txRepo, walletRepo)
	txHandler := handlers.NewTransactionHandler(txService)
	reportService := service.NewReportService(txRepo)
	reportHandler := handlers.NewReportHandler(reportService)

	graphService := service.NewGraphService(txRepo)
	graphHandler := handlers.NewGraphHandler(graphService)

	chartService := service.NewChartService(txRepo)
	chartHandler := handlers.NewChartHandler(chartService)

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
	mux.HandleFunc("/api/v1/graph/expenses", graphHandler.GetExpenseGraphData)
	mux.HandleFunc("/api/v1/charts", chartHandler.GetChartData)

	// Wallets
	mux.HandleFunc("/api/v1/wallets/", walletHandler.HandleDelete)

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}
