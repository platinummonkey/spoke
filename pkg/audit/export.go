package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
)

// exportJSON exports audit events as JSON array
func exportJSON(events []*AuditEvent) ([]byte, error) {
	return json.MarshalIndent(events, "", "  ")
}

// exportNDJSON exports audit events as newline-delimited JSON
func exportNDJSON(events []*AuditEvent) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return nil, fmt.Errorf("failed to encode event: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// exportCSV exports audit events as CSV
func exportCSV(events []*AuditEvent) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Timestamp",
		"EventType",
		"Status",
		"UserID",
		"Username",
		"OrganizationID",
		"TokenID",
		"ResourceType",
		"ResourceID",
		"ResourceName",
		"IPAddress",
		"UserAgent",
		"RequestID",
		"Method",
		"Path",
		"StatusCode",
		"Message",
		"ErrorMessage",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write events
	for _, event := range events {
		row := []string{
			strconv.FormatInt(event.ID, 10),
			event.Timestamp.Format("2006-01-02 15:04:05"),
			string(event.EventType),
			string(event.Status),
			formatInt64Ptr(event.UserID),
			event.Username,
			formatInt64Ptr(event.OrganizationID),
			formatInt64Ptr(event.TokenID),
			string(event.ResourceType),
			event.ResourceID,
			event.ResourceName,
			event.IPAddress,
			event.UserAgent,
			event.RequestID,
			event.Method,
			event.Path,
			strconv.Itoa(event.StatusCode),
			event.Message,
			event.ErrorMessage,
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// formatInt64Ptr formats an int64 pointer as string, returning empty string for nil
func formatInt64Ptr(val *int64) string {
	if val == nil {
		return ""
	}
	return strconv.FormatInt(*val, 10)
}
