package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/platinummonkey/spoke/pkg/api"
)

// CompilationRequest represents a request to compile a module version
type CompilationRequest struct {
	ModuleName string
	Version    string
	Timestamp  time.Time
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ModuleName  string
	Version     string
	Dependencies map[string]struct{} // Map of "moduleName@version" for quick lookups
	Dependents   map[string]struct{} // Map of "moduleName@version" for quick lookups
}

// DependencyGraph manages module dependencies for efficient compilation
type DependencyGraph struct {
	nodes map[string]*DependencyNode // Map of "moduleName@version" to node
	mu    sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
	}
}

// AddNode adds or updates a node in the dependency graph
func (g *DependencyGraph) AddNode(moduleName, version string, dependencies []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := fmt.Sprintf("%s@%s", moduleName, version)
	node, exists := g.nodes[key]
	if !exists {
		node = &DependencyNode{
			ModuleName:   moduleName,
			Version:      version,
			Dependencies: make(map[string]struct{}),
			Dependents:   make(map[string]struct{}),
		}
		g.nodes[key] = node
	}

	// Update dependencies
	// First clear existing ones
	for depKey := range node.Dependencies {
		if depNode, ok := g.nodes[depKey]; ok {
			delete(depNode.Dependents, key)
		}
	}
	node.Dependencies = make(map[string]struct{})

	// Add new dependencies
	for _, dep := range dependencies {
		node.Dependencies[dep] = struct{}{}
		
		// Create the dependency node if it doesn't exist
		if _, ok := g.nodes[dep]; !ok {
			parts := parseDependencyKey(dep)
			if len(parts) == 2 {
				g.nodes[dep] = &DependencyNode{
					ModuleName:   parts[0],
					Version:      parts[1],
					Dependencies: make(map[string]struct{}),
					Dependents:   make(map[string]struct{}),
				}
			}
		}
		
		// Add this node as a dependent of its dependency
		if depNode, ok := g.nodes[dep]; ok {
			depNode.Dependents[key] = struct{}{}
		}
	}
}

// GetDependentsTree returns all direct and indirect dependents of a node
func (g *DependencyGraph) GetDependentsTree(moduleName, version string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := fmt.Sprintf("%s@%s", moduleName, version)
	result := make(map[string]struct{})
	visited := make(map[string]struct{})
	
	g.collectDependents(key, result, visited)
	
	// Convert map to slice
	dependents := make([]string, 0, len(result))
	for dep := range result {
		dependents = append(dependents, dep)
	}
	
	return dependents
}

// collectDependents recursively collects all dependents
func (g *DependencyGraph) collectDependents(key string, result, visited map[string]struct{}) {
	if _, ok := visited[key]; ok {
		return // Already visited
	}
	visited[key] = struct{}{}
	
	node, exists := g.nodes[key]
	if !exists {
		return
	}
	
	for dependent := range node.Dependents {
		result[dependent] = struct{}{}
		g.collectDependents(dependent, result, visited)
	}
}

// parseDependencyKey splits a dependency key into module name and version
func parseDependencyKey(key string) []string {
	// Handle "latest" specially
	if key == "latest" {
		return []string{}
	}
	
	parts := strings.Split(key, "@")
	if len(parts) != 2 {
		return []string{}
	}
	
	return parts
}

// Compiler watches for changes to proto files and recompiles them after a delay
type Compiler struct {
	storage       api.Storage
	delay         time.Duration
	queue         map[string]*CompilationRequest
	pendingMutex  sync.Mutex
	compileMutex  sync.Mutex
	stopChan      chan struct{}
	compileTicker *time.Ticker
	graph         *DependencyGraph
	processedSet  map[string]time.Time // Track recently processed modules to avoid duplicate work
}

// NewCompiler creates a new compiler with the given delay
func NewCompiler(storage api.Storage, delay time.Duration) *Compiler {
	return &Compiler{
		storage:      storage,
		delay:        delay,
		queue:        make(map[string]*CompilationRequest),
		stopChan:     make(chan struct{}),
		graph:        NewDependencyGraph(),
		processedSet: make(map[string]time.Time),
	}
}

// Start begins the compilation process
func (c *Compiler) Start() {
	c.compileTicker = time.NewTicker(1 * time.Second)
	go c.processQueue()
	
	// Start a separate goroutine to clean up the processed set
	go c.cleanProcessedSet()
}

// Stop stops the compilation process
func (c *Compiler) Stop() {
	if c.compileTicker != nil {
		c.compileTicker.Stop()
	}
	close(c.stopChan)
}

// cleanProcessedSet periodically cleans up the processed set
func (c *Compiler) cleanProcessedSet() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.pendingMutex.Lock()
			now := time.Now()
			for key, timestamp := range c.processedSet {
				// Remove entries older than 2 minutes
				if now.Sub(timestamp) > 2*time.Minute {
					delete(c.processedSet, key)
				}
			}
			c.pendingMutex.Unlock()
		}
	}
}

