// Package api provides compilation routing and download handlers
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/analytics"
	"github.com/platinummonkey/spoke/pkg/async"
)

// downloadCompiled handles GET /modules/{name}/versions/{version}/download/{language}
// DEPRECATED: Use v2 compilation API instead
func (s *Server) downloadCompiled(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	vars := mux.Vars(r)
	language := Language(vars["language"])

	// Get the version
	version, err := s.storage.GetVersion(vars["name"], vars["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Find compilation info for the requested language
	var compilationInfo *CompilationInfo
	for _, info := range version.CompilationInfo {
		if info.Language == language {
			compilationInfo = &info
			break
		}
	}

	if compilationInfo == nil {
		http.Error(w, "compiled version not found", http.StatusNotFound)
		return
	}

	// Calculate file size
	var fileSize int64
	for _, file := range compilationInfo.Files {
		fileSize += int64(len(file.Content))
	}

	// Set appropriate headers based on language
	switch language {
	case LanguageGo:
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-go.zip", vars["name"], vars["version"]))
	case LanguagePython:
		w.Header().Set("Content-Type", "application/x-python-package")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-py.whl", vars["name"], vars["version"]))
	}

	// Stream the compiled files
	success := true
	var downloadErr error
	for _, file := range compilationInfo.Files {
		if _, err := w.Write([]byte(file.Content)); err != nil {
			success = false
			downloadErr = err
			http.Error(w, err.Error(), http.StatusInternalServerError)
			break
		}
	}

	// Track download event asynchronously (non-blocking)
	if s.eventTracker != nil {
		async.SafeGo(r.Context(), 5*time.Second, "track download", func(ctx context.Context) error {
			event := analytics.DownloadEvent{
				UserID:         analytics.ExtractUserID(r),
				OrganizationID: analytics.ExtractOrganizationID(r),
				ModuleName:     vars["name"],
				Version:        vars["version"],
				Language:       string(language),
				FileSize:       fileSize,
				Duration:       time.Since(startTime),
				Success:        success,
				IPAddress:      analytics.GetClientIP(r),
				UserAgent:      analytics.GetUserAgent(r),
				ClientSDK:      analytics.GetClientSDK(r),
				ClientVersion:  analytics.GetClientVersion(r),
				CacheHit:       false, // TODO: detect cache hit from response headers
			}
			if downloadErr != nil {
				event.ErrorMessage = downloadErr.Error()
			}

			return s.eventTracker.TrackDownload(ctx, event)
		})
	}
}

// compileForLanguage compiles a version using the v2 orchestrator
func (s *Server) compileForLanguage(version *Version, language Language) (CompilationInfo, error) {
	return s.compileWithGenerator(version, language)
}
