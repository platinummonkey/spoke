package codegen

import "sync"

// PackageRequest represents a package file generation request
type PackageRequest struct {
	ModuleName  string
	Version     string
	Language    string
	IncludeGRPC bool
	Options     map[string]string
}

// PackageGenerator generates package manager files
type PackageGenerator interface {
	Generate(req *PackageRequest) ([]GeneratedFile, error)
}

var (
	packageGenerators = make(map[string]PackageGenerator)
	pkgGenMu         sync.RWMutex
)

// RegisterPackageGenerator registers a package generator
func RegisterPackageGenerator(name string, gen PackageGenerator) {
	pkgGenMu.Lock()
	defer pkgGenMu.Unlock()
	packageGenerators[name] = gen
}

// GetPackageGenerator retrieves a package generator
func GetPackageGenerator(name string) PackageGenerator {
	pkgGenMu.RLock()
	defer pkgGenMu.RUnlock()
	return packageGenerators[name]
}
