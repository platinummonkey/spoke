// Package httputil provides HTTP handler utilities for consistent error handling,
// JSON encoding/decoding, and request parsing.
package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteError writes a JSON error response with the given status code
func WriteError(w http.ResponseWriter, status int, err error) {
	WriteErrorMessage(w, status, err.Error())
}

// WriteErrorMessage writes a JSON error response with a custom message
func WriteErrorMessage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// WriteValidationError writes a validation error response (400 Bad Request)
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusBadRequest, message)
}

// WriteNotFoundError writes a not found error response (404 Not Found)
func WriteNotFoundError(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusNotFound, message)
}

// WriteInternalError writes an internal server error response (500 Internal Server Error)
func WriteInternalError(w http.ResponseWriter, err error) {
	WriteError(w, http.StatusInternalServerError, err)
}

// WriteCreated writes a successful creation response (201 Created) with JSON data
func WriteCreated(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusCreated, data)
}

// WriteSuccess writes a successful response (200 OK) with JSON data
func WriteSuccess(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusOK, data)
}

// WriteNoContent writes a successful response with no content (204 No Content)
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// WriteDetailedError writes a detailed error response with additional context
func WriteDetailedError(w http.ResponseWriter, status int, err error, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   err.Error(),
		Details: details,
	})
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// WriteSuccessMessage writes a success response with a message
func WriteSuccessMessage(w http.ResponseWriter, message string, data interface{}) error {
	return WriteJSON(w, http.StatusOK, SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// WriteBadRequest writes a bad request error (400)
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusBadRequest, message)
}

// WriteUnauthorized writes an unauthorized error (401)
func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusUnauthorized, message)
}

// WriteForbidden writes a forbidden error (403)
func WriteForbidden(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusForbidden, message)
}

// WriteConflict writes a conflict error (409)
func WriteConflict(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusConflict, message)
}

// WriteTooManyRequests writes a rate limit error (429)
func WriteTooManyRequests(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusTooManyRequests, message)
}

// WriteServiceUnavailable writes a service unavailable error (503)
func WriteServiceUnavailable(w http.ResponseWriter, message string) {
	WriteErrorMessage(w, http.StatusServiceUnavailable, message)
}

// WriteJSONOrError writes JSON on success or error on failure
func WriteJSONOrError(w http.ResponseWriter, status int, data interface{}, errMsg string) {
	if err := WriteJSON(w, status, data); err != nil {
		WriteInternalError(w, fmt.Errorf("%s: %w", errMsg, err))
	}
}
