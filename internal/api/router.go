package api

import (
	"jarwise-backend/internal/api/handlers"
	"jarwise-backend/internal/service"
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// Dependencies
	migrationSvc := service.NewMigrationService()
	migrationHandler := handlers.NewMigrationHandler(migrationSvc)

	// Routes
	mux.HandleFunc("/api/v1/migrations/money-manager", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		migrationHandler.HandleUpload(w, r)
	})

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}
