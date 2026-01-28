// Package middleware provides HTTP middleware for authentication, authorization, and rate limiting.
//
// # Overview
//
// This package implements request processing middleware including token authentication,
// rate limiting (per-user and distributed), quota enforcement, and organization context.
//
// # Middleware Components
//
// AuthMiddleware: Token-based authentication
//
//	router.Use(middleware.AuthMiddleware(tokenManager, optional=false))
//	// Extracts Bearer token, validates, adds AuthContext to request
//
// RateLimitMiddleware: In-memory rate limiting
//
//	limiter := middleware.NewRateLimiter(100, 10) // 100/min, 10 burst
//	router.Use(middleware.RateLimitMiddleware(limiter))
//
// DistributedRateLimitMiddleware: Redis-backed rate limiting
//
//	limiter := middleware.NewDistributedRateLimiter(redisClient)
//	router.Use(middleware.DistributedRateLimitMiddleware(limiter))
//
// QuotaMiddleware: Organization quota enforcement
//
//	router.Use(middleware.QuotaMiddleware(orgService))
//
// OrganizationMiddleware: Extract org from URL
//
//	router.Use(middleware.OrganizationMiddleware(orgService))
//
// # Rate Limiting
//
// Default (Anonymous): 100 req/min, 10 burst
// Per-User: 1000 req/min, 50 burst
// Per-Bot: 5000 req/min, 100 burst
//
// # Related Packages
//
//   - pkg/auth: Token validation
//   - pkg/orgs: Quota checking
//   - pkg/rbac: Permission checking
package middleware
