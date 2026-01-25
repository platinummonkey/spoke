package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Alerter monitors metrics and triggers alerts
type Alerter struct {
	db *sql.DB
}

// NewAlerter creates a new Alerter instance
func NewAlerter(db *sql.DB) *Alerter {
	return &Alerter{db: db}
}

// Alert represents an alert notification
type Alert struct {
	Type        string    // "health", "performance", "usage"
	Severity    string    // "critical", "warning", "info"
	Title       string
	Message     string
	Details     map[string]interface{}
	TriggeredAt time.Time
}

// HealthAlert represents a schema health alert
type HealthAlert struct {
	ModuleName  string
	Version     string
	HealthScore float64
	Issues      []string
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	Language       string
	P95DurationMs  int
	ThresholdMs    int
	CompilationCount int
}

// UsageAlert represents a usage alert
type UsageAlert struct {
	ModuleName      string
	UnusedFields    int
	LastAccessDays  int
}

// CheckHealthAlerts checks for schemas with low health scores
func (a *Alerter) CheckHealthAlerts(ctx context.Context, threshold float64) ([]HealthAlert, error) {
	query := `
		SELECT
			m.name AS module_name,
			v.version,
			-- Calculate simplified health score from existing data
			GREATEST(0, 100 - (
				-- Penalty for old modules (no recent downloads)
				CASE
					WHEN MAX(de.downloaded_at) < NOW() - INTERVAL '90 days' THEN 30
					WHEN MAX(de.downloaded_at) < NOW() - INTERVAL '30 days' THEN 15
					ELSE 0
				END +
				-- Penalty for compilation failures
				CASE
					WHEN COUNT(ce.*) > 0 THEN
						(COUNT(*) FILTER (WHERE ce.success = false)::float / COUNT(ce.*) * 20)
					ELSE 0
				END
			)) AS health_score
		FROM modules m
		JOIN versions v ON m.id = v.module_id
		LEFT JOIN download_events de ON m.name = de.module_name AND v.version = de.version
			AND de.downloaded_at >= NOW() - INTERVAL '90 days'
		LEFT JOIN compilation_events ce ON m.name = ce.module_name AND v.version = ce.version
			AND ce.started_at >= NOW() - INTERVAL '30 days'
		GROUP BY m.name, v.version
		HAVING GREATEST(0, 100 - (
			CASE
				WHEN MAX(de.downloaded_at) < NOW() - INTERVAL '90 days' THEN 30
				WHEN MAX(de.downloaded_at) < NOW() - INTERVAL '30 days' THEN 15
				ELSE 0
			END +
			CASE
				WHEN COUNT(ce.*) > 0 THEN
					(COUNT(*) FILTER (WHERE ce.success = false)::float / COUNT(ce.*) * 20)
				ELSE 0
			END
		)) < $1
		ORDER BY health_score ASC
		LIMIT 20
	`

	rows, err := a.db.QueryContext(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query health alerts: %w", err)
	}
	defer rows.Close()

	var alerts []HealthAlert
	for rows.Next() {
		var alert HealthAlert
		if err := rows.Scan(&alert.ModuleName, &alert.Version, &alert.HealthScore); err != nil {
			return nil, fmt.Errorf("failed to scan health alert: %w", err)
		}

		// Determine issues based on health score
		alert.Issues = []string{}
		if alert.HealthScore < 40 {
			alert.Issues = append(alert.Issues, "Critical: Immediate attention required")
		}
		if alert.HealthScore < 60 {
			alert.Issues = append(alert.Issues, "Low usage or high failure rate detected")
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating health alerts: %w", err)
	}

	return alerts, nil
}

// CheckPerformanceAlerts checks for slow compilations
func (a *Alerter) CheckPerformanceAlerts(ctx context.Context, thresholdMs int) ([]PerformanceAlert, error) {
	query := `
		SELECT
			language,
			p95_duration_ms,
			compilation_count
		FROM language_stats_daily
		WHERE date >= CURRENT_DATE - INTERVAL '7 days'
		  AND p95_duration_ms > $1
		GROUP BY language, p95_duration_ms, compilation_count
		ORDER BY p95_duration_ms DESC
		LIMIT 10
	`

	rows, err := a.db.QueryContext(ctx, query, thresholdMs)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance alerts: %w", err)
	}
	defer rows.Close()

	var alerts []PerformanceAlert
	for rows.Next() {
		var alert PerformanceAlert
		alert.ThresholdMs = thresholdMs
		if err := rows.Scan(&alert.Language, &alert.P95DurationMs, &alert.CompilationCount); err != nil {
			return nil, fmt.Errorf("failed to scan performance alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating performance alerts: %w", err)
	}

	return alerts, nil
}

// CheckUsageAlerts checks for unused modules and fields
func (a *Alerter) CheckUsageAlerts(ctx context.Context, inactiveDays int) ([]UsageAlert, error) {
	query := `
		SELECT
			m.name AS module_name,
			0 AS unused_fields,
			COALESCE(DATE_PART('day', NOW() - MAX(de.downloaded_at)), 999) AS last_access_days
		FROM modules m
		LEFT JOIN download_events de ON m.name = de.module_name
		GROUP BY m.name
		HAVING COALESCE(DATE_PART('day', NOW() - MAX(de.downloaded_at)), 999) > $1
		ORDER BY last_access_days DESC
		LIMIT 20
	`

	rows, err := a.db.QueryContext(ctx, query, inactiveDays)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage alerts: %w", err)
	}
	defer rows.Close()

	var alerts []UsageAlert
	for rows.Next() {
		var alert UsageAlert
		var lastAccessDays float64
		if err := rows.Scan(&alert.ModuleName, &alert.UnusedFields, &lastAccessDays); err != nil {
			return nil, fmt.Errorf("failed to scan usage alert: %w", err)
		}
		alert.LastAccessDays = int(lastAccessDays)
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating usage alerts: %w", err)
	}

	return alerts, nil
}

