package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
	"github.com/platinummonkey/spoke/pkg/httputil"
	"github.com/platinummonkey/spoke/pkg/validation"
)

// ValidationHandlers handles schema validation HTTP requests
type ValidationHandlers struct {
	storage Storage
}

// NewValidationHandlers creates a new validation handlers instance
func NewValidationHandlers(storage Storage) *ValidationHandlers {
	return &ValidationHandlers{
		storage: storage,
	}
}

// RegisterRoutes registers validation routes
func (h *ValidationHandlers) RegisterRoutes(router *mux.Router) {
	// Validate a proto file
	router.HandleFunc("/validate", h.validateProto).Methods("POST")

	// Validate a specific module version
	router.HandleFunc("/modules/{name}/versions/{version}/validate", h.validateVersion).Methods("GET")

	// Normalize a proto file
	router.HandleFunc("/normalize", h.normalizeProto).Methods("POST")
}

// validateProto handles POST /validate
func (h *ValidationHandlers) validateProto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Config  *struct {
			EnforceFieldNumberRanges   bool `json:"enforce_field_number_ranges"`
			RequireEnumZeroValue       bool `json:"require_enum_zero_value"`
			CheckNamingConventions     bool `json:"check_naming_conventions"`
			DetectCircularDependencies bool `json:"detect_circular_dependencies"`
			DetectUnusedImports        bool `json:"detect_unused_imports"`
			CheckReservedFields        bool `json:"check_reserved_fields"`
		} `json:"config"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if !httputil.RequireNonEmpty(w, req.Content, "content") {
		return
	}

	// Parse the proto content
	ast, err := protobuf.ParseString(req.Content)
	if err != nil {
		httputil.WriteBadRequest(w, "failed to parse proto: "+err.Error())
		return
	}

	// Create validator with config
	config := validation.DefaultValidationConfig()
	if req.Config != nil {
		config.EnforceFieldNumberRanges = req.Config.EnforceFieldNumberRanges
		config.RequireEnumZeroValue = req.Config.RequireEnumZeroValue
		config.CheckNamingConventions = req.Config.CheckNamingConventions
		config.DetectCircularDependencies = req.Config.DetectCircularDependencies
		config.DetectUnusedImports = req.Config.DetectUnusedImports
		config.CheckReservedFields = req.Config.CheckReservedFields
	}

	validator := validation.NewValidator(config)
	result := validator.Validate(ast)

	// Return result
	response := struct {
		Valid        bool                          `json:"valid"`
		Errors       []*validation.ValidationError `json:"errors"`
		Warnings     []*validation.ValidationError `json:"warnings"`
		ErrorCount   int                           `json:"error_count"`
		WarningCount int                           `json:"warning_count"`
	}{
		Valid:        result.Valid,
		Errors:       result.Errors,
		Warnings:     result.Warnings,
		ErrorCount:   len(result.Errors),
		WarningCount: len(result.Warnings),
	}

	// Set appropriate status code and return result
	if !result.Valid {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, response)
		return
	}

	httputil.WriteSuccess(w, response)
}

// validateVersion handles GET /modules/{name}/versions/{version}/validate
func (h *ValidationHandlers) validateVersion(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	moduleName := vars["name"]
	version := vars["version"]

	// Get the version
	ver, err := h.storage.GetVersion(moduleName, version)
	if err != nil {
		httputil.WriteNotFoundError(w, "version not found: "+err.Error())
		return
	}

	if len(ver.Files) == 0 {
		httputil.WriteNotFoundError(w, "no proto files in version")
		return
	}

	// Parse the proto content
	ast, err := protobuf.ParseString(ver.Files[0].Content)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	// Validate with default config
	validator := validation.NewValidator(validation.DefaultValidationConfig())
	result := validator.Validate(ast)

	// Return result
	response := struct {
		Valid        bool                          `json:"valid"`
		ModuleName   string                        `json:"module_name"`
		Version      string                        `json:"version"`
		Errors       []*validation.ValidationError `json:"errors"`
		Warnings     []*validation.ValidationError `json:"warnings"`
		ErrorCount   int                           `json:"error_count"`
		WarningCount int                           `json:"warning_count"`
	}{
		Valid:        result.Valid,
		ModuleName:   moduleName,
		Version:      version,
		Errors:       result.Errors,
		Warnings:     result.Warnings,
		ErrorCount:   len(result.Errors),
		WarningCount: len(result.Warnings),
	}

	// Set appropriate status code and return result
	if !result.Valid {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, response)
		return
	}

	httputil.WriteSuccess(w, response)
}

// normalizeProto handles POST /normalize
func (h *ValidationHandlers) normalizeProto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Config  *struct {
			SortFields               bool `json:"sort_fields"`
			SortEnumValues           bool `json:"sort_enum_values"`
			SortImports              bool `json:"sort_imports"`
			CanonicalizeImports      bool `json:"canonicalize_imports"`
			PreserveComments         bool `json:"preserve_comments"`
			StandardizeWhitespace    bool `json:"standardize_whitespace"`
			RemoveTrailingWhitespace bool `json:"remove_trailing_whitespace"`
		} `json:"config"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if !httputil.RequireNonEmpty(w, req.Content, "content") {
		return
	}

	// Create normalizer with config
	config := validation.DefaultNormalizationConfig()
	if req.Config != nil {
		config.SortFields = req.Config.SortFields
		config.SortEnumValues = req.Config.SortEnumValues
		config.SortImports = req.Config.SortImports
		config.CanonicalizeImports = req.Config.CanonicalizeImports
		config.PreserveComments = req.Config.PreserveComments
		config.StandardizeWhitespace = req.Config.StandardizeWhitespace
		config.RemoveTrailingWhitespace = req.Config.RemoveTrailingWhitespace
	}

	normalizer := validation.NewNormalizer(config)
	normalized, err := normalizer.NormalizeString(req.Content)
	if err != nil {
		httputil.WriteBadRequest(w, "failed to normalize: "+err.Error())
		return
	}

	// Return normalized content
	response := struct {
		Normalized string `json:"normalized"`
		Original   string `json:"original,omitempty"`
	}{
		Normalized: normalized,
	}

	httputil.WriteSuccess(w, response)
}
