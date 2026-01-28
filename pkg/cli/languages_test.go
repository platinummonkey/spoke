package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLanguagesCommand tests the languages command initialization
func TestNewLanguagesCommand(t *testing.T) {
	cmd := newLanguagesCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "languages", cmd.Name)
	assert.Equal(t, "Language management commands", cmd.Description)
	assert.NotNil(t, cmd.Subcommands)
	assert.NotNil(t, cmd.Run)

	// Verify subcommands are registered
	assert.NotNil(t, cmd.Subcommands["list"])
	assert.NotNil(t, cmd.Subcommands["show"])
}

// TestNewLanguagesListCommand tests the languages list command initialization
func TestNewLanguagesListCommand(t *testing.T) {
	cmd := newLanguagesListCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name)
	assert.Equal(t, "List all supported languages", cmd.Description)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)

	// Verify flags are registered
	assert.NotNil(t, cmd.Flags.Lookup("registry"))
	assert.NotNil(t, cmd.Flags.Lookup("json"))
}

// TestNewLanguagesShowCommand tests the languages show command initialization
func TestNewLanguagesShowCommand(t *testing.T) {
	cmd := newLanguagesShowCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "show", cmd.Name)
	assert.Equal(t, "Show details for a specific language", cmd.Description)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Run)

	// Verify flags are registered
	assert.NotNil(t, cmd.Flags.Lookup("registry"))
	assert.NotNil(t, cmd.Flags.Lookup("json"))
}

// TestRunLanguagesHelp tests the help output
func TestRunLanguagesHelp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesHelp([]string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	// We can't easily read the output in this test, but we verify it doesn't error
}

