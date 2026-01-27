package api

import (
	"fmt"
	"net/http"

	"github.com/platinummonkey/spoke/pkg/httputil"
)

// DiffRequest represents a request to compare two versions
type DiffRequest struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

// compareDiff compares two versions and returns the differences
func (s *Server) compareDiff(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	moduleName := vars["name"]

	// Parse request body
	var req DiffRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Get both versions
	fromVer, err := s.storage.GetVersion(moduleName, req.FromVersion)
	if err != nil {
		httputil.WriteNotFoundError(w, fmt.Sprintf("From version not found: %v", err))
		return
	}

	toVer, err := s.storage.GetVersion(moduleName, req.ToVersion)
	if err != nil {
		httputil.WriteNotFoundError(w, fmt.Sprintf("To version not found: %v", err))
		return
	}

	// TODO: Implement actual diff analysis
	// For now, return a placeholder
	placeholder := map[string]interface{}{
		"from_version": req.FromVersion,
		"to_version":   req.ToVersion,
		"changes": []map[string]interface{}{
			{
				"type":        "placeholder",
				"severity":    "non_breaking",
				"location":    "example.proto",
				"description": "Schema diff analysis coming soon!",
			},
		},
		"_note": fmt.Sprintf("Comparing %d files in v%s with %d files in v%s",
			len(fromVer.Files), req.FromVersion, len(toVer.Files), req.ToVersion),
	}

	httputil.WriteSuccess(w, placeholder)
}
