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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToHTML(tt.input)
			if tt.contains != "" && !strings.Contains(string(result), tt.contains) {
				t.Errorf("Expected to contain %s, got %s", tt.contains, result)
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
