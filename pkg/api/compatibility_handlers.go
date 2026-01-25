package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/compatibility"
)

// CompatibilityHandlers handles compatibility checking HTTP requests
type CompatibilityHandlers struct {
	storage Storage
}

// NewCompatibilityHandlers creates a new compatibility handlers instance
func NewCompatibilityHandlers(storage Storage) *CompatibilityHandlers {
	return &CompatibilityHandlers{
		storage: storage,
	}
}

// RegisterRoutes registers compatibility routes
func (h *CompatibilityHandlers) RegisterRoutes(router *mux.Router) {
	// Check compatibility between two versions
	router.HandleFunc("/modules/{name}/compatibility", h.checkCompatibility).Methods("POST")

	// Check compatibility for a new version against the latest
	router.HandleFunc("/modules/{name}/versions/{version}/compatibility", h.checkVersionCompatibility).Methods("GET")
}

// checkCompatibility handles POST /modules/{name}/compatibility
func (h *CompatibilityHandlers) checkCompatibility(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]

	var req struct {
		OldVersion string `json:"old_version"`
		NewVersion string `json:"new_version"`
		Mode       string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.OldVersion == "" {
		http.Error(w, "old_version is required", http.StatusBadRequest)
		return
	}
	if req.NewVersion == "" {
		http.Error(w, "new_version is required", http.StatusBadRequest)
		return
	}

	// Parse compatibility mode
	mode := compatibility.CompatibilityModeBackward
	if req.Mode != "" {
		switch req.Mode {
		case "BACKWARD":
			mode = compatibility.CompatibilityModeBackward
		case "FORWARD":
			mode = compatibility.CompatibilityModeForward
		case "FULL":
			mode = compatibility.CompatibilityModeFull
		case "BACKWARD_TRANSITIVE":
			mode = compatibility.CompatibilityModeBackwardTransitive
		case "FORWARD_TRANSITIVE":
			mode = compatibility.CompatibilityModeForwardTransitive
		case "FULL_TRANSITIVE":
			mode = compatibility.CompatibilityModeFullTransitive
		case "NONE":
			mode = compatibility.CompatibilityModeNone
		default:
			http.Error(w, "invalid compatibility mode", http.StatusBadRequest)
			return
		}
	}

	// Get old version
	oldVer, err := h.storage.GetVersion(moduleName, req.OldVersion)
	if err != nil {
		http.Error(w, "old version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Get new version
	newVer, err := h.storage.GetVersion(moduleName, req.NewVersion)
	if err != nil {
		http.Error(w, "new version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Parse old schema
	oldParser := protobuf.NewStringParser(oldVer.Files[0].Content)
	oldAST, err := oldParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse old schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse new schema
	newParser := protobuf.NewStringParser(newVer.Files[0].Content)
	newAST, err := newParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse new schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build schema graphs
	builder := compatibility.NewSchemaGraphBuilder()
	oldSchema, err := builder.BuildFromAST(oldAST)
	if err != nil {
		http.Error(w, "failed to build old schema: "+err.Error(), http.StatusInternalServerError)
		return
	}
	newSchema, err := builder.BuildFromAST(newAST)
	if err != nil {
		http.Error(w, "failed to build new schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Compare schemas
	result, err := compatibility.CheckCompatibility(oldSchema, newSchema, mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result
	response := struct {
		Compatible   bool                        `json:"compatible"`
		Mode         string                      `json:"mode"`
		Violations   []compatibility.Violation   `json:"violations"`
		ErrorCount   int                         `json:"error_count"`
		WarningCount int                         `json:"warning_count"`
		InfoCount    int                         `json:"info_count"`
	}{
		Compatible:   result.Compatible,
		Mode:         result.Mode,
		Violations:   result.Violations,
		ErrorCount:   result.Summary.Errors,
		WarningCount: result.Summary.Warnings,
		InfoCount:    result.Summary.Infos,
	}

	// Set appropriate status code
	if !result.Compatible {
		w.WriteHeader(http.StatusConflict)
	}

	json.NewEncoder(w).Encode(response)
}

// checkVersionCompatibility handles GET /modules/{name}/versions/{version}/compatibility
func (h *CompatibilityHandlers) checkVersionCompatibility(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]

	// Get compatibility mode from query params
	modeStr := r.URL.Query().Get("mode")
	mode := compatibility.CompatibilityModeBackward
	if modeStr != "" {
		switch modeStr {
		case "BACKWARD":
			mode = compatibility.CompatibilityModeBackward
		case "FORWARD":
			mode = compatibility.CompatibilityModeForward
		case "FULL":
			mode = compatibility.CompatibilityModeFull
		case "BACKWARD_TRANSITIVE":
			mode = compatibility.CompatibilityModeBackwardTransitive
		case "FORWARD_TRANSITIVE":
			mode = compatibility.CompatibilityModeForwardTransitive
		case "FULL_TRANSITIVE":
			mode = compatibility.CompatibilityModeFullTransitive
		case "NONE":
			mode = compatibility.CompatibilityModeNone
		default:
			http.Error(w, "invalid compatibility mode", http.StatusBadRequest)
			return
		}
	}

	// Get the version to check
	newVer, err := h.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Get all versions to find the previous one
	versions, err := h.storage.ListVersions(moduleName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(versions) < 2 {
		// No previous version to compare against
		json.NewEncoder(w).Encode(map[string]interface{}{
			"compatible": true,
			"mode":       mode,
			"message":    "No previous version to compare against",
		})
		return
	}

	// Find the version before this one (sorted by creation time)
	var oldVer *Version
	for i, v := range versions {
		if v.Version == version && i+1 < len(versions) {
			oldVer = versions[i+1]
			break
		}
	}

	if oldVer == nil {
		http.Error(w, "could not find previous version", http.StatusNotFound)
		return
	}

	// Parse old schema
	oldParser := protobuf.NewStringParser(oldVer.Files[0].Content)
	oldAST, err := oldParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse old schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse new schema
	newParser := protobuf.NewStringParser(newVer.Files[0].Content)
	newAST, err := newParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse new schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build schema graphs
	builder := compatibility.NewSchemaGraphBuilder()
	oldSchema, err := builder.BuildFromAST(oldAST)
	if err != nil {
		http.Error(w, "failed to build old schema: "+err.Error(), http.StatusInternalServerError)
		return
	}
	newSchema, err := builder.BuildFromAST(newAST)
	if err != nil {
		http.Error(w, "failed to build new schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Compare schemas
	result, err := compatibility.CheckCompatibility(oldSchema, newSchema, mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result
	response := struct {
		Compatible     bool                      `json:"compatible"`
		Mode           string                    `json:"mode"`
		OldVersion     string                    `json:"old_version"`
		NewVersion     string                    `json:"new_version"`
		Violations     []compatibility.Violation `json:"violations"`
		ErrorCount     int                       `json:"error_count"`
		WarningCount   int                       `json:"warning_count"`
		InfoCount      int                       `json:"info_count"`
	}{
		Compatible:   result.Compatible,
		Mode:         result.Mode,
		OldVersion:   oldVer.Version,
		NewVersion:   version,
		Violations:   result.Violations,
		ErrorCount:   result.Summary.Errors,
		WarningCount: result.Summary.Warnings,
		InfoCount:    result.Summary.Infos,
	}

	// Set appropriate status code
	if !result.Compatible {
		w.WriteHeader(http.StatusConflict)
	}

	json.NewEncoder(w).Encode(response)
}
