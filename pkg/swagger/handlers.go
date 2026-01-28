package swagger

import (
	_ "embed"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/httputil"
)

//go:embed openapi.yaml
var openapiSpec []byte

// SwaggerHandlers provides HTTP handlers for OpenAPI/Swagger documentation
type SwaggerHandlers struct{}

// NewSwaggerHandlers creates a new SwaggerHandlers instance
func NewSwaggerHandlers() *SwaggerHandlers {
	return &SwaggerHandlers{}
}

// RegisterRoutes registers the swagger routes with the router
func (h *SwaggerHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/openapi.yaml", h.serveOpenAPISpec).Methods("GET")
	router.HandleFunc("/openapi.json", h.serveOpenAPISpecJSON).Methods("GET")
	router.HandleFunc("/swagger-ui", h.serveSwaggerUI).Methods("GET")
	router.HandleFunc("/api-docs", h.serveSwaggerUI).Methods("GET") // Alias
}

// serveOpenAPISpec serves the OpenAPI specification in YAML format
func (h *SwaggerHandlers) serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(openapiSpec)
}

// serveOpenAPISpecJSON serves the OpenAPI specification in JSON format
// Note: This requires converting YAML to JSON, which we'll implement if needed
func (h *SwaggerHandlers) serveOpenAPISpecJSON(w http.ResponseWriter, r *http.Request) {
	// For now, we'll just return the YAML version
	// TODO: Implement YAML to JSON conversion using gopkg.in/yaml.v3
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	httputil.WriteJSON(w, http.StatusNotImplemented, map[string]string{
		"error":   "Not implemented",
		"message": "JSON format not yet supported, use /openapi.yaml instead",
	})
}

// serveSwaggerUI serves the Swagger UI HTML page
func (h *SwaggerHandlers) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// Use Swagger UI CDN for convenience
	tmpl := template.Must(template.New("swagger").Parse(swaggerUITemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
}

const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Spoke API - Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/favicon-32x32.png" sizes="32x32" />
  <link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/favicon-16x16.png" sizes="16x16" />
  <style>
    html {
      box-sizing: border-box;
      overflow: -moz-scrollbars-vertical;
      overflow-y: scroll;
    }
    *, *:before, *:after {
      box-sizing: inherit;
    }
    body {
      margin:0;
      padding:0;
    }
  </style>
</head>
<body>
<div id="swagger-ui"></div>

<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script>
window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "/openapi.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    requestInterceptor: function(request) {
      // Add Authorization header if token is stored in localStorage
      const token = localStorage.getItem('spoke_api_token');
      if (token) {
        request.headers['Authorization'] = 'Bearer ' + token;
      }
      return request;
    }
  });
};
</script>
</body>
</html>`
