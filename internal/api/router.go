package api

import (
	"database/sql"
	"jarwise-backend/internal/api/handlers"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/repository"
	"jarwise-backend/internal/service"
	"net/http"
	"os"
	"strings"
)

var allowedOriginSet = map[string]struct{}{
	"http://localhost:5173": {},
	"http://127.0.0.1:5173": {},
}

type RouterOptions struct {
	DB             *sql.DB
	GoogleClientID string
	SecureCookies  bool
	Verifier       auth.GoogleTokenVerifier
}

func NewRouter() http.Handler {
	return NewRouterWithOptions(RouterOptions{
		GoogleClientID: os.Getenv("JARWISE_GOOGLE_CLIENT_ID"),
		SecureCookies:  strings.EqualFold(os.Getenv("JARWISE_SECURE_COOKIES"), "true"),
	})
}

func NewRouterWithOptions(options RouterOptions) http.Handler {
	mux := http.NewServeMux()

	dbConn := options.DB
	if dbConn == nil {
		var err error
		dbConn, err = db.InitDB("transactions.db")
		if err != nil {
			panic(err)
		}
	}

	googleVerifier := options.Verifier
	if googleVerifier == nil {
		googleVerifier = auth.NewHTTPGoogleVerifier(nil)
	}
	authService := auth.NewService(dbConn, googleVerifier, options.GoogleClientID, options.SecureCookies)
	authHandler := handlers.NewAuthHandler(authService)

	migrationSvc := service.NewMigrationService(dbConn)
	migrationHandler := handlers.NewMigrationHandler(migrationSvc)

	walletRepo := repository.NewSQLiteWalletRepository(dbConn)
	walletHandler := handlers.NewWalletHandler(walletRepo)

	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
	txService := service.NewTransactionService(txRepo, walletRepo)
	txHandler := handlers.NewTransactionHandler(txService)
	jarRepo := repository.NewSQLiteJarRepository(dbConn)
	reportService := service.NewReportService(txRepo, jarRepo, walletRepo)
	reportHandler := handlers.NewReportHandler(reportService)

	graphService := service.NewGraphService(txRepo)
	graphHandler := handlers.NewGraphHandler(graphService)

	chartService := service.NewChartService(txRepo)
	chartHandler := handlers.NewChartHandler(chartService)

	requireAuth := func(next http.HandlerFunc) http.Handler {
		return auth.RequireAuth(authService, http.HandlerFunc(next))
	}

	mux.HandleFunc("/api/v1/auth/google", authHandler.SignInWithGoogle)
	mux.Handle("/api/v1/auth/me", requireAuth(authHandler.Me))
	mux.Handle("/api/v1/auth/logout", requireAuth(authHandler.Logout))

	mux.Handle("/api/v1/migrations/money-manager/jobs", requireAuth(migrationHandler.CreateJob))
	mux.Handle("/api/v1/migrations/money-manager/jobs/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/confirm") {
			migrationHandler.ConfirmJob(w, r)
			return
		}
		migrationHandler.GetJob(w, r)
	}))

	mux.Handle("/api/v1/transfers", requireAuth(txHandler.CreateTransfer))
	mux.Handle("/api/v1/reports", requireAuth(reportHandler.GetReport))
	mux.Handle("/api/v1/reports/export", requireAuth(reportHandler.ExportReport))
	mux.Handle("/api/v1/graph/expenses", requireAuth(graphHandler.GetExpenseGraphData))
	mux.Handle("/api/v1/charts", requireAuth(chartHandler.GetChartData))
	mux.Handle("/api/v1/wallets/", requireAuth(walletHandler.HandleDelete))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return CORSMiddleware(mux)
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := isAllowedOrigin(origin); ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) (struct{}, bool) {
	value, ok := allowedOriginSet[origin]
	return value, ok
}
