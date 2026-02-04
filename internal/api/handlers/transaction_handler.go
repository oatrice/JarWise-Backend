package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"net/http"
	"time"
)

type TransactionHandler struct {
	service service.TransactionService
}

func NewTransactionHandler(service service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

type CreateTransferRequest struct {
	FromWalletID string  `json:"from_wallet_id"`
	ToWalletID   string  `json:"to_wallet_id"`
	Amount       float64 `json:"amount"`
	Date         string  `json:"date"` // IOS8601
	Notes        string  `json:"notes"`
}

type CreateTransferResponse struct {
	ExpenseTransaction *models.Transaction `json:"expense_transaction"`
	IncomeTransaction  *models.Transaction `json:"income_transaction"`
}

func (h *TransactionHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.FromWalletID == "" || req.ToWalletID == "" || req.Amount <= 0 {
		http.Error(w, "Invalid input parameters", http.StatusBadRequest)
		return
	}

	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		// Try short date
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
	}

	expense, income, err := h.service.CreateTransfer(req.FromWalletID, req.ToWalletID, req.Amount, date, req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateTransferResponse{
		ExpenseTransaction: expense,
		IncomeTransaction:  income,
	})
}
