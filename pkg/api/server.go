package api

// NewServerWithoutRoutes creates a new API server without setting up routes
func NewServerWithoutRoutes(storage Storage) *Server {
	return &Server{
		storage: storage,
	}
}

// CompileGo compiles a version into Go code, exposed for external use
// Routes to v1 (legacy) or v2 (orchestrator) based on SPOKE_CODEGEN_VERSION env var
func (s *Server) CompileGo(version *Version) (CompilationInfo, error) {
	return s.compileForLanguage(version, LanguageGo)
}

// CompilePython compiles a version into Python code, exposed for external use
// Routes to v1 (legacy) or v2 (orchestrator) based on SPOKE_CODEGEN_VERSION env var
func (s *Server) CompilePython(version *Version) (CompilationInfo, error) {
	return s.compileForLanguage(version, LanguagePython)
} 