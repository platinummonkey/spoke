package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/billing"
	"github.com/platinummonkey/spoke/pkg/middleware"
	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/platinummonkey/spoke/pkg/storage"
	"github.com/platinummonkey/spoke/pkg/storage/postgres"
)

// Example server with multi-tenancy and billing enabled
func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to listen on")
	postgresURL := flag.String("postgres-url", "", "PostgreSQL connection URL")
	stripeAPIKey := flag.String("stripe-api-key", "", "Stripe API key")
	stripeWebhookSecret := flag.String("stripe-webhook-secret", "", "Stripe webhook secret")
	flag.Parse()

	// Get from environment if not provided
	if *postgresURL == "" {
		*postgresURL = os.Getenv("DATABASE_URL")
	}
	if *stripeAPIKey == "" {
		*stripeAPIKey = os.Getenv("STRIPE_API_KEY")
	}
	if *stripeWebhookSecret == "" {
		*stripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	}

	// Validate required configuration
	if *postgresURL == "" {
		log.Fatal("PostgreSQL URL is required (--postgres-url or DATABASE_URL)")
	}
	if *stripeAPIKey == "" {
		log.Println("Warning: Stripe API key not provided, billing features disabled")
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", *postgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize storage (using postgres storage)
	config := storage.DefaultConfig()
	config.Type = "postgres"
	config.PostgresURL = *postgresURL

	store, err := postgres.NewPostgresStorage(config)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Println("Storage initialized")

	// Initialize organization service
	orgService := orgs.NewPostgresService(db)
	log.Println("Organization service initialized")

	// Initialize billing service
	var billingService billing.Service
	if *stripeAPIKey != "" {
		billingService = billing.NewPostgresService(db, *stripeAPIKey, *stripeWebhookSecret, orgService)
		log.Println("Billing service initialized")
	}

	// Create router
	router := mux.NewRouter()

	// Create API server
	server := api.NewServer(store, db)

	// Register organization handlers
	orgHandlers := api.NewOrgHandlers(orgService)
	orgHandlers.RegisterRoutes(router)
	log.Println("Organization routes registered")

	// Register billing handlers if available
	if billingService != nil {
		billingHandlers := api.NewBillingHandlers(billingService)
		billingHandlers.RegisterRoutes(router)
		log.Println("Billing routes registered")
	}

	// Register quota middleware
	quotaMiddleware := middleware.NewQuotaMiddleware(orgService)

	// Apply org context middleware globally
	router.Use(quotaMiddleware.OrgContextMiddleware)

	// Apply rate limit middleware to all API routes
	router.Use(quotaMiddleware.CheckAPIRateLimit)

	// Apply quota enforcement to specific routes
	// These would be applied to the specific handlers that need them
	// Example: router.Handle("/modules", quotaMiddleware.EnforceModuleQuota(createModuleHandler))

	// Note: Core Spoke API routes would be registered here via the server.Handler() interface

	// Start server
	log.Printf("Starting multi-tenant Spoke server on port %s...", *port)
	log.Println("Features enabled:")
	log.Println("  - Organization management")
	log.Println("  - Quota enforcement")
	log.Println("  - Usage tracking")
	if billingService != nil {
		log.Println("  - Billing and subscriptions")
		log.Println("  - Stripe integration")
	}

	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
