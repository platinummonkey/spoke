package docs

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// DocsHandlers provides HTTP handlers for documentation
type DocsHandlers struct {
	storage        api.Storage
	generator      *Generator
	htmlExporter   *HTMLExporter
	markdownExporter *MarkdownExporter
}

// NewDocsHandlers creates new documentation handlers
func NewDocsHandlers(storage api.Storage) *DocsHandlers {
	return &DocsHandlers{
		storage:        storage,
		generator:      NewGenerator(),
		htmlExporter:   NewHTMLExporter(),
		markdownExporter: NewMarkdownExporter(),
	}
}

// RegisterRoutes registers documentation routes
func (h *DocsHandlers) RegisterRoutes(router *mux.Router) {
	// Get documentation for a specific version
	router.HandleFunc("/docs/{module}/{version}", h.getVersionDocs).Methods("GET")
	router.HandleFunc("/docs/{module}/{version}/markdown", h.getVersionDocsMarkdown).Methods("GET")
	router.HandleFunc("/docs/{module}/{version}/json", h.getVersionDocsJSON).Methods("GET")

	// Compare versions
	router.HandleFunc("/docs/{module}/compare", h.compareVersions).Methods("GET")
}

// getVersionDocs handles GET /docs/{module}/{version}
func (h *DocsHandlers) getVersionDocs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["module"]
	version := vars["version"]

	// Get version from storage
	ver, err := h.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if len(ver.Files) == 0 {
		http.Error(w, "no proto files in version", http.StatusNotFound)
		return
	}

	// Parse proto file
	parser := protobuf.NewStringParser(ver.Files[0].Content)
	ast, err := parser.Parse()
	if err != nil {
		http.Error(w, "failed to parse proto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate documentation
	doc, err := h.generator.Generate(ast)
	if err != nil {
		http.Error(w, "failed to generate documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Export to HTML
	html, err := h.htmlExporter.ExportWithVersion(doc, version)
	if err != nil {
		http.Error(w, "failed to export HTML: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// getVersionDocsMarkdown handles GET /docs/{module}/{version}/markdown
func (h *DocsHandlers) getVersionDocsMarkdown(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["module"]
	version := vars["version"]

	// Get version from storage
	ver, err := h.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if len(ver.Files) == 0 {
		http.Error(w, "no proto files in version", http.StatusNotFound)
		return
	}

	// Parse proto file
	parser := protobuf.NewStringParser(ver.Files[0].Content)
	ast, err := parser.Parse()
	if err != nil {
		http.Error(w, "failed to parse proto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate documentation
	doc, err := h.generator.Generate(ast)
	if err != nil {
		http.Error(w, "failed to generate documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Export to Markdown
	markdown := h.markdownExporter.ExportWithVersion(doc, version)

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+moduleName+"-"+version+".md")
	w.Write([]byte(markdown))
}

// getVersionDocsJSON handles GET /docs/{module}/{version}/json
func (h *DocsHandlers) getVersionDocsJSON(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["module"]
	version := vars["version"]

	// Get version from storage
	ver, err := h.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if len(ver.Files) == 0 {
		http.Error(w, "no proto files in version", http.StatusNotFound)
		return
	}

	// Parse proto file
	parser := protobuf.NewStringParser(ver.Files[0].Content)
	ast, err := parser.Parse()
	if err != nil {
		http.Error(w, "failed to parse proto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate documentation
	doc, err := h.generator.Generate(ast)
	if err != nil {
		http.Error(w, "failed to generate documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// compareVersions handles GET /docs/{module}/compare?old={version}&new={version}
func (h *DocsHandlers) compareVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["module"]

	oldVersion := r.URL.Query().Get("old")
	newVersion := r.URL.Query().Get("new")

	if oldVersion == "" || newVersion == "" {
		http.Error(w, "old and new version parameters required", http.StatusBadRequest)
		return
	}

	// Get old version
	oldVer, err := h.storage.GetVersion(moduleName, oldVersion)
	if err != nil {
		http.Error(w, "old version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Get new version
	newVer, err := h.storage.GetVersion(moduleName, newVersion)
	if err != nil {
		http.Error(w, "new version not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Parse old version
	oldParser := protobuf.NewStringParser(oldVer.Files[0].Content)
	oldAST, err := oldParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse old proto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse new version
	newParser := protobuf.NewStringParser(newVer.Files[0].Content)
	newAST, err := newParser.Parse()
	if err != nil {
		http.Error(w, "failed to parse new proto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate documentation for both versions
	oldDoc, err := h.generator.Generate(oldAST)
	if err != nil {
		http.Error(w, "failed to generate old documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	newDoc, err := h.generator.Generate(newAST)
	if err != nil {
		http.Error(w, "failed to generate new documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Compare documentation
	diff := h.compareDocs(oldDoc, newDoc)

	// Return comparison
	response := map[string]interface{}{
		"old_version": oldVersion,
		"new_version": newVersion,
		"changes":     diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// compareDocs compares two documentation objects and returns changes
func (h *DocsHandlers) compareDocs(oldDoc, newDoc *Documentation) map[string]interface{} {
	changes := make(map[string]interface{})

	// Compare messages
	messagesAdded := []string{}
	messagesRemoved := []string{}
	messagesModified := []string{}

	oldMessages := make(map[string]*MessageDoc)
	for _, msg := range oldDoc.Messages {
		oldMessages[msg.Name] = msg
	}

	newMessages := make(map[string]*MessageDoc)
	for _, msg := range newDoc.Messages {
		newMessages[msg.Name] = msg
	}

	// Find added messages
	for name := range newMessages {
		if _, exists := oldMessages[name]; !exists {
			messagesAdded = append(messagesAdded, name)
		}
	}

	// Find removed messages
	for name := range oldMessages {
		if _, exists := newMessages[name]; !exists {
			messagesRemoved = append(messagesRemoved, name)
		}
	}

	// Find modified messages (simplified - just check field count)
	for name, newMsg := range newMessages {
		if oldMsg, exists := oldMessages[name]; exists {
			if len(oldMsg.Fields) != len(newMsg.Fields) {
				messagesModified = append(messagesModified, name)
			}
		}
	}

	changes["messages"] = map[string]interface{}{
		"added":    messagesAdded,
		"removed":  messagesRemoved,
		"modified": messagesModified,
	}

	// Compare services
	servicesAdded := []string{}
	servicesRemoved := []string{}

	oldServices := make(map[string]*ServiceDoc)
	for _, svc := range oldDoc.Services {
		oldServices[svc.Name] = svc
	}

	newServices := make(map[string]*ServiceDoc)
	for _, svc := range newDoc.Services {
		newServices[svc.Name] = svc
	}

	for name := range newServices {
		if _, exists := oldServices[name]; !exists {
			servicesAdded = append(servicesAdded, name)
		}
	}

	for name := range oldServices {
		if _, exists := newServices[name]; !exists {
			servicesRemoved = append(servicesRemoved, name)
		}
	}

	changes["services"] = map[string]interface{}{
		"added":   servicesAdded,
		"removed": servicesRemoved,
	}

	return changes
}
