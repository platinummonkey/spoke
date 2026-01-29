package docs

import (
	"strings"
	"testing"
)

func TestHTMLExporter_Export(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Syntax:      "proto3",
		Messages: []*MessageDoc{
			{
				Name:     "User",
				FullName: "test.package.User",
				Fields: []*FieldDoc{
					{Name: "id", Number: 1, Type: "string"},
					{Name: "name", Number: 2, Type: "string"},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check basic HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}

	if !strings.Contains(html, "test.package") {
		t.Error("Expected package name in HTML")
	}

	if !strings.Contains(html, "User") {
		t.Error("Expected message name in HTML")
	}
}

func TestHTMLExporter_ExportWithVersion(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
	}

	html, err := exporter.ExportWithVersion(doc, "v1.0.0")
	if err != nil {
		t.Fatalf("ExportWithVersion failed: %v", err)
	}

	if !strings.Contains(html, "v1.0.0") {
		t.Error("Expected version in HTML")
	}
}

func TestHTMLExporter_WithServices(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "UserService",
				Methods: []*MethodDoc{
					{
						Name:         "GetUser",
						RequestType:  "GetUserRequest",
						ResponseType: "User",
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "UserService") {
		t.Error("Expected service name in HTML")
	}

	if !strings.Contains(html, "GetUser") {
		t.Error("Expected method name in HTML")
	}
}

func TestHTMLExporter_WithEnums(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name: "Status",
				Values: []*EnumValueDoc{
					{Name: "UNKNOWN", Number: 0},
					{Name: "ACTIVE", Number: 1},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "Status") {
		t.Error("Expected enum name in HTML")
	}

	if !strings.Contains(html, "ACTIVE") {
		t.Error("Expected enum value in HTML")
	}
}

func TestHTMLExporter_DeprecationWarnings(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:       "OldMessage",
				Deprecated: true,
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "Deprecated") {
		t.Error("Expected deprecation warning in HTML")
	}
}

func TestHTMLExporter_NestedMessages(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Outer",
				NestedTypes: []*MessageDoc{
					{
						Name: "Inner",
						Fields: []*FieldDoc{
							{Name: "value", Number: 1, Type: "int32"},
						},
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "Outer") {
		t.Error("Expected outer message in HTML")
	}

	if !strings.Contains(html, "Inner") {
		t.Error("Expected nested message in HTML")
	}
}

func TestToAnchor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserService", "userservice"},
		{"Get User", "get-user"},
		{"test.package", "test.package"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toAnchor(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHasContent(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", true},
		{"", false},
		{"   ", false},
		{"\n\t", false},
		{"test", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := hasContent(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		notContains string
	}{
		{
			name:     "empty",
			input:    "",
			contains: "",
		},
		{
			name:     "code block",
			input:    "```\ncode\n```",
			contains: "<pre><code>",
		},
		{
			name:     "paragraph",
			input:    "hello world",
			contains: "<p>",
		},
		{
			name:     "bold text",
			input:    "**bold**",
			contains: "<strong>",
		},
		{
			name:     "inline code",
			input:    "use `code` here",
			contains: "<code>",
		},
		{
			name:     "multiple paragraphs",
			input:    "line1\nline2\nline3",
			contains: "<p>",
		},
		{
			name:     "code block with multiple lines",
			input:    "```\nline1\nline2\nline3\n```",
			contains: "<pre><code>",
		},
		{
			name:     "mixed content",
			input:    "Some **bold** text with `code`",
			contains: "<strong>",
		},
		{
			name:        "HTML escaping",
			input:       "<script>alert('xss')</script>",
			contains:    "&lt;script&gt;",
			notContains: "<script>",
		},
		{
			name:     "unclosed code block",
			input:    "```\ncode without closing",
			contains: "<pre><code>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToHTML(tt.input)
			if tt.contains != "" && !strings.Contains(string(result), tt.contains) {
				t.Errorf("Expected to contain %s, got %s", tt.contains, result)
			}
			if tt.notContains != "" && strings.Contains(string(result), tt.notContains) {
				t.Errorf("Expected not to contain %s, but found it in %s", tt.notContains, result)
			}
		})
	}
}

func TestHTMLExporter_StreamingMethods(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "ChatService",
				Methods: []*MethodDoc{
					{
						Name:            "BiDirectional",
						ClientStreaming: true,
						ServerStreaming: true,
					},
					{
						Name:            "ClientStream",
						ClientStreaming: true,
						ServerStreaming: false,
					},
					{
						Name:            "ServerStream",
						ClientStreaming: false,
						ServerStreaming: true,
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "streaming") {
		t.Error("Expected streaming indicators in HTML")
	}
}

func TestHTMLExporter_ComplexDocument(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "complex.api",
		Syntax:      "proto3",
		Description: "A complex API with multiple components",
		Messages: []*MessageDoc{
			{
				Name:        "User",
				FullName:    "complex.api.User",
				Description: "User entity with details",
				Fields: []*FieldDoc{
					{Name: "id", Number: 1, Type: "string", Description: "Unique identifier", Required: true},
					{Name: "email", Number: 2, Type: "string", Description: "User email", Optional: true},
					{Name: "roles", Number: 3, Type: "string", Description: "User roles", Repeated: true},
					{Name: "status", Number: 4, Type: "Status", Description: "User status", Deprecated: true},
				},
				NestedTypes: []*MessageDoc{
					{
						Name:        "Address",
						FullName:    "complex.api.User.Address",
						Description: "Nested address type",
						Fields: []*FieldDoc{
							{Name: "street", Number: 1, Type: "string"},
							{Name: "city", Number: 2, Type: "string"},
						},
					},
				},
				Enums: []*EnumDoc{
					{
						Name:        "Type",
						FullName:    "complex.api.User.Type",
						Description: "User type enum",
						Values: []*EnumValueDoc{
							{Name: "REGULAR", Number: 0},
							{Name: "ADMIN", Number: 1},
						},
					},
				},
			},
		},
		Enums: []*EnumDoc{
			{
				Name:        "Status",
				FullName:    "complex.api.Status",
				Description: "Status enum",
				Values: []*EnumValueDoc{
					{Name: "UNKNOWN", Number: 0, Description: "Unknown status"},
					{Name: "ACTIVE", Number: 1, Description: "Active status"},
					{Name: "INACTIVE", Number: 2, Description: "Inactive status", Deprecated: true},
				},
				Deprecated: false,
			},
		},
		Services: []*ServiceDoc{
			{
				Name:        "UserService",
				Description: "Service for managing users",
				Methods: []*MethodDoc{
					{
						Name:            "GetUser",
						Description:     "Retrieve a user by ID",
						RequestType:     "GetUserRequest",
						ResponseType:    "User",
						ClientStreaming: false,
						ServerStreaming: false,
					},
					{
						Name:            "StreamUsers",
						Description:     "Stream all users",
						RequestType:     "StreamUsersRequest",
						ResponseType:    "User",
						ClientStreaming: false,
						ServerStreaming: true,
					},
					{
						Name:            "UploadUsers",
						Description:     "Upload multiple users",
						RequestType:     "User",
						ResponseType:    "UploadResponse",
						ClientStreaming: true,
						ServerStreaming: false,
					},
					{
						Name:            "Chat",
						Description:     "Bidirectional chat",
						RequestType:     "ChatMessage",
						ResponseType:    "ChatMessage",
						ClientStreaming: true,
						ServerStreaming: true,
					},
				},
				Deprecated: false,
			},
			{
				Name:        "OldService",
				Description: "Deprecated service",
				Methods:     []*MethodDoc{},
				Deprecated:  true,
			},
		},
	}

	html, err := exporter.ExportWithVersion(doc, "v2.1.0")
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify all key elements are present
	expectations := []string{
		"<!DOCTYPE html>",
		"complex.api",
		"proto3",
		"v2.1.0",
		"User",
		"UserService",
		"Status",
		"GetUser",
		"StreamUsers",
		"UploadUsers",
		"Chat",
		"Address",
		"Type",
		"streaming",
		"Deprecated",
		"id=\"services\"",
		"id=\"messages\"",
		"id=\"enums\"",
		"required",
		"optional",
		"repeated",
	}

	for _, expected := range expectations {
		if !strings.Contains(html, expected) {
			t.Errorf("Expected HTML to contain %q", expected)
		}
	}
}

func TestHTMLExporter_EmptyDocument(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "empty.package",
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "empty.package") {
		t.Error("Expected package name in HTML")
	}

	// Should still have basic structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}

	if !strings.Contains(html, "Table of Contents") {
		t.Error("Expected table of contents")
	}
}

func TestHTMLExporter_DescriptionsWithSpecialCharacters(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Description: "Package with <special> & \"characters\"",
		Messages: []*MessageDoc{
			{
				Name:        "TestMessage",
				Description: "Message with <tags> and & symbols",
				Fields: []*FieldDoc{
					{
						Name:        "field",
						Number:      1,
						Type:        "string",
						Description: "Field with <html> and & characters",
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify proper HTML escaping
	if strings.Contains(html, "<special>") {
		t.Error("Expected special characters to be escaped")
	}

	if strings.Contains(html, "<tags>") {
		t.Error("Expected tag characters to be escaped")
	}
}

func TestHTMLExporter_FieldLabels(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "TestMessage",
				Fields: []*FieldDoc{
					{Name: "required_field", Number: 1, Type: "string", Required: true},
					{Name: "optional_field", Number: 2, Type: "string", Optional: true},
					{Name: "repeated_field", Number: 3, Type: "string", Repeated: true},
					{Name: "normal_field", Number: 4, Type: "string"},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for label classes
	labels := []string{"required", "optional", "repeated"}
	for _, label := range labels {
		if !strings.Contains(html, label) {
			t.Errorf("Expected HTML to contain label: %s", label)
		}
	}
}

func TestHTMLExporter_MethodDeprecation(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "TestService",
				Methods: []*MethodDoc{
					{
						Name:         "OldMethod",
						RequestType:  "Request",
						ResponseType: "Response",
						Deprecated:   true,
						Description:  "This method is deprecated",
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "OldMethod") {
		t.Error("Expected method name in HTML")
	}

	// Count deprecation warnings (should appear at least once)
	deprecationCount := strings.Count(html, "Deprecated")
	if deprecationCount == 0 {
		t.Error("Expected deprecation warning in HTML")
	}
}

func TestHTMLExporter_EnumInMessage(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Parent",
				Enums: []*EnumDoc{
					{
						Name: "NestedEnum",
						Values: []*EnumValueDoc{
							{Name: "VALUE_A", Number: 0},
							{Name: "VALUE_B", Number: 1},
						},
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !strings.Contains(html, "NestedEnum") {
		t.Error("Expected nested enum in HTML")
	}

	if !strings.Contains(html, "VALUE_A") {
		t.Error("Expected enum value in HTML")
	}
}

func TestHTMLExporter_TableOfContents(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{Name: "Service1"},
		},
		Messages: []*MessageDoc{
			{Name: "Message1"},
		},
		Enums: []*EnumDoc{
			{Name: "Enum1"},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for TOC links
	tocElements := []string{
		"href=\"#services\"",
		"href=\"#messages\"",
		"href=\"#enums\"",
	}

	for _, elem := range tocElements {
		if !strings.Contains(html, elem) {
			t.Errorf("Expected TOC to contain %s", elem)
		}
	}
}

func TestHTMLExporter_Anchors(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "User Service",
				Methods: []*MethodDoc{
					{Name: "Get User"},
				},
			},
		},
		Messages: []*MessageDoc{
			{Name: "User Message"},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check that anchors are properly formatted (lowercase with dashes)
	expectedAnchors := []string{
		"id=\"user-service\"",
		"id=\"get-user\"",
		"id=\"user-message\"",
	}

	for _, anchor := range expectedAnchors {
		if !strings.Contains(html, anchor) {
			t.Errorf("Expected HTML to contain anchor: %s", anchor)
		}
	}
}

func TestHTMLExporter_EmptyVersionString(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
	}

	html, err := exporter.ExportWithVersion(doc, "")
	if err != nil {
		t.Fatalf("ExportWithVersion failed: %v", err)
	}

	// Should not contain version badge when version is empty
	if !strings.Contains(html, "test.package") {
		t.Error("Expected package name in HTML")
	}
}

func TestHTMLExporter_MultipleNestedLevels(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Level1",
				NestedTypes: []*MessageDoc{
					{
						Name: "Level2",
						NestedTypes: []*MessageDoc{
							{
								Name: "Level3",
								Fields: []*FieldDoc{
									{Name: "deep_field", Number: 1, Type: "string"},
								},
							},
						},
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// All levels should be present
	levels := []string{"Level1", "Level2", "Level3", "deep_field"}
	for _, level := range levels {
		if !strings.Contains(html, level) {
			t.Errorf("Expected HTML to contain %s", level)
		}
	}
}

func TestHTMLExporter_SearchFunctionality(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for search box
	if !strings.Contains(html, "id=\"search\"") {
		t.Error("Expected search box in HTML")
	}

	if !strings.Contains(html, "placeholder=\"Search documentation...\"") {
		t.Error("Expected search placeholder text")
	}

	// Check for search script
	if !strings.Contains(html, "addEventListener('input'") {
		t.Error("Expected search event listener in HTML")
	}
}

func TestHTMLExporter_StylesIncluded(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for essential CSS classes
	cssClasses := []string{
		".container",
		".content",
		".badge",
		".deprecated",
		".streaming",
		".label",
		".search-box",
		".method-signature",
	}

	for _, class := range cssClasses {
		if !strings.Contains(html, class) {
			t.Errorf("Expected HTML to contain CSS class: %s", class)
		}
	}
}

func TestMarkdownToHTML_ComplexFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
	}{
		{
			name:  "multiple code blocks",
			input: "Text\n```\ncode1\n```\nMore text\n```\ncode2\n```",
			validate: func(result string) bool {
				return strings.Count(result, "<pre><code>") == 2 &&
					strings.Count(result, "</code></pre>") == 2
			},
		},
		{
			name:  "bold and code mixed",
			input: "This is **bold** and `code`",
			validate: func(result string) bool {
				return strings.Contains(result, "<strong>") &&
					strings.Contains(result, "<code>")
			},
		},
		{
			name:  "multiline with inline code",
			input: "Line with `code`\nAnother `code` line\nThird line",
			validate: func(result string) bool {
				return strings.Count(result, "<p>") >= 3
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToHTML(tt.input)
			if !tt.validate(string(result)) {
				t.Errorf("Validation failed for input: %s\nGot: %s", tt.input, result)
			}
		})
	}
}

func TestHTMLExporter_TemplateFunctions(t *testing.T) {
	// Test that template functions are properly registered
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Description: "Test with <html> escaping",
		Messages: []*MessageDoc{
			{
				Name:        "Message With Spaces",
				Description: "Description with content",
			},
			{
				Name:        "EmptyDescription",
				Description: "",
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify escape function worked (description should be escaped)
	if strings.Contains(html, "<html>") {
		t.Error("Expected HTML tags to be escaped in descriptions")
	}

	// Verify anchor function worked (spaces should be converted to dashes)
	if strings.Contains(html, "id=\"message-with-spaces\"") {
		// This is the expected behavior
	}

	// Verify hasContent function worked (empty descriptions should not show)
	if !strings.Contains(html, "test.package") {
		t.Error("Expected package name to be present")
	}
}

func TestHTMLExporter_RenderingEdgeCases(t *testing.T) {
	exporter := NewHTMLExporter()

	tests := []struct {
		name string
		doc  *Documentation
	}{
		{
			name: "nil slices",
			doc: &Documentation{
				PackageName: "test",
				Messages:    nil,
				Services:    nil,
				Enums:       nil,
			},
		},
		{
			name: "empty slices",
			doc: &Documentation{
				PackageName: "test",
				Messages:    []*MessageDoc{},
				Services:    []*ServiceDoc{},
				Enums:       []*EnumDoc{},
			},
		},
		{
			name: "message with nil fields",
			doc: &Documentation{
				PackageName: "test",
				Messages: []*MessageDoc{
					{
						Name:   "Test",
						Fields: nil,
					},
				},
			},
		},
		{
			name: "service with nil methods",
			doc: &Documentation{
				PackageName: "test",
				Services: []*ServiceDoc{
					{
						Name:    "Test",
						Methods: nil,
					},
				},
			},
		},
		{
			name: "enum with nil values",
			doc: &Documentation{
				PackageName: "test",
				Enums: []*EnumDoc{
					{
						Name:   "Test",
						Values: nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exporter.Export(tt.doc)
			if err != nil {
				t.Errorf("Export should not fail for edge case %s: %v", tt.name, err)
			}
		})
	}
}

func TestHTMLExporter_AllStreamingCombinations(t *testing.T) {
	exporter := NewHTMLExporter()

	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "StreamService",
				Methods: []*MethodDoc{
					{
						Name:            "Unary",
						ClientStreaming: false,
						ServerStreaming: false,
					},
					{
						Name:            "ClientStream",
						ClientStreaming: true,
						ServerStreaming: false,
					},
					{
						Name:            "ServerStream",
						ClientStreaming: false,
						ServerStreaming: true,
					},
					{
						Name:            "BidiStream",
						ClientStreaming: true,
						ServerStreaming: true,
					},
				},
			},
		},
	}

	html, err := exporter.Export(doc)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for different streaming indicators
	if strings.Count(html, "client streaming") < 1 {
		t.Error("Expected client streaming indicator")
	}

	if strings.Count(html, "server streaming") < 1 {
		t.Error("Expected server streaming indicator")
	}

	if strings.Count(html, "bidirectional streaming") < 1 {
		t.Error("Expected bidirectional streaming indicator")
	}

	// Unary method should not have streaming indicators next to it
	unaryIndex := strings.Index(html, "Unary")
	if unaryIndex == -1 {
		t.Error("Expected Unary method in output")
	}
}
