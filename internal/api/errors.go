package api

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type ValidationError struct {
	Error  string            `json:"error"`
	Code   string            `json:"code"`
	Fields map[string]string `json:"fields"`
}

func writeError(w http.ResponseWriter, message string, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Error: message,
		Code:  code,
	})
}

func errBadRequest(w http.ResponseWriter, message string) {
	// Return a 400 Bad Request
	writeError(w, message, "bad_request", http.StatusBadRequest)
}

func errUnprocessable(w http.ResponseWriter, message string) {
	// Return a 422 Unprocessable Entity
	writeError(w, message, "unprocessable", http.StatusUnprocessableEntity)
}

func errNotFound(w http.ResponseWriter, message string) {
	// Return a 404 Not Found
	writeError(w, message, "not_found", http.StatusNotFound)
}

func errConflict(w http.ResponseWriter, message string) {
	// Return a 409 Conflict
	writeError(w, message, "conflict", http.StatusConflict)
}

func errUnauthorized(w http.ResponseWriter, message string) {
	// Return a 401 Unauthorized
	writeError(w, message, "unauthorized", http.StatusUnauthorized)
}

func errInternal(w http.ResponseWriter) {
	// Return a 500 Internal Server Error
	writeError(w, "internal server error", "internal_error", http.StatusInternalServerError)
}

func errValidation(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(ValidationError{
		Error:  "validation failed",
		Code:   "validation_error",
		Fields: fields,
	})
}