// CheckAllAlerts runs all alert checks and logs results
func (a *Alerter) CheckAllAlerts(ctx context.Context) error {
	log.Println("Running analytics alert checks...")

	// Check health alerts (threshold: 50)
	healthAlerts, err := a.CheckHealthAlerts(ctx, 50.0)
	if err != nil {
		log.Printf("ERROR: Failed to check health alerts: %v", err)
	} else if len(healthAlerts) > 0 {
		log.Printf("ALERT: Found %d modules with low health scores:", len(healthAlerts))
		for _, alert := range healthAlerts {
			log.Printf("  - %s@%s: Health Score %.1f - %v",
				alert.ModuleName, alert.Version, alert.HealthScore, alert.Issues)
		}
	} else {
		log.Println("INFO: No health alerts")
	}

	// Check performance alerts (threshold: 5000ms = 5 seconds)
	perfAlerts, err := a.CheckPerformanceAlerts(ctx, 5000)
	if err != nil {
		log.Printf("ERROR: Failed to check performance alerts: %v", err)
	} else if len(perfAlerts) > 0 {
		log.Printf("ALERT: Found %d languages with slow compilation performance:", len(perfAlerts))
		for _, alert := range perfAlerts {
			log.Printf("  - %s: p95=%dms (threshold=%dms, compilations=%d)",
				alert.Language, alert.P95DurationMs, alert.ThresholdMs, alert.CompilationCount)
		}
	} else {
		log.Println("INFO: No performance alerts")
	}

	// Check usage alerts (inactive for 90+ days)
	usageAlerts, err := a.CheckUsageAlerts(ctx, 90)
	if err != nil {
		log.Printf("ERROR: Failed to check usage alerts: %v", err)
	} else if len(usageAlerts) > 0 {
		log.Printf("ALERT: Found %d modules with no recent usage:", len(usageAlerts))
		for _, alert := range usageAlerts {
			log.Printf("  - %s: No downloads in %d days",
				alert.ModuleName, alert.LastAccessDays)
		}
	} else {
		log.Println("INFO: No usage alerts")
	}

	log.Println("Analytics alert checks completed")
	return nil
}

// SendAlert would send alerts to external systems (email, Slack, etc.)
// This is a placeholder for integration with notification systems
func (a *Alerter) SendAlert(alert Alert) error {
	// TODO: Integrate with notification system
	// Examples:
	// - Send email via SMTP
	// - Post to Slack webhook
	// - Create PagerDuty incident
	// - Write to monitoring system

	log.Printf("[%s] %s: %s - %s\n",
		alert.Severity, alert.Type, alert.Title, alert.Message)

	return nil
}
