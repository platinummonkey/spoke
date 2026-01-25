package validation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/platinummonkey/spoke/pkg/api/protobuf"
)

// Normalizer normalizes protobuf schemas for consistent storage
type Normalizer struct {
	config *NormalizationConfig
}

// NormalizationConfig defines normalization rules
type NormalizationConfig struct {
	// SortFields sorts fields by field number
	SortFields bool
	// SortEnumValues sorts enum values by number
	SortEnumValues bool
	// SortImports sorts import statements alphabetically
	SortImports bool
	// CanonicalizeImports standardizes import paths
	CanonicalizeImports bool
	// PreserveComments keeps comments in normalized output
	PreserveComments bool
	// StandardizeWhitespace normalizes whitespace
	StandardizeWhitespace bool
	// RemoveTrailingWhitespace removes trailing whitespace
	RemoveTrailingWhitespace bool
}

// DefaultNormalizationConfig returns default normalization settings
func DefaultNormalizationConfig() *NormalizationConfig {
	return &NormalizationConfig{
		SortFields:               true,
		SortEnumValues:           true,
		SortImports:              true,
		CanonicalizeImports:      true,
		PreserveComments:         true,
		StandardizeWhitespace:    true,
		RemoveTrailingWhitespace: true,
	}
}

// NewNormalizer creates a new normalizer
func NewNormalizer(config *NormalizationConfig) *Normalizer {
	if config == nil {
		config = DefaultNormalizationConfig()
	}
	return &Normalizer{config: config}
}

// Normalize normalizes a protobuf AST
func (n *Normalizer) Normalize(ast *protobuf.RootNode) (*protobuf.RootNode, error) {
	// Create a normalized copy
	normalized := &protobuf.RootNode{
		Syntax:          ast.Syntax,
		Package:         ast.Package,
		Imports:         n.normalizeImports(ast.Imports),
		Options:         ast.Options,
		Messages:        n.normalizeMessages(ast.Messages),
		Enums:           n.normalizeEnums(ast.Enums),
		Services:        n.normalizeServices(ast.Services),
		Comments:        ast.Comments,
		SpokeDirectives: ast.SpokeDirectives,
		Pos:             ast.Pos,
		EndPos:          ast.EndPos,
	}

	return normalized, nil
}

func (n *Normalizer) normalizeImports(imports []*protobuf.ImportNode) []*protobuf.ImportNode {
	if !n.config.SortImports && !n.config.CanonicalizeImports {
		return imports
	}

	// Copy imports
	normalized := make([]*protobuf.ImportNode, len(imports))
	for i, imp := range imports {
		normalized[i] = &protobuf.ImportNode{
			Path:            n.canonicalizeImportPath(imp.Path),
			Public:          imp.Public,
			Weak:            imp.Weak,
			Comments:        imp.Comments,
			SpokeDirectives: imp.SpokeDirectives,
			Pos:             imp.Pos,
			EndPos:          imp.EndPos,
		}
	}

	if n.config.SortImports {
		sort.Slice(normalized, func(i, j int) bool {
			return normalized[i].Path < normalized[j].Path
		})
	}

	return normalized
}

func (n *Normalizer) canonicalizeImportPath(path string) string {
	if !n.config.CanonicalizeImports {
		return path
	}

	// Remove quotes if present
	path = strings.Trim(path, "\"")

	// Standardize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Remove redundant slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

func (n *Normalizer) normalizeMessages(messages []*protobuf.MessageNode) []*protobuf.MessageNode {
	normalized := make([]*protobuf.MessageNode, len(messages))
	for i, msg := range messages {
		normalized[i] = &protobuf.MessageNode{
			Name:            msg.Name,
			Fields:          n.normalizeFields(msg.Fields),
			Nested:          n.normalizeMessages(msg.Nested),
			Enums:           n.normalizeEnums(msg.Enums),
			OneOfs:          n.normalizeOneOfs(msg.OneOfs),
			Options:         msg.Options,
			Comments:        msg.Comments,
			SpokeDirectives: msg.SpokeDirectives,
			Pos:             msg.Pos,
			EndPos:          msg.EndPos,
		}
	}
	return normalized
}

func (n *Normalizer) normalizeFields(fields []*protobuf.FieldNode) []*protobuf.FieldNode {
	// Copy fields
	normalized := make([]*protobuf.FieldNode, len(fields))
	copy(normalized, fields)

	if n.config.SortFields {
		sort.Slice(normalized, func(i, j int) bool {
			return normalized[i].Number < normalized[j].Number
		})
	}

	return normalized
}

func (n *Normalizer) normalizeOneOfs(oneofs []*protobuf.OneOfNode) []*protobuf.OneOfNode {
	normalized := make([]*protobuf.OneOfNode, len(oneofs))
	for i, oneof := range oneofs {
		normalized[i] = &protobuf.OneOfNode{
			Name:            oneof.Name,
			Fields:          n.normalizeFields(oneof.Fields),
			Comments:        oneof.Comments,
			SpokeDirectives: oneof.SpokeDirectives,
			Pos:             oneof.Pos,
			EndPos:          oneof.EndPos,
		}
	}
	return normalized
}

func (n *Normalizer) normalizeEnums(enums []*protobuf.EnumNode) []*protobuf.EnumNode {
	normalized := make([]*protobuf.EnumNode, len(enums))
	for i, enum := range enums {
		normalized[i] = &protobuf.EnumNode{
			Name:            enum.Name,
			Values:          n.normalizeEnumValues(enum.Values),
			Options:         enum.Options,
			Comments:        enum.Comments,
			SpokeDirectives: enum.SpokeDirectives,
			Pos:             enum.Pos,
			EndPos:          enum.EndPos,
		}
	}
	return normalized
}

func (n *Normalizer) normalizeEnumValues(values []*protobuf.EnumValueNode) []*protobuf.EnumValueNode {
	// Copy values
	normalized := make([]*protobuf.EnumValueNode, len(values))
	copy(normalized, values)

	if n.config.SortEnumValues {
		sort.Slice(normalized, func(i, j int) bool {
			return normalized[i].Number < normalized[j].Number
		})
	}

	return normalized
}

func (n *Normalizer) normalizeServices(services []*protobuf.ServiceNode) []*protobuf.ServiceNode {
	// Services are typically kept in declaration order
	// but we could add sorting by name if needed
	return services
}

// NormalizeString normalizes a protobuf file content string
func (n *Normalizer) NormalizeString(content string) (string, error) {
	// Parse the content
	parser := protobuf.NewStringParser(content)
	ast, err := parser.Parse()
	if err != nil {
		return "", fmt.Errorf("failed to parse proto: %w", err)
	}

	// Normalize the AST
	normalizedAST, err := n.Normalize(ast)
	if err != nil {
		return "", fmt.Errorf("failed to normalize: %w", err)
	}

	// Serialize back to string
	serializer := NewSerializer(n.config)
	return serializer.Serialize(normalizedAST)
}

// CompareNormalized compares two schemas after normalization
// Returns true if they are semantically equivalent
func (n *Normalizer) CompareNormalized(content1, content2 string) (bool, error) {
	normalized1, err := n.NormalizeString(content1)
	if err != nil {
		return false, fmt.Errorf("failed to normalize first schema: %w", err)
	}

	normalized2, err := n.NormalizeString(content2)
	if err != nil {
		return false, fmt.Errorf("failed to normalize second schema: %w", err)
	}

	return normalized1 == normalized2, nil
}
