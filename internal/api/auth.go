package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewAuthHandler(db *sql.DB, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{db: db, logger: logger}
}

type authInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input authInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errBadRequest(w, "invalid request body")
		return
	}

	if fields := validateStruct(input); fields != nil {
		errValidation(w, fields)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		errInternal(w)
		return
	}

	var id string
	err = h.db.QueryRowContext(
		r.Context(),
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		input.Email, hash,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			errConflict(w, "email already in use")
			return
		}
		h.logger.Error("failed to create user", "error", err)
		errInternal(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input authInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errBadRequest(w, "invalid request body")
		return
	}

	var userID, passwordHash string
	err := h.db.QueryRowContext(
		r.Context(),
		"SELECT id, password_hash FROM users WHERE email = $1",
		input.Email,
	).Scan(&userID, &passwordHash)

	if err == sql.ErrNoRows {
		errUnauthorized(w, "invalid credentials")
		return
	}
	if err != nil {
		h.logger.Error("failed to query user", "error", err)
		errInternal(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		errUnauthorized(w, "invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		h.logger.Error("failed to sign JWT", "error", err)
		errInternal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
