package audit

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for audit log API
type Handlers struct {
	store Store
}

// NewHandlers creates new audit handlers
func NewHandlers(store Store) *Handlers {
	return &Handlers{
		store: store,
	}
}

// RegisterRoutes registers audit log routes
func (h *Handlers) RegisterRoutes(router *mux.Router) {
	// Audit log routes
	router.HandleFunc("/audit/events", h.listEvents).Methods("GET")
	router.HandleFunc("/audit/events/{id}", h.getEvent).Methods("GET")
	router.HandleFunc("/audit/export", h.exportEvents).Methods("GET")
	router.HandleFunc("/audit/stats", h.getStats).Methods("GET")
}

// listEvents handles GET /audit/events
func (h *Handlers) listEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := h.parseFilter(r)

	// Search events
	events, err := h.store.Search(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// getEvent handles GET /audit/events/{id}
func (h *Handlers) getEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid event ID", http.StatusBadRequest)
		return
	}

	event, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// exportEvents handles GET /audit/export
func (h *Handlers) exportEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := h.parseFilter(r)

	// Get export format
	formatStr := r.URL.Query().Get("format")
	format := ExportFormat(formatStr)
	if format == "" {
		format = ExportFormatJSON
	}

	// Export events
	data, err := h.store.Export(r.Context(), filter, format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set appropriate content type and headers
	switch format {
	case ExportFormatCSV:
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.csv")
	case ExportFormatNDJSON:
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.ndjson")
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.json")
	}

	w.Write(data)
}

// getStats handles GET /audit/stats
func (h *Handlers) getStats(w http.ResponseWriter, r *http.Request) {
	// Parse time range from query parameters
	var startTime, endTime *time.Time

	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = &t
		}
	}

	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = &t
		}
	}

	// Get statistics
	stats, err := h.store.GetStats(r.Context(), startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// parseFilter parses search filter from query parameters
func (h *Handlers) parseFilter(r *http.Request) SearchFilter {
	query := r.URL.Query()
	filter := SearchFilter{}

	// Parse time range
	if startStr := query.Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}

	if endStr := query.Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	// Parse actor filters
	if userIDStr := query.Get("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			filter.UserID = &userID
		}
	}

	filter.Username = query.Get("username")

	if orgIDStr := query.Get("organization_id"); orgIDStr != "" {
		if orgID, err := strconv.ParseInt(orgIDStr, 10, 64); err == nil {
			filter.OrganizationID = &orgID
		}
	}

	// Parse event filters
	if eventTypesStr := query.Get("event_types"); eventTypesStr != "" {
		// Parse comma-separated event types
		eventTypeStrs := parseCommaSeparated(eventTypesStr)
		for _, etStr := range eventTypeStrs {
			filter.EventTypes = append(filter.EventTypes, EventType(etStr))
		}
	}

	if statusStr := query.Get("status"); statusStr != "" {
		status := EventStatus(statusStr)
		filter.Status = &status
	}

	// Parse resource filters
	filter.ResourceType = ResourceType(query.Get("resource_type"))
	filter.ResourceID = query.Get("resource_id")
	filter.ResourceName = query.Get("resource_name")

	// Parse request context filters
	filter.IPAddress = query.Get("ip_address")
	filter.Method = query.Get("method")
	filter.Path = query.Get("path")

	// Parse pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 100 // Default limit
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// Parse sorting
	filter.SortBy = query.Get("sort_by")
	filter.SortOrder = query.Get("sort_order")
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	return filter
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	for i := 0; i < len(s); {
		// Find next comma
		end := i
		for end < len(s) && s[end] != ',' {
			end++
		}

		// Extract and trim the value
		val := s[i:end]
		// Simple trim of leading/trailing spaces
		for len(val) > 0 && val[0] == ' ' {
			val = val[1:]
		}
		for len(val) > 0 && val[len(val)-1] == ' ' {
			val = val[:len(val)-1]
		}

		if val != "" {
			result = append(result, val)
		}

		i = end + 1
	}

	return result
}
