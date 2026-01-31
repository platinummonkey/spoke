package protobuf

import (
	"errors"
	"strings"
)

// ParseSpokeDirectivesFromContent extracts all @spoke directives and comments from proto file content.
// This function is used by both the legacy parser and the new descriptor-based parser.
//
// @spoke directives have the format: // @spoke:option:value
// Examples:
//   // @spoke:domain:github.com/example/test
//   // @spoke:option:required
//
// Returns directives and comments with line numbers for later association with AST nodes.
func ParseSpokeDirectivesFromContent(content string) (map[int]*SpokeDirectiveNode, map[int][]*CommentNode, error) {
	directives := make(map[int]*SpokeDirectiveNode)
	comments := make(map[int][]*CommentNode)

	lines := strings.Split(content, "\n")
	inBlockComment := false

	for lineNum, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Handle block comments
		if strings.Contains(line, "/*") {
			inBlockComment = true

			// Check if block comment ends on same line
			if strings.Contains(line, "*/") {
				inBlockComment = false
				// Extract text between /* and */
				blockText := extractBlockCommentText(line)
				if IsSpokeDirective(blockText) {
					directive, err := ExtractSpokeDirective(blockText, lineNum+1, 0)
					if err != nil {
						return nil, nil, err
					}
					directives[lineNum+1] = directive
				}
				continue
			}

			// Start of multi-line block comment
			// Extract any text after /*
			afterStart := line[strings.Index(line, "/*")+2:]
			afterStart = strings.TrimSpace(afterStart)
			if afterStart != "" && IsSpokeDirective(afterStart) {
				directive, err := ExtractSpokeDirective(afterStart, lineNum+1, 0)
				if err != nil {
					return nil, nil, err
				}
				directives[lineNum+1] = directive
			}
			continue
		}

		if inBlockComment {
			// Check if this line ends the block comment
			if strings.Contains(line, "*/") {
				inBlockComment = false
				// Extract text before */
				beforeEnd := line[:strings.Index(line, "*/")]
				beforeEnd = strings.TrimSpace(strings.TrimPrefix(beforeEnd, "*"))
				beforeEnd = strings.TrimSpace(beforeEnd)
				if beforeEnd != "" && IsSpokeDirective(beforeEnd) {
					directive, err := ExtractSpokeDirective(beforeEnd, lineNum+1, 0)
					if err != nil {
						return nil, nil, err
					}
					directives[lineNum+1] = directive
				}
				continue
			}

			// Line inside block comment
			commentLine := strings.TrimPrefix(line, "*")
			commentLine = strings.TrimSpace(commentLine)
			if commentLine != "" && IsSpokeDirective(commentLine) {
				directive, err := ExtractSpokeDirective(commentLine, lineNum+1, 0)
				if err != nil {
					return nil, nil, err
				}
				directives[lineNum+1] = directive
			}
			continue
		}

		// Check for line comments
		if strings.HasPrefix(line, "//") {
			commentText := strings.TrimPrefix(line, "//")
			commentText = strings.TrimSpace(commentText)

			if IsSpokeDirective(commentText) {
				directive, err := ExtractSpokeDirective(commentText, lineNum+1, 0)
				if err != nil {
					return nil, nil, err
				}
				directives[lineNum+1] = directive
			} else {
				comment := &CommentNode{
					Text: originalLine,
					Pos: Position{
						Line:   lineNum + 1,
						Column: 0,
						Offset: 0,
					},
				}
				comments[lineNum+1] = append(comments[lineNum+1], comment)
			}
		}
	}

	return directives, comments, nil
}

// extractBlockCommentText extracts text from a single-line block comment /* ... */
func extractBlockCommentText(line string) string {
	start := strings.Index(line, "/*")
	end := strings.Index(line, "*/")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	text := line[start+2 : end]
	return strings.TrimSpace(text)
}

// IsSpokeDirective checks if a comment text contains a spoke directive.
// A spoke directive starts with @spoke: followed by option:value
func IsSpokeDirective(text string) bool {
	return strings.HasPrefix(text, "@spoke:")
}

// ExtractSpokeDirective extracts a spoke directive from comment text.
// Expected format: @spoke:option:value
// Returns a SpokeDirectiveNode with the parsed option and value.
func ExtractSpokeDirective(text string, line, column int) (*SpokeDirectiveNode, error) {
	// Remove the @spoke: prefix
	if !strings.HasPrefix(text, "@spoke:") {
		return nil, errors.New("not a spoke directive")
	}

	directive := strings.TrimPrefix(text, "@spoke:")

	// Split on the second colon to get option and value
	parts := strings.SplitN(directive, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid spoke directive format, expected @spoke:option:value")
	}

	return &SpokeDirectiveNode{
		Option: strings.TrimSpace(parts[0]),
		Value:  strings.TrimSpace(parts[1]),
		Pos: Position{
			Line:   line,
			Column: column,
			Offset: 0,
		},
	}, nil
}

// AssociateSpokeDirectivesWithNode associates spoke directives and comments with AST nodes
// based on line number proximity. Directives/comments that appear immediately before a node
// are associated with that node.
func AssociateSpokeDirectivesWithNode(
	node interface{},
	directives map[int]*SpokeDirectiveNode,
	comments map[int][]*CommentNode,
	startLine int,
) {
	// Look for directives/comments in the 3 lines before the node
	// This handles cases where there are multiple comments before a declaration
	// but prevents directives from being associated too far away
	for line := startLine - 3; line < startLine; line++ {
		if line < 1 {
			continue
		}

		// Check if this node type supports spoke directives
		switch n := node.(type) {
		case *RootNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *SyntaxNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *PackageNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *ImportNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *OptionNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *MessageNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *FieldNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *EnumNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *EnumValueNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *ServiceNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		case *RPCNode:
			if directive, ok := directives[line]; ok {
				n.SpokeDirectives = append(n.SpokeDirectives, directive)
			}
			if commentList, ok := comments[line]; ok {
				n.Comments = append(n.Comments, commentList...)
			}
		}
	}
}