// TestRunLanguagesNoArgs tests running languages command without arguments
func TestRunLanguagesNoArgs(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguages([]string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
}

// TestRunLanguagesUnknownSubcommand tests handling of unknown subcommand
func TestRunLanguagesUnknownSubcommand(t *testing.T) {
	err := runLanguages([]string{"unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown languages subcommand")
}

// TestRunLanguagesListSubcommand tests routing to list subcommand
func TestRunLanguagesListSubcommand(t *testing.T) {
	// Create a test server
	languages := []LanguageInfo{
		{
			ID:               "go",
			Name:             "Go",
			DisplayName:      "Go",
			SupportsGRPC:     true,
			FileExtensions:   []string{".go"},
			Enabled:          true,
			Stable:           true,
			Description:      "Go language support",
			DocumentationURL: "https://golang.org",
			PluginVersion:    "1.0.0",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguages([]string{"list", "-registry", server.URL})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesList tests the list command with successful response
func TestRunLanguagesList(t *testing.T) {
	languages := []LanguageInfo{
		{
			ID:               "go",
			Name:             "Go",
			DisplayName:      "Go",
			SupportsGRPC:     true,
			FileExtensions:   []string{".go"},
			Enabled:          true,
			Stable:           true,
			Description:      "Go language support",
			DocumentationURL: "https://golang.org",
			PluginVersion:    "1.0.0",
		},
		{
			ID:               "python",
			Name:             "Python",
			DisplayName:      "Python",
			SupportsGRPC:     false,
			FileExtensions:   []string{".py"},
			Enabled:          true,
			Stable:           false,
			Description:      "Python language support",
			DocumentationURL: "https://python.org",
			PluginVersion:    "0.9.0",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/languages", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesList([]string{"-registry", server.URL})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesListJSON tests the list command with JSON output
func TestRunLanguagesListJSON(t *testing.T) {
	languages := []LanguageInfo{
		{
			ID:               "go",
			Name:             "Go",
			DisplayName:      "Go",
			SupportsGRPC:     true,
			FileExtensions:   []string{".go", ".mod"},
			Enabled:          true,
			Stable:           true,
			Description:      "Go language support",
			DocumentationURL: "https://golang.org",
			PluginVersion:    "1.0.0",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesList([]string{"-registry", server.URL, "-json"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesListServerError tests handling of server errors
func TestRunLanguagesListServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	err := runLanguagesList([]string{"-registry", server.URL})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry returned error")
}

// TestRunLanguagesListConnectionError tests handling of connection errors
func TestRunLanguagesListConnectionError(t *testing.T) {
	err := runLanguagesList([]string{"-registry", "http://localhost:9999"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to registry")
}

// TestRunLanguagesListInvalidJSON tests handling of invalid JSON response
func TestRunLanguagesListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	err := runLanguagesList([]string{"-registry", server.URL})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

// TestRunLanguagesShow tests the show command with successful response
func TestRunLanguagesShow(t *testing.T) {
	language := LanguageInfo{
		ID:               "go",
		Name:             "Go",
		DisplayName:      "Go Programming Language",
		SupportsGRPC:     true,
		FileExtensions:   []string{".go", ".mod", ".sum"},
		Enabled:          true,
		Stable:           true,
		Description:      "Go is a statically typed, compiled programming language",
		DocumentationURL: "https://golang.org",
		PluginVersion:    "1.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/languages/go", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(language)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesShow([]string{"-registry", server.URL, "go"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesShowJSON tests the show command with JSON output
func TestRunLanguagesShowJSON(t *testing.T) {
	language := LanguageInfo{
		ID:               "python",
		Name:             "Python",
		DisplayName:      "Python Programming Language",
		SupportsGRPC:     false,
		FileExtensions:   []string{".py"},
		Enabled:          true,
		Stable:           true,
		Description:      "Python is an interpreted high-level programming language",
		DocumentationURL: "https://python.org",
		PluginVersion:    "0.9.5",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(language)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesShow([]string{"-registry", server.URL, "-json", "python"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesShowNoID tests show command without language ID
func TestRunLanguagesShowNoID(t *testing.T) {
	err := runLanguagesShow([]string{"-registry", "http://localhost:8080"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "language ID required")
}

// TestRunLanguagesShowServerError tests handling of server errors
func TestRunLanguagesShowServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Language not found"))
	}))
	defer server.Close()

	err := runLanguagesShow([]string{"-registry", server.URL, "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry returned error")
}

// TestRunLanguagesShowConnectionError tests handling of connection errors
func TestRunLanguagesShowConnectionError(t *testing.T) {
	err := runLanguagesShow([]string{"-registry", "http://localhost:9999", "go"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to registry")
}

// TestRunLanguagesShowInvalidJSON tests handling of invalid JSON response
func TestRunLanguagesShowInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	err := runLanguagesShow([]string{"-registry", server.URL, "go"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

// TestLanguageInfoStruct tests the LanguageInfo struct
func TestLanguageInfoStruct(t *testing.T) {
	lang := LanguageInfo{
		ID:               "rust",
		Name:             "Rust",
		DisplayName:      "Rust Programming Language",
		SupportsGRPC:     true,
		FileExtensions:   []string{".rs"},
		Enabled:          true,
		Stable:           true,
		Description:      "Rust is a multi-paradigm programming language",
		DocumentationURL: "https://rust-lang.org",
		PluginVersion:    "1.2.0",
	}

	assert.Equal(t, "rust", lang.ID)
	assert.Equal(t, "Rust", lang.Name)
	assert.True(t, lang.SupportsGRPC)
	assert.True(t, lang.Stable)
	assert.Len(t, lang.FileExtensions, 1)
}

// TestLanguageInfoJSON tests JSON marshaling/unmarshaling
func TestLanguageInfoJSON(t *testing.T) {
	lang := LanguageInfo{
		ID:               "java",
		Name:             "Java",
		DisplayName:      "Java Programming Language",
		SupportsGRPC:     true,
		FileExtensions:   []string{".java", ".class"},
		Enabled:          true,
		Stable:           true,
		Description:      "Java is a class-based, object-oriented programming language",
		DocumentationURL: "https://java.com",
		PluginVersion:    "2.0.0",
	}

	// Marshal to JSON
	data, err := json.Marshal(lang)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded LanguageInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, lang.ID, decoded.ID)
	assert.Equal(t, lang.Name, decoded.Name)
	assert.Equal(t, lang.DisplayName, decoded.DisplayName)
	assert.Equal(t, lang.SupportsGRPC, decoded.SupportsGRPC)
	assert.Equal(t, lang.FileExtensions, decoded.FileExtensions)
	assert.Equal(t, lang.Enabled, decoded.Enabled)
	assert.Equal(t, lang.Stable, decoded.Stable)
	assert.Equal(t, lang.Description, decoded.Description)
	assert.Equal(t, lang.DocumentationURL, decoded.DocumentationURL)
	assert.Equal(t, lang.PluginVersion, decoded.PluginVersion)
}

// TestRunLanguagesListEmptyResponse tests handling of empty language list
func TestRunLanguagesListEmptyResponse(t *testing.T) {
	languages := []LanguageInfo{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesList([]string{"-registry", server.URL})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesListMultipleExtensions tests display with multiple file extensions
func TestRunLanguagesListMultipleExtensions(t *testing.T) {
	languages := []LanguageInfo{
		{
			ID:             "cpp",
			Name:           "C++",
			DisplayName:    "C++",
			SupportsGRPC:   true,
			FileExtensions: []string{".cpp", ".cc", ".cxx", ".h", ".hpp"},
			Enabled:        true,
			Stable:         true,
			PluginVersion:  "1.5.0",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesList([]string{"-registry", server.URL})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesShowMultipleExtensions tests show command with multiple extensions
func TestRunLanguagesShowMultipleExtensions(t *testing.T) {
	language := LanguageInfo{
		ID:               "typescript",
		Name:             "TypeScript",
		DisplayName:      "TypeScript",
		SupportsGRPC:     true,
		FileExtensions:   []string{".ts", ".tsx", ".d.ts"},
		Enabled:          true,
		Stable:           true,
		Description:      "TypeScript is a typed superset of JavaScript",
		DocumentationURL: "https://typescriptlang.org",
		PluginVersion:    "1.1.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(language)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLanguagesShow([]string{"-registry", server.URL, "typescript"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesListFlagParsing tests flag parsing
func TestRunLanguagesListFlagParsing(t *testing.T) {
	languages := []LanguageInfo{
		{
			ID:            "kotlin",
			Name:          "Kotlin",
			PluginVersion: "1.0.0",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(languages)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with custom registry URL
	err := runLanguagesList([]string{fmt.Sprintf("-registry=%s", server.URL)})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}

// TestRunLanguagesShowFlagParsing tests flag parsing for show command
func TestRunLanguagesShowFlagParsing(t *testing.T) {
	language := LanguageInfo{
		ID:            "swift",
		Name:          "Swift",
		DisplayName:   "Swift",
		PluginVersion: "1.0.0",
		Description:   "Swift programming language",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(language)
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with custom registry URL
	err := runLanguagesShow([]string{fmt.Sprintf("-registry=%s", server.URL), "swift"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old
	r.Close()
}
