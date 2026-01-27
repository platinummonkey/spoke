package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// getExamples generates code examples for a specific language
func (s *Server) getExamples(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := vars["name"]
	version := vars["version"]
	language := vars["language"]

	// Get the version from storage
	ver, err := s.storage.GetVersion(moduleName, version)
	if err != nil {
		http.Error(w, fmt.Sprintf("Version not found: %v", err), http.StatusNotFound)
		return
	}

	// TODO: Implement example generation using the examples package
	// For now, return a placeholder
	fileCount := len(ver.Files)
	placeholder := fmt.Sprintf(`// Example code for %s
// Module: %s
// Version: %s
// Language: %s
// Proto files: %d

// Code generation coming soon!
`, moduleName, moduleName, version, language, fileCount)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(placeholder))
}
