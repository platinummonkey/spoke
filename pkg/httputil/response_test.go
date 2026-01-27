package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}

	err := WriteJSON(w, http.StatusOK, data)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "success")
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	err := errors.New("test error")

	WriteError(w, http.StatusBadRequest, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "test error")
}

func TestWriteErrorMessage(t *testing.T) {
	w := httptest.NewRecorder()

	WriteErrorMessage(w, http.StatusNotFound, "resource not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "resource not found")
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteValidationError(w, "invalid input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid input")
}

func TestWriteNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNotFoundError(w, "user not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "user not found")
}

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	err := errors.New("internal error")

	WriteInternalError(w, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal error")
}

func TestWriteCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"id": 123}

	err := WriteCreated(w, data)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "123")
}

func TestWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	err := WriteSuccess(w, data)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestWriteNoContent(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNoContent(w)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestWriteSuccessMessage(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteSuccessMessage(w, "Operation completed", map[string]int{"count": 5})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Operation completed")
	assert.Contains(t, w.Body.String(), "success")
}

func TestWriteUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()

	WriteUnauthorized(w, "invalid credentials")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credentials")
}

func TestWriteForbidden(t *testing.T) {
	w := httptest.NewRecorder()

	WriteForbidden(w, "access denied")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "access denied")
}

func TestWriteConflict(t *testing.T) {
	w := httptest.NewRecorder()

	WriteConflict(w, "resource already exists")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "resource already exists")
}

func TestWriteTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()

	WriteTooManyRequests(w, "rate limit exceeded")

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestWriteServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()

	WriteServiceUnavailable(w, "service unavailable")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "service unavailable")
}
