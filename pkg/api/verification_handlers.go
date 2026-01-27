package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/httputil"
	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/sirupsen/logrus"
)

// VerificationHandlers handles plugin verification API endpoints
type VerificationHandlers struct {
	verifier *plugins.Verifier
	logger   *logrus.Logger
}

// NewVerificationHandlers creates a new verification handlers instance
func NewVerificationHandlers(db *sql.DB, logger *logrus.Logger) *VerificationHandlers {
	return &VerificationHandlers{
		verifier: plugins.NewVerifier(db, logger),
		logger:   logger,
	}
}

// RegisterRoutes registers verification API routes
func (h *VerificationHandlers) RegisterRoutes(r *mux.Router) {
	// Verification requests
	r.HandleFunc("/api/v1/plugins/{id}/versions/{version}/verify", h.submitVerification).Methods("POST")
	r.HandleFunc("/api/v1/verifications/{id}", h.getVerification).Methods("GET")
	r.HandleFunc("/api/v1/verifications", h.listVerifications).Methods("GET")

	// Manual approval/rejection (admin only)
	r.HandleFunc("/api/v1/verifications/{id}/approve", h.approveVerification).Methods("POST")
	r.HandleFunc("/api/v1/verifications/{id}/reject", h.rejectVerification).Methods("POST")

	// Verification statistics
	r.HandleFunc("/api/v1/verifications/stats", h.getVerificationStats).Methods("GET")
	r.HandleFunc("/api/v1/plugins/{id}/security-score", h.getPluginSecurityScore).Methods("GET")
}

// SubmitVerificationRequest contains the request body for submitting a verification
type SubmitVerificationRequest struct {
	SubmittedBy string `json:"submitted_by"`
	AutoApprove bool   `json:"auto_approve"`
}

// SubmitVerificationResponse contains the response for a verification submission
type SubmitVerificationResponse struct {
	VerificationID int64  `json:"verification_id"`
	Status         string `json:"status"`
	Message        string `json:"message"`
}

