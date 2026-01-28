// Package httputil provides HTTP utilities for standardized request/response handling.
//
// # Overview
//
// This package offers helper functions for JSON encoding/decoding, error responses,
// parameter parsing, validation, and common HTTP middleware patterns.
//
// # Response Helpers
//
// JSON responses:
//
//	httputil.WriteJSON(w, http.StatusOK, data)
//	httputil.WriteSuccess(w, "Operation completed")
//	httputil.WriteCreated(w, resource)
//
// Error responses:
//
//	httputil.WriteError(w, http.StatusBadRequest, err)
//	httputil.WriteBadRequest(w, "Invalid input")
//	httputil.WriteUnauthorized(w, "Token expired")
//	httputil.WriteForbidden(w, "Insufficient permissions")
//
// # Request Parsing
//
// JSON parsing:
//
//	var req CreateModuleRequest
//	if !httputil.ParseJSONOrError(w, r, &req) {
//		return // Error response already written
//	}
//
// Path parameters:
//
//	id := httputil.ParsePathInt64(mux.Vars(r), "id")
//	name := httputil.ParsePathString(mux.Vars(r), "name")
//
// Query parameters:
//
//	limit := httputil.ParseQueryInt(r, "limit", 20)
//	offset := httputil.ParseQueryInt(r, "offset", 0)
//	recursive := httputil.ParseQueryBool(r, "recursive", false)
//
// # Validation
//
//	httputil.ValidateAll(w,
//		httputil.RequireNonEmpty("name", req.Name),
//		httputil.RequirePositive("limit", limit),
//	)
//
// # Middleware
//
//	httputil.Chain(
//		httputil.LoggingMiddleware(),
//		httputil.RecoveryMiddleware(),
//		httputil.RequestIDMiddleware(),
//		httputil.TimeoutMiddleware(30*time.Second),
//		httputil.MaxBytesMiddleware(10*1024*1024), // 10MB
//	)
//
// # Related Packages
//
//   - pkg/middleware: Authentication and authorization middleware
package httputil
