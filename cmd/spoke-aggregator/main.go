package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/robfig/cron/v3"
)

var (
	dbURL              = flag.String("db-url", getEnv("DATABASE_URL", "postgres://localhost/spoke?sslmode=disable"), "PostgreSQL connection URL")
	dailySchedule      = flag.String("daily-schedule", "5 0 * * *", "Cron schedule for daily aggregation (default: 00:05 UTC)")
	weeklySchedule     = flag.String("weekly-schedule", "10 0 * * 0", "Cron schedule for weekly aggregation (default: Sunday 00:10 UTC)")
	monthlySchedule    = flag.String("monthly-schedule", "15 0 1 * *", "Cron schedule for monthly aggregation (default: 1st day 00:15 UTC)")
	refreshSchedule    = flag.String("refresh-schedule", "0 * * * *", "Cron schedule for materialized view refresh (default: every hour)")
	alertSchedule      = flag.String("alert-schedule", "0 */6 * * *", "Cron schedule for alert checks (default: every 6 hours)")
	runOnce            = flag.Bool("run-once", false, "Run aggregation once and exit (for testing)")
	aggregationDate    = flag.String("date", "", "Date to aggregate (YYYY-MM-DD format). If empty, aggregates yesterday. Only used with --run-once")
)

func main() {
	flag.Parse()

	// Connect to database
	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	aggregator := analytics.NewAggregator(db)
	alerter := analytics.NewAlerter(db)

	// Run once mode (for testing or backfilling)
	if *runOnce {
		var date time.Time
		if *aggregationDate != "" {
			date, err = time.Parse("2006-01-02", *aggregationDate)
			if err != nil {
				log.Fatalf("Invalid date format: %v", err)
			}
		} else {
			// Default to yesterday
			date = time.Now().UTC().AddDate(0, 0, -1)
		}

		log.Printf("Running aggregation for date: %s", date.Format("2006-01-02"))
		if err := runAggregation(aggregator, date); err != nil {
			log.Fatalf("Aggregation failed: %v", err)
		}

		log.Println("Aggregation completed successfully")
		return
	}

	// Scheduled mode
	c := cron.New()

	// Daily aggregation job (aggregates yesterday's data at 00:05 UTC)
	_, err = c.AddFunc(*dailySchedule, func() {
		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		log.Printf("Starting daily aggregation for %s", yesterday.Format("2006-01-02"))

		if err := runAggregation(aggregator, yesterday); err != nil {
			log.Printf("Daily aggregation failed: %v", err)
		} else {
			log.Println("Daily aggregation completed successfully")
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule daily aggregation: %v", err)
	}

	// Refresh materialized views (every hour)
	_, err = c.AddFunc(*refreshSchedule, func() {
		log.Println("Refreshing materialized views")

		ctx := context.Background()
		if err := aggregator.RefreshMaterializedViews(ctx); err != nil {
			log.Printf("Failed to refresh materialized views: %v", err)
		} else {
			log.Println("Materialized views refreshed successfully")
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule materialized view refresh: %v", err)
	}

	// Analytics alert checks (every 6 hours)
	_, err = c.AddFunc(*alertSchedule, func() {
		log.Println("Running analytics alert checks")

		ctx := context.Background()
		if err := alerter.CheckAllAlerts(ctx); err != nil {
			log.Printf("Alert checks failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule alert checks: %v", err)
	}

	// Start the cron scheduler
	c.Start()
	log.Println("Spoke Analytics Aggregator started")
	log.Printf("Daily aggregation schedule: %s", *dailySchedule)
	log.Printf("Materialized view refresh schedule: %s", *refreshSchedule)
	log.Printf("Alert check schedule: %s", *alertSchedule)

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down gracefully...")

	// Stop the cron scheduler
	ctx := c.Stop()
	<-ctx.Done()

	log.Println("Aggregator stopped")
}

func runAggregation(aggregator *analytics.Aggregator, date time.Time) error {
	ctx := context.Background()

	// Run module stats aggregation
	if err := aggregator.AggregateModuleStatsDaily(ctx, date); err != nil {
		log.Printf("Module stats aggregation failed: %v", err)
		return err
	}
	log.Println("✓ Module stats aggregated")

	// Run language stats aggregation
	if err := aggregator.AggregateLanguageStatsDaily(ctx, date); err != nil {
		log.Printf("Language stats aggregation failed: %v", err)
		return err
	}
	log.Println("✓ Language stats aggregated")

	// Run organization stats aggregation
	if err := aggregator.AggregateOrgStatsDaily(ctx, date); err != nil {
		log.Printf("Organization stats aggregation failed: %v", err)
		return err
	}
	log.Println("✓ Organization stats aggregated")

	// Aggregate weekly (if it's Sunday)
	if date.Weekday() == time.Sunday {
		weekStart := date.AddDate(0, 0, -6)
		if err := aggregator.AggregateModuleStatsWeekly(ctx, weekStart); err != nil {
			log.Printf("Weekly stats aggregation failed: %v", err)
			return err
		}
		log.Println("✓ Weekly stats aggregated")
	}

	// Aggregate monthly (if it's the first day of month)
	if date.Day() == 1 {
		month := time.Date(date.Year(), date.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		if err := aggregator.AggregateModuleStatsMonthly(ctx, month); err != nil {
			log.Printf("Monthly stats aggregation failed: %v", err)
			return err
		}
		log.Println("✓ Monthly stats aggregated")
	}

	// Refresh materialized views
	if err := aggregator.RefreshMaterializedViews(ctx); err != nil {
		log.Printf("Materialized view refresh failed: %v", err)
		return err
	}
	log.Println("✓ Materialized views refreshed")

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
