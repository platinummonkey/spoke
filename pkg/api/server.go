package api

// NewServerWithoutRoutes creates a new API server without setting up routes
func NewServerWithoutRoutes(storage Storage) *Server {
	return &Server{
		storage: storage,
	}
}

// CompileGo compiles a version into Go code, exposed for external use
func (s *Server) CompileGo(version *Version) (CompilationInfo, error) {
	return s.compileGo(version)
}

// CompilePython compiles a version into Python code, exposed for external use
func (s *Server) CompilePython(version *Version) (CompilationInfo, error) {
	return s.compilePython(version)
} 