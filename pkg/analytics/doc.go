// Package analytics provides usage analytics and insights for the Spoke registry.
//
// # Overview
//
// This package tracks downloads, views, compilations, and user activity with pre-aggregated
// statistics at multiple time scales (daily, weekly, monthly) for dashboard KPIs and trending.
//
// # Key Metrics
//
// Overview KPIs:
//   - Total modules and versions
//   - Downloads (24h, 7d, 30d)
//   - Active users (24h, 7d)
//   - Top language
//   - Average compilation time
//   - Cache hit rate
//
// Per-Module Analytics:
//   - Total views and downloads
//   - Unique users
//   - Downloads by day/language
//   - Popular versions
//   - Compilation success rate
//
// # Usage Example
//
// Track event:
//
//	tracker.RecordDownload(ctx, &analytics.DownloadEvent{
//		ModuleName: "user-service",
//		Version:    "v1.0.0",
//		Language:   "go",
//		UserID:     user.ID,
//	})
//
// Get module analytics:
//
//	stats, err := service.GetModuleStats(ctx, "user-service")
//	fmt.Printf("Downloads: %d, Views: %d, Users: %d\n",
//		stats.TotalDownloads, stats.TotalViews, stats.UniqueUsers)
//
// Find trending modules:
//
//	trending, err := service.GetTrending(ctx, 30) // Last 30 days
//	for _, module := range trending {
//		fmt.Printf("%s: %.1f%% growth\n", module.Name, module.GrowthRate*100)
//	}
//
// # Aggregation
//
// Batch aggregation runs daily to compute statistics:
//
//	aggregator.RunDaily(ctx)  // Computes module_stats_daily
//	aggregator.RunWeekly(ctx) // Computes module_stats_weekly
//	aggregator.RunMonthly(ctx) // Computes module_stats_monthly
//
// # Related Packages
//
//   - pkg/observability: Metrics and monitoring
package analytics
