package api

// NewServerWithoutRoutes creates a new API server without setting up routes
func NewServerWithoutRoutes(storage Storage) *Server {
	return &Server{
		storage: storage,
	}
}

// CompileGo compiles a version into Go code using the orchestrator
func (s *Server) CompileGo(version *Version) (CompilationInfo, error) {
	return s.compileForLanguage(version, LanguageGo)
}

// CompilePython compiles a version into Python code using the orchestrator
func (s *Server) CompilePython(version *Version) (CompilationInfo, error) {
	return s.compileForLanguage(version, LanguagePython)
} 