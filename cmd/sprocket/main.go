package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/platinummonkey/spoke/pkg/codegen/languages"
	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/platinummonkey/spoke/pkg/plugins/buf"
	"github.com/platinummonkey/spoke/pkg/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	storageDir := flag.String("storage-dir", filepath.Join(os.TempDir(), "spoke"), "Directory to store protobuf files")
	delaySeconds := flag.Int("delay", 10, "Delay in seconds before recompiling changed protos")
	flag.Parse()

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(*storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize storage
	store, err := storage.NewFileSystemStorage(*storageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized in %s", *storageDir)

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Start watching the storage directory
	if err := setupWatcher(watcher, *storageDir); err != nil {
		log.Fatalf("Failed to setup watcher: %v", err)
	}

	// Initialize language registry and load plugins
	langRegistry := initializeLanguageRegistry()

	// Create compiler with language registry
	compiler := NewCompiler(store, time.Duration(*delaySeconds)*time.Second, langRegistry)

	// Start the compiler
	compiler.Start()
	defer compiler.Stop()

	// Scan for existing modules and queue them for compilation
	scanExistingModules(*storageDir, compiler)

	// Process events
	log.Printf("Started watching for proto file changes in %s", *storageDir)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only care about write and create events for .proto files
			if (event.Op&(fsnotify.Write|fsnotify.Create) != 0) && filepath.Ext(event.Name) == ".proto" {
				log.Printf("Modified file: %s", event.Name)
				relPath, err := filepath.Rel(*storageDir, event.Name)
				if err != nil {
					log.Printf("Error getting relative path: %v", err)
					continue
				}

				// Extract module and version from path
				pathParts := strings.Split(relPath, string(filepath.Separator))
				if len(pathParts) < 4 || pathParts[1] != "versions" {
					// Not a valid module path
					log.Printf("Not a valid module path: %s", relPath)
					continue
				}
				moduleName := pathParts[0]
				versionName := pathParts[2]

				// Queue the file for recompilation
				compiler.QueueRecompilation(moduleName, versionName)
			}

			// Also watch new directories
			if event.Op&fsnotify.Create != 0 {
				fi, err := os.Stat(event.Name)
				if err == nil && fi.IsDir() {
					log.Printf("New directory: %s", event.Name)
					if err := watcher.Add(event.Name); err != nil {
						log.Printf("Error watching new directory: %v", err)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// setupWatcher recursively adds all directories to the watcher
func setupWatcher(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

// scanExistingModules scans the storage directory for existing modules and queues them for compilation
func scanExistingModules(storageDir string, compiler *Compiler) {
	log.Printf("Scanning for existing modules...")
	
	// Walk through the modules directory
	modulesDir := storageDir
	moduleEntries, err := os.ReadDir(modulesDir)
	if err != nil {
		log.Printf("Error reading modules directory: %v", err)
		return
	}

	// First, build the dependency graph for all modules
	for _, moduleEntry := range moduleEntries {
		if !moduleEntry.IsDir() {
			continue
		}
		
		moduleName := moduleEntry.Name()
		versionsDir := filepath.Join(modulesDir, moduleName, "versions")
		
		// Check if versions directory exists
		versionEntries, err := os.ReadDir(versionsDir)
		if err != nil {
			continue // Skip if versions directory doesn't exist or can't be read
		}
		
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			
			versionName := versionEntry.Name()
			log.Printf("Found existing module: %s@%s", moduleName, versionName)
			
			// Read the version.json file to get dependency information
			versionFile := filepath.Join(versionsDir, versionName, "version.json")
			data, err := os.ReadFile(versionFile)
			if err != nil {
				log.Printf("Error reading version file %s: %v", versionFile, err)
				continue
			}
			
			var version struct {
				Dependencies []string `json:"dependencies"`
			}
			
			if err := json.Unmarshal(data, &version); err != nil {
				log.Printf("Error parsing version file %s: %v", versionFile, err)
				continue
			}
			
			// Add to dependency graph
			compiler.graph.AddNode(moduleName, versionName, version.Dependencies)
		}
	}
	
	// Now, identify modules to compile
	// Start with those that have no dependents (leaf nodes)
	modulesToCompile := make(map[string]struct{})

	// Process all modules to find those without dependents
	for _, moduleEntry := range moduleEntries {
		if !moduleEntry.IsDir() {
			continue
		}

		moduleName := moduleEntry.Name()
		versionsDir := filepath.Join(modulesDir, moduleName, "versions")

		versionEntries, err := os.ReadDir(versionsDir)
		if err != nil {
			continue
		}

		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}

			versionName := versionEntry.Name()
			key := fmt.Sprintf("%s@%s", moduleName, versionName)

			// Check if this module has any dependents
			dependents := compiler.graph.GetDependentsTree(moduleName, versionName)
			if len(dependents) == 0 {
				// This is a leaf node, compile it first
				modulesToCompile[key] = struct{}{}
			}
		}
	}

	log.Printf("Found %d leaf modules to compile first", len(modulesToCompile))

	// Queue leaf modules for compilation with staggered delays
	i := 0
	for key := range modulesToCompile {
		parts := strings.Split(key, "@")
		if len(parts) != 2 {
			continue
		}

		// Add with staggered delay to avoid overwhelming the system
		delay := time.Duration(i*500) * time.Millisecond
		compiler.QueueRecompilationWithDelay(parts[0], parts[1], delay)
		i++
	}
}

// initializeLanguageRegistry initializes the language registry and loads plugins
func initializeLanguageRegistry() *languages.Registry {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create language registry
	registry := languages.NewRegistry()

	// Load default built-in languages (for backward compatibility)
	// These would be loaded from the default languages package if it exists
	log.Println("Initializing language registry...")

	// Load plugins from default directories
	pluginDirs := plugins.GetDefaultPluginDirectories()
	pluginLoader := plugins.NewLoader(pluginDirs, logger)

	// Configure Buf plugin support
	buf.ConfigureLoader(pluginLoader)
	log.Println("Buf plugin support enabled")

	ctx := context.Background()
	if err := registry.LoadPlugins(ctx, pluginLoader, logger); err != nil {
		log.Printf("Warning: Failed to load plugins: %v", err)
	}

	// Log registered languages
	enabledLanguages := registry.ListEnabled()
	log.Printf("Loaded %d enabled languages", len(enabledLanguages))
	for _, langSpec := range enabledLanguages {
		log.Printf("  - %s (%s)", langSpec.Name, langSpec.ID)
	}

	return registry
} 