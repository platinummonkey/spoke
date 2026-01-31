package swagger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSwaggerHandlers(t *testing.T) {
	handlers := NewSwaggerHandlers()
	assert.NotNil(t, handlers)
	assert.IsType(t, &SwaggerHandlers{}, handlers)
}

func TestRegisterRoutes(t *testing.T) {
	router := mux.NewRouter()
	handlers := NewSwaggerHandlers()

	handlers.RegisterRoutes(router)

	// Test that routes are registered by making requests
	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
	}{
		{
			name:           "OpenAPI YAML endpoint",
			path:           "/openapi.yaml",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OpenAPI JSON endpoint",
			path:           "/openapi.json",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Swagger UI endpoint",
			path:           "/swagger-ui",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API docs alias endpoint",
			path:           "/api-docs",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestServeOpenAPISpec(t *testing.T) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	handlers.serveOpenAPISpec(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, w.Body.Bytes())
	assert.Equal(t, openapiSpec, w.Body.Bytes())
}

func TestServeOpenAPISpecJSON(t *testing.T) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()

	handlers.serveOpenAPISpecJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// Verify valid JSON response
	body := w.Body.String()
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "openapi")
	assert.Contains(t, body, "paths")
}

func TestServeSwaggerUI(t *testing.T) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/swagger-ui", nil)
	w := httptest.NewRecorder()

	handlers.serveSwaggerUI(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "Spoke API - Swagger UI")
	assert.Contains(t, body, "swagger-ui-dist")
	assert.Contains(t, body, "swagger-ui")
	assert.Contains(t, body, "/openapi.yaml")
	assert.Contains(t, body, "SwaggerUIBundle")
	assert.Contains(t, body, "spoke_api_token")
}

func TestSwaggerUITemplate(t *testing.T) {
	// Verify the template contains expected elements
	assert.Contains(t, swaggerUITemplate, "<!DOCTYPE html>")
	assert.Contains(t, swaggerUITemplate, "Spoke API - Swagger UI")
	assert.Contains(t, swaggerUITemplate, "swagger-ui-dist")
	assert.Contains(t, swaggerUITemplate, "/openapi.yaml")
	assert.Contains(t, swaggerUITemplate, "SwaggerUIBundle")
	assert.Contains(t, swaggerUITemplate, "localStorage.getItem('spoke_api_token')")
}

func TestRouteIntegration(t *testing.T) {
	// Test full integration with router
	router := mux.NewRouter()
	handlers := NewSwaggerHandlers()
	handlers.RegisterRoutes(router)

	tests := []struct {
		name               string
		path               string
		expectedStatus     int
		expectedContentType string
		expectedBodyContains []string
	}{
		{
			name:               "YAML spec returns correct content",
			path:               "/openapi.yaml",
			expectedStatus:     http.StatusOK,
			expectedContentType: "application/x-yaml",
			expectedBodyContains: nil, // Just check it returns data
		},
		{
			name:               "JSON spec returns valid JSON",
			path:               "/openapi.json",
			expectedStatus:     http.StatusOK,
			expectedContentType: "application/json",
			expectedBodyContains: []string{"openapi", "paths"},
		},
		{
			name:               "Swagger UI returns HTML",
			path:               "/swagger-ui",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html; charset=utf-8",
			expectedBodyContains: []string{"<!DOCTYPE html>", "swagger-ui"},
		},
		{
			name:               "API docs alias works",
			path:               "/api-docs",
			expectedStatus:     http.StatusOK,
			expectedContentType: "text/html; charset=utf-8",
			expectedBodyContains: []string{"<!DOCTYPE html>", "swagger-ui"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedContentType, w.Header().Get("Content-Type"))

			if tt.expectedBodyContains != nil {
				body := w.Body.String()
				for _, expected := range tt.expectedBodyContains {
					assert.Contains(t, body, expected)
				}
			}
		})
	}
}

func TestOpenAPISpecNotEmpty(t *testing.T) {
	// Verify that the embedded openapi.yaml is not empty
	assert.NotEmpty(t, openapiSpec, "OpenAPI spec should not be empty")
}

func TestCORSHeaders(t *testing.T) {
	handlers := NewSwaggerHandlers()

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name:    "YAML spec has CORS headers",
			handler: handlers.serveOpenAPISpec,
		},
		{
			name:    "JSON spec has CORS headers",
			handler: handlers.serveOpenAPISpecJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestRouterMethodRestrictions(t *testing.T) {
	router := mux.NewRouter()
	handlers := NewSwaggerHandlers()
	handlers.RegisterRoutes(router)

	// Test that non-GET methods return 405 Method Not Allowed
	paths := []string{"/openapi.yaml", "/openapi.json", "/swagger-ui", "/api-docs"}
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, path := range paths {
		for _, method := range methods {
			t.Run(method+" "+path, func(t *testing.T) {
				req := httptest.NewRequest(method, path, nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			})
		}
	}
}

func TestMultipleHandlerInstances(t *testing.T) {
	// Test that multiple instances can be created independently
	h1 := NewSwaggerHandlers()
	h2 := NewSwaggerHandlers()

	assert.NotNil(t, h1)
	assert.NotNil(t, h2)
	assert.NotSame(t, h1, h2)
}

func TestSwaggerUIContainsAuthorizationSupport(t *testing.T) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/swagger-ui", nil)
	w := httptest.NewRecorder()

	handlers.serveSwaggerUI(w, req)

	body := w.Body.String()

	// Verify authorization token support
	assert.Contains(t, body, "spoke_api_token")
	assert.Contains(t, body, "Authorization")
	assert.Contains(t, body, "Bearer")
	assert.Contains(t, body, "requestInterceptor")
}

func BenchmarkServeOpenAPISpec(b *testing.B) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/openapi.yaml", nil)

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.serveOpenAPISpec(w, req)
	}
}

func BenchmarkServeSwaggerUI(b *testing.B) {
	handlers := NewSwaggerHandlers()
	req := httptest.NewRequest("GET", "/swagger-ui", nil)

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.serveSwaggerUI(w, req)
	}
}

func TestRegisterRoutesMultipleTimes(t *testing.T) {
	// Test that registering routes multiple times doesn't cause issues
	router := mux.NewRouter()
	handlers := NewSwaggerHandlers()

	// Register routes twice
	handlers.RegisterRoutes(router)
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	// Should still work without panicking
	require.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})
}
