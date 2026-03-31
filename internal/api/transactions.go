package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Transaction struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TransactionHandler struct {
	db *sql.DB
}

func NewTransactionHandler(db *sql.DB) *TransactionHandler {
	return &TransactionHandler{db: db}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	_, err := uuid.Parse(accountID)
	if accountID == "" || err != nil {
		http.Error(w, "Account ID is required and must be a valid UUID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), "SELECT * FROM transactions WHERE account_id = $1 ORDER BY created_at DESC", accountID)
	if err != nil {
		http.Error(w, "failed to query transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	txns := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			http.Error(w, "failed to scan transaction", http.StatusInternalServerError)
			return
		}
		txns = append(txns, t)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txns)
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	txID := chi.URLParam(r, "txID")

	if accountID == "" || txID == "" {
		http.Error(w, "Account ID and Transaction ID are required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(accountID); err != nil {
		http.Error(w, "Account ID must be a valid UUID", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(txID); err != nil {
		http.Error(w, "Transaction ID must be a valid UUID", http.StatusBadRequest)
		return
	}

	var tx Transaction
	err := h.db.QueryRowContext(
		r.Context(),
		"SELECT * FROM transactions WHERE id = $1 AND account_id = $2", txID, accountID,
	).Scan(&tx.ID, &tx.AccountID, &tx.Amount, &tx.Type, &tx.Description, &tx.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to query transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tx)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(accountID); err != nil {
		http.Error(w, "Account ID must be a valid UUID", http.StatusBadRequest)
		return
	}

	var input struct {
		Amount      float64 `json:"amount"`
		Type        string  `json:"type"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusUnprocessableEntity)
		return
	}

	if input.Type != "deposit" && input.Type != "withdrawal" {
		http.Error(w, "Type must be 'deposit' or 'withdrawal'", http.StatusUnprocessableEntity)
		return
	}

	dbTx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer dbTx.Rollback()

	var transaction Transaction
	err = dbTx.QueryRowContext(
		r.Context(),
		"INSERT INTO transactions (account_id, amount, type, description) VALUES ($1, $2, $3, $4) RETURNING id, account_id, amount, type, description, created_at",
		accountID, input.Amount, input.Type, input.Description,
	).Scan(&transaction.ID, &transaction.AccountID, &transaction.Amount, &transaction.Type, &transaction.Description, &transaction.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to create transaction", http.StatusInternalServerError)
		return
	}

	balanceChange := input.Amount
	if input.Type == "withdrawal" {
		balanceChange = -input.Amount
	}

	_, err = dbTx.ExecContext(
		r.Context(),
		"UPDATE accounts SET balance = balance + $1 WHERE id = $2",
		balanceChange, accountID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "failed to update account balance", http.StatusInternalServerError)
		return
	}

	if err := dbTx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}
