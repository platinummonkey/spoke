package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// DiffRequest represents a request to compare two versions
type DiffRequest struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

// compareDiff compares two versions and returns the differences
func (s *Server) compareDiff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]

	// Parse request body
	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get both versions
	fromVer, err := s.storage.GetVersion(moduleName, req.FromVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("From version not found: %v", err), http.StatusNotFound)
		return
	}

	toVer, err := s.storage.GetVersion(moduleName, req.ToVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("To version not found: %v", err), http.StatusNotFound)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(placeholder)
}