// submitVerification handles POST /api/v1/plugins/{id}/versions/{version}/verify
func (h *VerificationHandlers) submitVerification(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	pluginID := vars["id"]
	version := vars["version"]

	var req SubmitVerificationRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Create verification request
	verificationReq := &plugins.VerificationRequest{
		PluginID:    pluginID,
		Version:     version,
		SubmittedBy: req.SubmittedBy,
		AutoApprove: req.AutoApprove,
	}

	verificationID, err := h.verifier.SubmitForVerification(r.Context(), verificationReq)
	if err != nil {
		h.logger.Errorf("Failed to submit verification: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	response := SubmitVerificationResponse{
		VerificationID: verificationID,
		Status:         "pending",
		Message:        "Verification request submitted successfully",
	}

	httputil.WriteJSON(w, http.StatusCreated, response)
}

// VerificationResponse contains detailed verification information
type VerificationResponse struct {
	VerificationID   int64                       `json:"verification_id"`
	PluginID         string                      `json:"plugin_id"`
	Version          string                      `json:"version"`
	Status           string                      `json:"status"`
	SecurityLevel    string                      `json:"security_level,omitempty"`
	ManifestErrors   []plugins.ValidationError   `json:"manifest_errors,omitempty"`
	SecurityIssues   []plugins.SecurityIssue     `json:"security_issues,omitempty"`
	Reason           string                      `json:"reason,omitempty"`
	SubmittedAt      string                      `json:"submitted_at"`
	StartedAt        string                      `json:"started_at,omitempty"`
	CompletedAt      string                      `json:"completed_at,omitempty"`
	ProcessingTimeMs int64                       `json:"processing_time_ms,omitempty"`
}

// getVerification handles GET /api/v1/verifications/{id}
func (h *VerificationHandlers) getVerification(w http.ResponseWriter, r *http.Request) {
	verificationID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	result, err := h.verifier.GetVerificationStatus(r.Context(), verificationID)
	if err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteNotFoundError(w, "Verification not found")
			return
		}
		h.logger.Errorf("Failed to get verification: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	response := VerificationResponse{
		VerificationID:   result.VerificationID,
		PluginID:         result.PluginID,
		Version:          result.Version,
		Status:           result.Status,
		SecurityLevel:    result.SecurityLevel,
		ManifestErrors:   result.ManifestErrors,
		SecurityIssues:   result.SecurityIssues,
		Reason:           result.Reason,
		SubmittedAt:      result.StartedAt.Format("2006-01-02T15:04:05Z"),
		ProcessingTimeMs: result.ProcessingTime.Milliseconds(),
	}

	if !result.CompletedAt.IsZero() {
		response.CompletedAt = result.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	httputil.WriteSuccess(w, response)
}

// ListVerificationsResponse contains a list of verifications
type ListVerificationsResponse struct {
	Verifications []VerificationSummary `json:"verifications"`
	Total         int                   `json:"total"`
	Limit         int                   `json:"limit"`
	Offset        int                   `json:"offset"`
}

// VerificationSummary contains summary information about a verification
type VerificationSummary struct {
	VerificationID int64  `json:"verification_id"`
	PluginID       string `json:"plugin_id"`
	Version        string `json:"version"`
	Status         string `json:"status"`
	SubmittedAt    string `json:"submitted_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

// listVerifications handles GET /api/v1/verifications
func (h *VerificationHandlers) listVerifications(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	limit := parseIntParam(r.URL.Query().Get("limit"), 20)
	offset := parseIntParam(r.URL.Query().Get("offset"), 0)

	// Build query
	query := `
		SELECT id, plugin_id, version, status, submitted_at, completed_at
		FROM plugin_verifications
		WHERE 1=1
	`
	args := []interface{}{}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY submitted_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.verifier.GetDB().QueryContext(r.Context(), query, args...)
	if err != nil {
		h.logger.Errorf("Failed to list verifications: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	var verifications []VerificationSummary
	for rows.Next() {
		var v VerificationSummary
		var completedAt sql.NullString

		err := rows.Scan(&v.VerificationID, &v.PluginID, &v.Version, &v.Status, &v.SubmittedAt, &completedAt)
		if err != nil {
			continue
		}

		if completedAt.Valid {
			v.CompletedAt = completedAt.String
		}

		verifications = append(verifications, v)
	}

	response := ListVerificationsResponse{
		Verifications: verifications,
		Total:         len(verifications),
		Limit:         limit,
		Offset:        offset,
	}

	httputil.WriteSuccess(w, response)
}

// ApprovalRequest contains the request body for approval/rejection
type ApprovalRequest struct {
	ApprovedBy string `json:"approved_by"`
	Reason     string `json:"reason"`
}

// approveVerification handles POST /api/v1/verifications/{id}/approve
func (h *VerificationHandlers) approveVerification(w http.ResponseWriter, r *http.Request) {
	verificationID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req ApprovalRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if !httputil.RequireNonEmpty(w, req.ApprovedBy, "approved_by") {
		return
	}

	err := h.verifier.ApproveVerification(r.Context(), verificationID, req.ApprovedBy, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to approve verification: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, map[string]string{
		"message": "Verification approved successfully",
		"status":  "approved",
	})
}

// rejectVerification handles POST /api/v1/verifications/{id}/reject
func (h *VerificationHandlers) rejectVerification(w http.ResponseWriter, r *http.Request) {
	verificationID, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req ApprovalRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if !httputil.RequireNonEmpty(w, req.ApprovedBy, "approved_by") {
		return
	}

	if !httputil.RequireNonEmpty(w, req.Reason, "reason") {
		return
	}

	err := h.verifier.RejectVerification(r.Context(), verificationID, req.ApprovedBy, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to reject verification: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, map[string]string{
		"message": "Verification rejected",
		"status":  "rejected",
	})
}

// VerificationStatsResponse contains verification statistics
type VerificationStatsResponse struct {
	TotalVerifications int            `json:"total_verifications"`
	Pending            int            `json:"pending"`
	InProgress         int            `json:"in_progress"`
	Approved           int            `json:"approved"`
	Rejected           int            `json:"rejected"`
	ReviewRequired     int            `json:"review_required"`
	AvgProcessingTime  string         `json:"avg_processing_time"`
	StatusBreakdown    map[string]int `json:"status_breakdown"`
}

// getVerificationStats handles GET /api/v1/verifications/stats
func (h *VerificationHandlers) getVerificationStats(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) as approved,
			SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected,
			SUM(CASE WHEN status = 'review_required' THEN 1 ELSE 0 END) as review_required,
			AVG(TIMESTAMPDIFF(SECOND, started_at, completed_at)) as avg_seconds
		FROM plugin_verifications
		WHERE started_at IS NOT NULL
	`

	var stats VerificationStatsResponse
	var avgSeconds sql.NullFloat64

	err := h.verifier.GetDB().QueryRowContext(r.Context(), query).Scan(
		&stats.TotalVerifications,
		&stats.Pending,
		&stats.InProgress,
		&stats.Approved,
		&stats.Rejected,
		&stats.ReviewRequired,
		&avgSeconds,
	)

	if err != nil {
		h.logger.Errorf("Failed to get verification stats: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	if avgSeconds.Valid {
		stats.AvgProcessingTime = formatDuration(avgSeconds.Float64)
	} else {
		stats.AvgProcessingTime = "N/A"
	}

	stats.StatusBreakdown = map[string]int{
		"pending":         stats.Pending,
		"in_progress":     stats.InProgress,
		"approved":        stats.Approved,
		"rejected":        stats.Rejected,
		"review_required": stats.ReviewRequired,
	}

	httputil.WriteSuccess(w, stats)
}

// SecurityScoreResponse contains plugin security score information
type SecurityScoreResponse struct{
	PluginID            string  `json:"plugin_id"`
	PluginName          string  `json:"plugin_name"`
	SecurityLevel       string  `json:"security_level"`
	TotalVerifications  int     `json:"total_verifications"`
	ApprovedVerifications int   `json:"approved_verifications"`
	RejectedVerifications int   `json:"rejected_verifications"`
	ApprovalRate        float64 `json:"approval_rate"`
	TotalSecurityIssues int     `json:"total_security_issues"`
	CriticalIssues      int     `json:"critical_issues"`
	HighIssues          int     `json:"high_issues"`
	LastVerifiedAt      string  `json:"last_verified_at,omitempty"`
}

// getPluginSecurityScore handles GET /api/v1/plugins/{id}/security-score
func (h *VerificationHandlers) getPluginSecurityScore(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	pluginID := vars["id"]

	query := `SELECT * FROM plugin_security_scores WHERE plugin_id = ?`

	var score SecurityScoreResponse
	var lastVerifiedAt sql.NullString

	err := h.verifier.GetDB().QueryRowContext(r.Context(), query, pluginID).Scan(
		&score.PluginID,
		&score.PluginName,
		&score.SecurityLevel,
		&score.TotalVerifications,
		&score.ApprovedVerifications,
		&score.RejectedVerifications,
		&score.ApprovalRate,
		&score.TotalSecurityIssues,
		&score.CriticalIssues,
		&score.HighIssues,
		&lastVerifiedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			httputil.WriteNotFoundError(w, "Plugin not found")
			return
		}
		h.logger.Errorf("Failed to get security score: %v", err)
		httputil.WriteInternalError(w, err)
		return
	}

	if lastVerifiedAt.Valid {
		score.LastVerifiedAt = lastVerifiedAt.String
	}

	httputil.WriteSuccess(w, score)
}

// Helper methods

func parseIntParam(param string, defaultValue int) int {
	if param == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(param)
	if err != nil {
		return defaultValue
	}
	return value
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	}
	return fmt.Sprintf("%.1fh", seconds/3600)
}
