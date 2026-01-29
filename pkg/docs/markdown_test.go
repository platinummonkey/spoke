package docs

import (
	"strings"
	"testing"
)

func TestNewMarkdownExporter(t *testing.T) {
	exporter := NewMarkdownExporter()
	if exporter == nil {
		t.Fatal("Expected non-nil MarkdownExporter")
	}
}

func TestMarkdownExporter_Export_EmptyDoc(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
	}

	result := exporter.Export(doc)
	if result == "" {
		t.Fatal("Expected non-empty result")
	}

	// Should contain title
	if !strings.Contains(result, "# test.package") {
		t.Error("Expected title with package name")
	}

	// Should contain table of contents header
	if !strings.Contains(result, "## Table of Contents") {
		t.Error("Expected table of contents")
	}
}

func TestMarkdownExporter_Export_WithSyntax(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Syntax:      "proto3",
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**Syntax:** `proto3`") {
		t.Error("Expected syntax in output")
	}
}

func TestMarkdownExporter_Export_WithDescription(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Description: "This is a test package description",
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "This is a test package description") {
		t.Error("Expected description in output")
	}
}

func TestMarkdownExporter_Export_WithMessages(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:        "User",
				Description: "User message",
				Fields: []*FieldDoc{
					{
						Name:        "id",
						Type:        "string",
						Label:       "required",
						Description: "User ID",
						Number:      1,
					},
					{
						Name:        "name",
						Type:        "string",
						Label:       "optional",
						Description: "User name",
						Number:      2,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Check for Messages section
	if !strings.Contains(result, "## Messages") {
		t.Error("Expected Messages section")
	}

	// Check for message name
	if !strings.Contains(result, "### User") {
		t.Error("Expected User message header")
	}

	// Check for description
	if !strings.Contains(result, "User message") {
		t.Error("Expected User message description")
	}

	// Check for field table
	if !strings.Contains(result, "| Field | Type | Label | Description |") {
		t.Error("Expected field table header")
	}

	// Check for field entries
	if !strings.Contains(result, "| id | `string` | required | User ID |") {
		t.Error("Expected id field in table")
	}
	if !strings.Contains(result, "| name | `string` | optional | User name |") {
		t.Error("Expected name field in table")
	}

	// Check table of contents
	if !strings.Contains(result, "- [Messages](#messages)") {
		t.Error("Expected Messages in table of contents")
	}
}

func TestMarkdownExporter_Export_WithEnums(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name:        "Status",
				Description: "Status enum",
				Values: []*EnumValueDoc{
					{
						Name:        "UNKNOWN",
						Number:      0,
						Description: "Unknown status",
					},
					{
						Name:        "ACTIVE",
						Number:      1,
						Description: "Active status",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Check for Enums section
	if !strings.Contains(result, "## Enums") {
		t.Error("Expected Enums section")
	}

	// Check for enum name
	if !strings.Contains(result, "### Status") {
		t.Error("Expected Status enum header")
	}

	// Check for description
	if !strings.Contains(result, "Status enum") {
		t.Error("Expected Status enum description")
	}

	// Check for value table
	if !strings.Contains(result, "| Name | Number | Description |") {
		t.Error("Expected enum value table header")
	}

	// Check for value entries
	if !strings.Contains(result, "| UNKNOWN | 0 | Unknown status |") {
		t.Error("Expected UNKNOWN value in table")
	}
	if !strings.Contains(result, "| ACTIVE | 1 | Active status |") {
		t.Error("Expected ACTIVE value in table")
	}

	// Check table of contents
	if !strings.Contains(result, "- [Enums](#enums)") {
		t.Error("Expected Enums in table of contents")
	}
}

func TestMarkdownExporter_Export_WithServices(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name:        "UserService",
				Description: "User service",
				Methods: []*MethodDoc{
					{
						Name:         "GetUser",
						Description:  "Get user by ID",
						RequestType:  "GetUserRequest",
						ResponseType: "User",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Check for Services section
	if !strings.Contains(result, "## Services") {
		t.Error("Expected Services section")
	}

	// Check for service name
	if !strings.Contains(result, "### UserService") {
		t.Error("Expected UserService header")
	}

	// Check for description
	if !strings.Contains(result, "User service") {
		t.Error("Expected UserService description")
	}

	// Check for Methods section
	if !strings.Contains(result, "#### Methods") {
		t.Error("Expected Methods section")
	}

	// Check for method name
	if !strings.Contains(result, "##### `GetUser`") {
		t.Error("Expected GetUser method header")
	}

	// Check for method description
	if !strings.Contains(result, "Get user by ID") {
		t.Error("Expected GetUser method description")
	}

	// Check for protobuf code block
	if !strings.Contains(result, "```protobuf") {
		t.Error("Expected protobuf code block")
	}
	if !strings.Contains(result, "rpc GetUser (GetUserRequest) returns (User)") {
		t.Error("Expected RPC signature")
	}

	// Check table of contents
	if !strings.Contains(result, "- [Services](#services)") {
		t.Error("Expected Services in table of contents")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedMessage(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:       "OldMessage",
				Deprecated: true,
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**⚠️ Deprecated**") {
		t.Error("Expected deprecated warning for message")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedField(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Message",
				Fields: []*FieldDoc{
					{
						Name:       "old_field",
						Type:       "string",
						Deprecated: true,
						Description: "Old field",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "⚠️ **Deprecated**") {
		t.Error("Expected deprecated warning for field")
	}
	if !strings.Contains(result, "Old field") {
		t.Error("Expected field description to be preserved")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedEnum(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name:       "OldEnum",
				Deprecated: true,
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**⚠️ Deprecated**") {
		t.Error("Expected deprecated warning for enum")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedEnumValue(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name: "Status",
				Values: []*EnumValueDoc{
					{
						Name:        "OLD_VALUE",
						Number:      0,
						Deprecated:  true,
						Description: "Old value",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "⚠️ **Deprecated**") {
		t.Error("Expected deprecated warning for enum value")
	}
	if !strings.Contains(result, "Old value") {
		t.Error("Expected enum value description to be preserved")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedService(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name:       "OldService",
				Deprecated: true,
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**⚠️ Deprecated**") {
		t.Error("Expected deprecated warning for service")
	}
}

func TestMarkdownExporter_Export_WithDeprecatedMethod(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "Service",
				Methods: []*MethodDoc{
					{
						Name:         "OldMethod",
						RequestType:  "Request",
						ResponseType: "Response",
						Deprecated:   true,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**⚠️ Deprecated**") {
		t.Error("Expected deprecated warning for method")
	}
}

func TestMarkdownExporter_Export_WithNestedMessages(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "OuterMessage",
				NestedTypes: []*MessageDoc{
					{
						Name:        "InnerMessage",
						Description: "Nested message",
						Fields: []*FieldDoc{
							{
								Name: "field",
								Type: "string",
							},
						},
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Check for nested message with proper header depth
	if !strings.Contains(result, "#### InnerMessage") {
		t.Error("Expected nested message with #### header")
	}
	if !strings.Contains(result, "Nested message") {
		t.Error("Expected nested message description")
	}
}

func TestMarkdownExporter_Export_WithDeeplyNestedMessages(t *testing.T) {
	exporter := NewMarkdownExporter()
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
							},
						},
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Check for proper header depths
	if !strings.Contains(result, "### Level1") {
		t.Error("Expected Level1 with ### header")
	}
	if !strings.Contains(result, "#### Level2") {
		t.Error("Expected Level2 with #### header")
	}
	if !strings.Contains(result, "##### Level3") {
		t.Error("Expected Level3 with ##### header")
	}
}

func TestMarkdownExporter_Export_WithNestedEnums(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Message",
				Enums: []*EnumDoc{
					{
						Name: "NestedEnum",
						Values: []*EnumValueDoc{
							{Name: "VALUE", Number: 0},
						},
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "### NestedEnum") {
		t.Error("Expected nested enum")
	}
}

func TestMarkdownExporter_Export_WithClientStreaming(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "StreamService",
				Methods: []*MethodDoc{
					{
						Name:            "Upload",
						RequestType:     "UploadRequest",
						ResponseType:    "UploadResponse",
						ClientStreaming: true,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "(client streaming)") {
		t.Error("Expected client streaming notation")
	}
}

func TestMarkdownExporter_Export_WithServerStreaming(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "StreamService",
				Methods: []*MethodDoc{
					{
						Name:            "Download",
						RequestType:     "DownloadRequest",
						ResponseType:    "DownloadResponse",
						ServerStreaming: true,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "(server streaming)") {
		t.Error("Expected server streaming notation")
	}
}

func TestMarkdownExporter_Export_WithBidirectionalStreaming(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "StreamService",
				Methods: []*MethodDoc{
					{
						Name:            "Chat",
						RequestType:     "ChatMessage",
						ResponseType:    "ChatMessage",
						ClientStreaming: true,
						ServerStreaming: true,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "(bidirectional streaming)") {
		t.Error("Expected bidirectional streaming notation")
	}
}

func TestMarkdownExporter_Export_WithHTTPAnnotations(t *testing.T) {
	exporter := NewMarkdownExporter()
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
						HTTPMethod:   "GET",
						HTTPPath:     "/v1/users/{id}",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "**HTTP:** `GET /v1/users/{id}`") {
		t.Error("Expected HTTP annotation")
	}
}

func TestMarkdownExporter_Export_WithOneofField(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Message",
				Fields: []*FieldDoc{
					{
						Name:        "option_a",
						Type:        "string",
						OneofName:   "choice",
						Description: "First option",
					},
					{
						Name:        "option_b",
						Type:        "int32",
						OneofName:   "choice",
						Description: "Second option",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	if !strings.Contains(result, "(oneof choice)") {
		t.Error("Expected oneof notation")
	}
	if !strings.Contains(result, "First option") {
		t.Error("Expected field description to be preserved")
	}
}

func TestMarkdownExporter_Export_WithEmptyLabel(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Message",
				Fields: []*FieldDoc{
					{
						Name:  "field",
						Type:  "string",
						Label: "", // Empty label
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Should replace empty label with "-"
	lines := strings.Split(result, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "| field |") && strings.Contains(line, "| - |") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected empty label to be replaced with '-'")
	}
}

func TestMarkdownExporter_Export_CompleteDocument(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "complete.package",
		Syntax:      "proto3",
		Description: "A complete test package",
		Messages: []*MessageDoc{
			{
				Name:        "User",
				Description: "User entity",
				Fields: []*FieldDoc{
					{Name: "id", Type: "string", Label: "required"},
					{Name: "name", Type: "string", Label: "optional"},
				},
			},
		},
		Enums: []*EnumDoc{
			{
				Name: "Status",
				Values: []*EnumValueDoc{
					{Name: "ACTIVE", Number: 0},
				},
			},
		},
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

	result := exporter.Export(doc)

	// Verify all sections are present
	expectedSections := []string{
		"# complete.package",
		"**Syntax:** `proto3`",
		"A complete test package",
		"## Table of Contents",
		"- [Services](#services)",
		"- [Messages](#messages)",
		"- [Enums](#enums)",
		"## Services",
		"## Messages",
		"## Enums",
	}

	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("Expected section: %s", section)
		}
	}
}

func TestMarkdownExporter_ExportWithVersion(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Syntax:      "proto3",
	}

	result := exporter.ExportWithVersion(doc, "v1.0.0")

	// Should have version in title
	if !strings.Contains(result, "# test.package (Version v1.0.0)") {
		t.Error("Expected version in title")
	}

	// Should still have syntax
	if !strings.Contains(result, "**Syntax:** `proto3`") {
		t.Error("Expected syntax to be preserved")
	}

	// Should not have duplicate title
	lines := strings.Split(result, "\n")
	titleCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "# test.package") {
			titleCount++
		}
	}
	if titleCount > 1 {
		t.Error("Expected only one title")
	}
}

func TestMarkdownExporter_ExportWithVersion_EmptyDoc(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
	}

	result := exporter.ExportWithVersion(doc, "v2.0.0")

	if !strings.Contains(result, "# test.package (Version v2.0.0)") {
		t.Error("Expected version in title")
	}
}

func TestMarkdownExporter_Export_NoServices(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{Name: "Message"},
		},
	}

	result := exporter.Export(doc)

	// Should not have Services link in TOC
	if strings.Contains(result, "- [Services](#services)") {
		t.Error("Should not have Services in TOC when no services")
	}

	// Should not have Services section
	if strings.Contains(result, "## Services") {
		t.Error("Should not have Services section when no services")
	}
}

func TestMarkdownExporter_Export_NoMessages(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{Name: "Service"},
		},
	}

	result := exporter.Export(doc)

	// Should not have Messages link in TOC
	if strings.Contains(result, "- [Messages](#messages)") {
		t.Error("Should not have Messages in TOC when no messages")
	}

	// Should not have Messages section
	if strings.Contains(result, "## Messages") {
		t.Error("Should not have Messages section when no messages")
	}
}

func TestMarkdownExporter_Export_NoEnums(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{Name: "Message"},
		},
	}

	result := exporter.Export(doc)

	// Should not have Enums link in TOC
	if strings.Contains(result, "- [Enums](#enums)") {
		t.Error("Should not have Enums in TOC when no enums")
	}

	// Should not have Enums section
	if strings.Contains(result, "## Enums") {
		t.Error("Should not have Enums section when no enums")
	}
}

func TestMarkdownExporter_Export_MessageWithNoFields(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:   "EmptyMessage",
				Fields: []*FieldDoc{},
			},
		},
	}

	result := exporter.Export(doc)

	// Should have message name
	if !strings.Contains(result, "### EmptyMessage") {
		t.Error("Expected message header")
	}

	// Should not have field table when no fields
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.Contains(line, "### EmptyMessage") {
			// Check next few lines don't contain field table
			for j := i + 1; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "| Field | Type |") {
					t.Error("Should not have field table for message with no fields")
				}
			}
			break
		}
	}
}

func TestMarkdownExporter_Export_EnumWithNoValues(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Enums: []*EnumDoc{
			{
				Name:   "EmptyEnum",
				Values: []*EnumValueDoc{},
			},
		},
	}

	result := exporter.Export(doc)

	// Should have enum name
	if !strings.Contains(result, "### EmptyEnum") {
		t.Error("Expected enum header")
	}

	// Should not have value table when no values
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.Contains(line, "### EmptyEnum") {
			// Check next few lines don't contain value table
			for j := i + 1; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "| Name | Number |") {
					t.Error("Should not have value table for enum with no values")
				}
			}
			break
		}
	}
}

