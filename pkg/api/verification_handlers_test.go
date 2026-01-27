package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVerificationHandlers verifies handler initialization
func TestNewVerificationHandlers(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.verifier)
	assert.NotNil(t, handlers.logger)
}

// TestVerificationHandlers_RegisterRoutes verifies all routes are registered
func TestVerificationHandlers_RegisterRoutes(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/plugins/test-plugin/versions/1.0.0/verify"},
		{"GET", "/api/v1/verifications/1"},
		{"GET", "/api/v1/verifications"},
		{"POST", "/api/v1/verifications/1/approve"},
		{"POST", "/api/v1/verifications/1/reject"},
		{"GET", "/api/v1/verifications/stats"},
		{"GET", "/api/v1/plugins/test-plugin/security-score"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}
}

// TestSubmitVerification_InvalidJSON tests with invalid JSON body
func TestSubmitVerification_InvalidJSON(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("POST", "/api/v1/plugins/test-plugin/versions/1.0.0/verify",
		bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "test-plugin", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.submitVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSubmitVerification_DBError tests database error handling
func TestSubmitVerification_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	// Expect INSERT query to fail
	mock.ExpectExec("INSERT INTO plugin_verifications").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress error logs in test output
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(SubmitVerificationRequest{
		SubmittedBy: "test-user",
		AutoApprove: false,
	})
	req := httptest.NewRequest("POST", "/api/v1/plugins/test-plugin/versions/1.0.0/verify",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "test-plugin", "version": "1.0.0"})
	w := httptest.NewRecorder()

	handlers.submitVerification(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetVerification_InvalidID tests with invalid verification ID
func TestGetVerification_InvalidID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.getVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetVerification_NotFound tests when verification not found
// Note: GetVerificationStatus wraps sql.ErrNoRows, so the handler returns 500 instead of 404
func TestGetVerification_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	// Expect SELECT query to return no rows
	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.getVerification(w, req)

	// Due to error wrapping in GetVerificationStatus, this returns 500 not 404
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "sql: no rows")
}

// TestGetVerification_DBError tests database error handling
func TestGetVerification_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	// Expect SELECT query to fail
	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.getVerification(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListVerifications_NoStatus tests listing all verifications
func TestListVerifications_NoStatus(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{"id", "plugin_id", "version", "status", "submitted_at", "completed_at"}).
		AddRow(1, "plugin-1", "1.0.0", "approved", "2024-01-01T00:00:00Z", "2024-01-01T00:01:00Z").
		AddRow(2, "plugin-2", "1.0.0", "pending", "2024-01-02T00:00:00Z", nil)

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WithArgs(20, 0).
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications", nil)
	w := httptest.NewRecorder()

	handlers.listVerifications(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ListVerificationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, 20, response.Limit)
	assert.Equal(t, 0, response.Offset)
}

// TestListVerifications_WithStatus tests listing verifications with status filter
func TestListVerifications_WithStatus(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{"id", "plugin_id", "version", "status", "submitted_at", "completed_at"}).
		AddRow(1, "plugin-1", "1.0.0", "pending", "2024-01-01T00:00:00Z", nil)

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WithArgs("pending", 20, 0).
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications?status=pending", nil)
	w := httptest.NewRecorder()

	handlers.listVerifications(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestListVerifications_WithPagination tests listing with custom limit and offset
func TestListVerifications_WithPagination(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{"id", "plugin_id", "version", "status", "submitted_at", "completed_at"})

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WithArgs(10, 5).
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications?limit=10&offset=5", nil)
	w := httptest.NewRecorder()

	handlers.listVerifications(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ListVerificationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 10, response.Limit)
	assert.Equal(t, 5, response.Offset)
}

