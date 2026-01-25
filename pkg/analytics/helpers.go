package analytics

import (
	"net/http"
	"strings"

	"github.com/platinummonkey/spoke/pkg/middleware"
)

// ExtractUserID extracts user ID from request context
func ExtractUserID(r *http.Request) *int64 {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil || authCtx.User == nil {
		return nil
	}
	userID := authCtx.User.ID
	return &userID
}

// ExtractOrganizationID extracts organization ID from request context
func ExtractOrganizationID(r *http.Request) *int64 {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil || authCtx.Organization == nil {
		return nil
	}
	orgID := authCtx.Organization.ID
	return &orgID
}

// GetClientIP extracts client IP address from request
func GetClientIP(r *http.Request) string {
	// Try X-Forwarded-For header first (proxy/load balancer)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Try X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// RemoteAddr includes port, strip it
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// GetUserAgent extracts user agent from request
func GetUserAgent(r *http.Request) string {
	return r.UserAgent()
}

// GetClientSDK extracts SDK name from User-Agent
func GetClientSDK(r *http.Request) string {
	ua := r.UserAgent()
	// Parse User-Agent for SDK identifier
	// Format: "spoke-cli/1.0.0" or "spoke-python-sdk/2.1.0"
	if strings.HasPrefix(ua, "spoke-") {
		parts := strings.Split(ua, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// GetClientVersion extracts SDK version from User-Agent
func GetClientVersion(r *http.Request) string {
	ua := r.UserAgent()
	// Parse User-Agent for version
	// Format: "spoke-cli/1.0.0" or "spoke-python-sdk/2.1.0"
	if strings.HasPrefix(ua, "spoke-") {
		parts := strings.Split(ua, "/")
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return ""
}

// GetReferrer extracts referrer from request
func GetReferrer(r *http.Request) string {
	return r.Header.Get("Referer")
}
