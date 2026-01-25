package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// SearchEntry represents a single entry in the search index
type SearchEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Messages    []string `json:"messages"`
	Enums       []string `json:"enums"`
	Services    []string `json:"services"`
	Methods     []string `json:"methods"`
	Fields      []string `json:"fields"`
}

// SearchIndex contains all search entries
type SearchIndex struct {
	Modules []SearchEntry `json:"modules"`
}

func main() {
	storageDir := flag.String("storage-dir", "./storage", "Path to storage directory")
	outputFile := flag.String("output", "web/public/search-index.json", "Output file path")
	flag.Parse()

	log.Printf("Building search index from storage directory: %s", *storageDir)

	// Initialize storage
	store, err := storage.NewFileSystemStorage(*storageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Get all modules
	modules, err := store.ListModules()
	if err != nil {
		log.Fatalf("Failed to list modules: %v", err)
	}

	log.Printf("Found %d modules", len(modules))

	// Build index
	index := SearchIndex{
		Modules: []SearchEntry{},
	}

	for _, module := range modules {
		// Get all versions for this module
		versions, err := store.ListVersions(module.Name)
		if err != nil {
			log.Printf("Warning: Failed to list versions for module %s: %v", module.Name, err)
			continue
		}

		for _, version := range versions {
			// Get full version details
			ver, err := store.GetVersion(module.Name, version.Version)
			if err != nil {
				log.Printf("Warning: Failed to get version %s@%s: %v", module.Name, version.Version, err)
				continue
			}

			// Extract searchable data
			entry := extractSearchData(module, ver)
			index.Modules = append(index.Modules, entry)
		}
	}

	log.Printf("Built index with %d entries", len(index.Modules))

	// Write index to file
	if err := writeIndex(&index, *outputFile); err != nil {
		log.Fatalf("Failed to write index: %v", err)
	}

	log.Printf("Search index written to: %s", *outputFile)
}

// extractSearchData extracts searchable data from a version
func extractSearchData(module *api.Module, version *api.Version) SearchEntry {
	entry := SearchEntry{
		ID:          fmt.Sprintf("%s-%s", module.Name, version.Version),
		Name:        module.Name,
		Version:     version.Version,
		Description: module.Description,
		Messages:    []string{},
		Enums:       []string{},
		Services:    []string{},
		Methods:     []string{},
		Fields:      []string{},
	}

	// Parse proto files to extract types
	// For now, we'll do basic content scanning
	// In a full implementation, we'd use a proto parser

	for _, file := range version.Files {
		content := file.Content

		// Extract messages (simplified)
		messages := extractPattern(content, `message\s+(\w+)`)
		entry.Messages = append(entry.Messages, messages...)

		// Extract enums
		enums := extractPattern(content, `enum\s+(\w+)`)
		entry.Enums = append(entry.Enums, enums...)

		// Extract services
		services := extractPattern(content, `service\s+(\w+)`)
		entry.Services = append(entry.Services, services...)

		// Extract RPC methods
		methods := extractPattern(content, `rpc\s+(\w+)`)
		entry.Methods = append(entry.Methods, methods...)

		// Extract field names (simplified - matches common patterns)
		fields := extractPattern(content, `\s+(\w+)\s+=\s+\d+;`)
		entry.Fields = append(entry.Fields, fields...)
	}

	// Deduplicate
	entry.Messages = deduplicate(entry.Messages)
	entry.Enums = deduplicate(entry.Enums)
	entry.Services = deduplicate(entry.Services)
	entry.Methods = deduplicate(entry.Methods)
	entry.Fields = deduplicate(entry.Fields)

	return entry
}

// extractPattern extracts all matches for a regex pattern
func extractPattern(content, pattern string) []string {
	// Simple string-based extraction
	// In production, use regexp package
	results := []string{}

	// This is a placeholder - real implementation would use regexp
	// For now, just return empty slice
	// TODO: Implement proper regex extraction

	return results
}

// deduplicate removes duplicates from a string slice
func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// writeIndex writes the search index to a JSON file
func writeIndex(index *SearchIndex, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Calculate and log size
	sizeKB := len(data) / 1024
	log.Printf("Index size: %d KB", sizeKB)

	return nil
}

// extractTypes extracts types from content using basic string parsing
func extractTypes(content string) (messages, enums, services []string) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				messages = append(messages, parts[1])
			}
		} else if strings.HasPrefix(line, "enum ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				enums = append(enums, parts[1])
			}
		} else if strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				services = append(services, parts[1])
			}
		}
	}

	return
}