// QueueRecompilation adds a module version to the compilation queue
func (c *Compiler) QueueRecompilation(moduleName, version string) {
	c.QueueRecompilationWithDelay(moduleName, version, 0)
}

// QueueRecompilationWithDelay adds a module version to the compilation queue with a specified delay
func (c *Compiler) QueueRecompilationWithDelay(moduleName, version string, additionalDelay time.Duration) {
	c.pendingMutex.Lock()
	defer c.pendingMutex.Unlock()

	key := fmt.Sprintf("%s@%s", moduleName, version)
	
	// Skip if recently processed
	if lastProcessed, exists := c.processedSet[key]; exists {
		if time.Since(lastProcessed) < c.delay*2 {
			log.Printf("Skipping recently processed module: %s", key)
			return
		}
	}
	
	// If already in queue, just update the timestamp
	if existing, exists := c.queue[key]; exists {
		existing.Timestamp = time.Now().Add(-additionalDelay)
		log.Printf("Updated existing compilation request for %s", key)
		return
	}
	
	c.queue[key] = &CompilationRequest{
		ModuleName: moduleName,
		Version:    version,
		Timestamp:  time.Now().Add(-additionalDelay),
	}
	log.Printf("Queued recompilation for %s with delay %v", key, additionalDelay)
}

// processQueue processes the compilation queue
func (c *Compiler) processQueue() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-c.compileTicker.C:
			c.checkQueue()
		}
	}
}

// checkQueue checks the queue for items that are ready to be compiled
func (c *Compiler) checkQueue() {
	c.pendingMutex.Lock()
	var readyItems []CompilationRequest
	now := time.Now()

	// Find items ready for compilation
	for key, request := range c.queue {
		if now.Sub(request.Timestamp) >= c.delay {
			readyItems = append(readyItems, *request)
			delete(c.queue, key)
			
			// Mark as processed
			c.processedSet[key] = now
		}
	}
	c.pendingMutex.Unlock()

	// Process ready items
	for _, request := range readyItems {
		c.compile(request)
	}
}

// compile compiles a module version
func (c *Compiler) compile(request CompilationRequest) {
	// Avoid concurrent compilations of the same module
	c.compileMutex.Lock()
	defer c.compileMutex.Unlock()

	log.Printf("Compiling %s@%s", request.ModuleName, request.Version)
	
	// Get the version from storage
	version, err := c.storage.GetVersion(request.ModuleName, request.Version)
	if err != nil {
		log.Printf("Error getting version: %v", err)
		return
	}

	// Update dependency graph
	c.graph.AddNode(request.ModuleName, request.Version, version.Dependencies)

	// Compile for each supported language
	for _, lang := range []api.Language{api.LanguageGo, api.LanguagePython} {
		log.Printf("Compiling %s@%s for %s", request.ModuleName, request.Version, lang)
		c.compileLanguage(version, lang)
	}

	// Find all dependents and queue them for compilation
	dependents := c.graph.GetDependentsTree(request.ModuleName, request.Version)
	if len(dependents) > 0 {
		log.Printf("Found %d dependent modules to recompile", len(dependents))
		for _, depKey := range dependents {
			parts := parseDependencyKey(depKey)
			if len(parts) == 2 {
				// Add a small staggered delay to avoid overloading the system
				c.pendingMutex.Lock()
				c.queue[depKey] = &CompilationRequest{
					ModuleName: parts[0],
					Version:    parts[1],
					Timestamp:  time.Now().Add(-c.delay + 500*time.Millisecond),
				}
				c.pendingMutex.Unlock()
				log.Printf("Queued dependent module: %s", depKey)
			}
		}
	}
}

// compileLanguage compiles a specific language
func (c *Compiler) compileLanguage(version *api.Version, language api.Language) {
	var compilationInfo api.CompilationInfo
	var err error

	// Create a temporary server instance for compilation
	tempServer := api.NewServerWithoutRoutes(c.storage)

	// Compile based on language
	switch language {
	case api.LanguageGo:
		compilationInfo, err = tempServer.CompileGo(version)
	case api.LanguagePython:
		compilationInfo, err = tempServer.CompilePython(version)
	default:
		err = fmt.Errorf("unsupported language: %s", language)
	}

	if err != nil {
		log.Printf("Error compiling %s@%s for %s: %v", version.ModuleName, version.Version, language, err)
		return
	}

	// Update compilation info
	updated := false
	for i, info := range version.CompilationInfo {
		if info.Language == language {
			version.CompilationInfo[i] = compilationInfo
			updated = true
			break
		}
	}
	if !updated {
		version.CompilationInfo = append(version.CompilationInfo, compilationInfo)
	}

	// Save updated version
	if err := c.storage.UpdateVersion(version); err != nil {
		log.Printf("Error updating version: %v", err)
	}
} 