// TestListVerifications_DBError tests database error handling
func TestListVerifications_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications", nil)
	w := httptest.NewRecorder()

	handlers.listVerifications(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestApproveVerification_InvalidID tests with invalid verification ID
func TestApproveVerification_InvalidID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("POST", "/api/v1/verifications/invalid/approve", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.approveVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestApproveVerification_InvalidJSON tests with invalid JSON body
func TestApproveVerification_InvalidJSON(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("POST", "/api/v1/verifications/1/approve",
		bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.approveVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestApproveVerification_MissingApprovedBy tests with missing approved_by field
func TestApproveVerification_MissingApprovedBy(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(ApprovalRequest{
		ApprovedBy: "",
		Reason:     "Looks good",
	})
	req := httptest.NewRequest("POST", "/api/v1/verifications/1/approve",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.approveVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestApproveVerification_DBError tests database error handling
func TestApproveVerification_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectExec("UPDATE plugin_verifications").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(ApprovalRequest{
		ApprovedBy: "admin",
		Reason:     "Approved",
	})
	req := httptest.NewRequest("POST", "/api/v1/verifications/1/approve",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.approveVerification(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRejectVerification_InvalidID tests with invalid verification ID
func TestRejectVerification_InvalidID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("POST", "/api/v1/verifications/invalid/reject", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.rejectVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRejectVerification_InvalidJSON tests with invalid JSON body
func TestRejectVerification_InvalidJSON(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("POST", "/api/v1/verifications/1/reject",
		bytes.NewBufferString("invalid json"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.rejectVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRejectVerification_MissingApprovedBy tests with missing approved_by field
func TestRejectVerification_MissingApprovedBy(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(ApprovalRequest{
		ApprovedBy: "",
		Reason:     "Security issues",
	})
	req := httptest.NewRequest("POST", "/api/v1/verifications/1/reject",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.rejectVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRejectVerification_MissingReason tests with missing reason field
func TestRejectVerification_MissingReason(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(ApprovalRequest{
		ApprovedBy: "admin",
		Reason:     "",
	})
	req := httptest.NewRequest("POST", "/api/v1/verifications/1/reject",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.rejectVerification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRejectVerification_DBError tests database error handling
func TestRejectVerification_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectExec("UPDATE plugin_verifications").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	reqBody, _ := json.Marshal(ApprovalRequest{
		ApprovedBy: "admin",
		Reason:     "Security issues",
	})
	req := httptest.NewRequest("POST", "/api/v1/verifications/1/reject",
		bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.rejectVerification(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetVerificationStats_Success tests successful stats retrieval
func TestGetVerificationStats_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{"total", "pending", "in_progress", "approved", "rejected", "review_required", "avg_seconds"}).
		AddRow(100, 10, 5, 70, 10, 5, 120.5)

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/stats", nil)
	w := httptest.NewRecorder()

	handlers.getVerificationStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response VerificationStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 100, response.TotalVerifications)
	assert.Equal(t, 10, response.Pending)
	assert.Equal(t, 5, response.InProgress)
	assert.Equal(t, 70, response.Approved)
	assert.Equal(t, 10, response.Rejected)
	assert.Equal(t, 5, response.ReviewRequired)
	assert.NotEmpty(t, response.AvgProcessingTime)
}

// TestGetVerificationStats_NoData tests stats with no verification data
func TestGetVerificationStats_NoData(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{"total", "pending", "in_progress", "approved", "rejected", "review_required", "avg_seconds"}).
		AddRow(0, 0, 0, 0, 0, 0, nil)

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/stats", nil)
	w := httptest.NewRecorder()

	handlers.getVerificationStats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response VerificationStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "N/A", response.AvgProcessingTime)
}

// TestGetVerificationStats_DBError tests database error handling
func TestGetVerificationStats_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectQuery("SELECT (.+) FROM plugin_verifications").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/verifications/stats", nil)
	w := httptest.NewRecorder()

	handlers.getVerificationStats(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetPluginSecurityScore_Success tests successful security score retrieval
func TestGetPluginSecurityScore_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	rows := sqlmock.NewRows([]string{
		"plugin_id", "plugin_name", "security_level", "total_verifications",
		"approved_verifications", "rejected_verifications", "approval_rate",
		"total_security_issues", "critical_issues", "high_issues", "last_verified_at",
	}).AddRow(
		"test-plugin", "Test Plugin", "high", 10,
		8, 2, 0.8,
		3, 0, 1, "2024-01-01T00:00:00Z",
	)

	mock.ExpectQuery("SELECT (.+) FROM plugin_security_scores").
		WithArgs("test-plugin").
		WillReturnRows(rows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/plugins/test-plugin/security-score", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-plugin"})
	w := httptest.NewRecorder()

	handlers.getPluginSecurityScore(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SecurityScoreResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", response.PluginID)
	assert.Equal(t, "Test Plugin", response.PluginName)
	assert.Equal(t, "high", response.SecurityLevel)
	assert.Equal(t, 10, response.TotalVerifications)
	assert.Equal(t, 8, response.ApprovedVerifications)
	assert.Equal(t, 2, response.RejectedVerifications)
	assert.Equal(t, 0.8, response.ApprovalRate)
	assert.Equal(t, 3, response.TotalSecurityIssues)
	assert.Equal(t, 0, response.CriticalIssues)
	assert.Equal(t, 1, response.HighIssues)
}

// TestGetPluginSecurityScore_NotFound tests when plugin not found
func TestGetPluginSecurityScore_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectQuery("SELECT (.+) FROM plugin_security_scores").
		WithArgs("nonexistent-plugin").
		WillReturnError(sql.ErrNoRows)

	logger := logrus.New()
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/plugins/nonexistent-plugin/security-score", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent-plugin"})
	w := httptest.NewRecorder()

	handlers.getPluginSecurityScore(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

// TestGetPluginSecurityScore_DBError tests database error handling
func TestGetPluginSecurityScore_DBError(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()
	defer mock.ExpectClose()

	mock.ExpectQuery("SELECT (.+) FROM plugin_security_scores").
		WithArgs("test-plugin").
		WillReturnError(sql.ErrConnDone)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handlers := NewVerificationHandlers(db, logger)

	req := httptest.NewRequest("GET", "/api/v1/plugins/test-plugin/security-score", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-plugin"})
	w := httptest.NewRecorder()

	handlers.getPluginSecurityScore(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestParseIntParam tests the helper function
func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name         string
		param        string
		defaultValue int
		expected     int
	}{
		{"empty string", "", 10, 10},
		{"valid number", "25", 10, 25},
		{"invalid number", "abc", 10, 10},
		{"negative number", "-5", 10, -5},
		{"zero", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntParam(tt.param, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatDuration tests the helper function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		contains string
	}{
		{"seconds", 45.5, "s"},
		{"minutes", 120.0, "m"},
		{"hours", 7200.0, "h"},
		{"zero", 0.0, "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.seconds)
			assert.Contains(t, result, tt.contains)
		})
	}
}