func TestMarkdownExporter_Export_ServiceWithNoMethods(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name:    "EmptyService",
				Methods: []*MethodDoc{},
			},
		},
	}

	result := exporter.Export(doc)

	// Should have service name
	if !strings.Contains(result, "### EmptyService") {
		t.Error("Expected service header")
	}

	// Should not have Methods section when no methods
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.Contains(line, "### EmptyService") {
			// Check next few lines don't contain Methods section
			for j := i + 1; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "#### Methods") {
					t.Error("Should not have Methods section for service with no methods")
				}
			}
			break
		}
	}
}

func TestMarkdownExporter_Export_FieldWithEmptyDescription(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name: "Message",
				Fields: []*FieldDoc{
					{
						Name:        "field",
						Type:        "string",
						Description: "", // Empty description
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Should still generate table row
	if !strings.Contains(result, "| field | `string` |") {
		t.Error("Expected field in table even with empty description")
	}
}

func TestMarkdownExporter_Export_MethodWithoutHTTP(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "Service",
				Methods: []*MethodDoc{
					{
						Name:         "Method",
						RequestType:  "Request",
						ResponseType: "Response",
						HTTPMethod:   "",
						HTTPPath:     "",
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Should not have HTTP annotation
	if strings.Contains(result, "**HTTP:**") {
		t.Error("Should not have HTTP annotation when method/path are empty")
	}
}

func TestMarkdownExporter_Export_MethodWithPartialHTTP(t *testing.T) {
	exporter := NewMarkdownExporter()

	// Test with HTTPMethod but no HTTPPath
	doc1 := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "Service",
				Methods: []*MethodDoc{
					{
						Name:         "Method",
						RequestType:  "Request",
						ResponseType: "Response",
						HTTPMethod:   "GET",
						HTTPPath:     "",
					},
				},
			},
		},
	}

	result1 := exporter.Export(doc1)
	if strings.Contains(result1, "**HTTP:**") {
		t.Error("Should not have HTTP annotation when HTTPPath is empty")
	}

	// Test with HTTPPath but no HTTPMethod
	doc2 := &Documentation{
		PackageName: "test.package",
		Services: []*ServiceDoc{
			{
				Name: "Service",
				Methods: []*MethodDoc{
					{
						Name:         "Method",
						RequestType:  "Request",
						ResponseType: "Response",
						HTTPMethod:   "",
						HTTPPath:     "/v1/resource",
					},
				},
			},
		},
	}

	result2 := exporter.Export(doc2)
	if strings.Contains(result2, "**HTTP:**") {
		t.Error("Should not have HTTP annotation when HTTPMethod is empty")
	}
}

func TestMarkdownExporter_Export_MultipleDeprecatedElements(t *testing.T) {
	exporter := NewMarkdownExporter()
	doc := &Documentation{
		PackageName: "test.package",
		Messages: []*MessageDoc{
			{
				Name:       "OldMessage",
				Deprecated: true,
				Fields: []*FieldDoc{
					{
						Name:       "old_field",
						Type:       "string",
						Deprecated: true,
					},
				},
			},
		},
		Enums: []*EnumDoc{
			{
				Name:       "OldEnum",
				Deprecated: true,
				Values: []*EnumValueDoc{
					{
						Name:       "OLD_VALUE",
						Number:     0,
						Deprecated: true,
					},
				},
			},
		},
		Services: []*ServiceDoc{
			{
				Name:       "OldService",
				Deprecated: true,
				Methods: []*MethodDoc{
					{
						Name:         "OldMethod",
						RequestType:  "Request",
						ResponseType: "Response",
						Deprecated:   true,
					},
				},
			},
		},
	}

	result := exporter.Export(doc)

	// Count deprecated warnings - should have at least 6
	deprecatedCount := strings.Count(result, "⚠️")
	if deprecatedCount < 6 {
		t.Errorf("Expected at least 6 deprecated warnings, got %d", deprecatedCount)
	}
}
