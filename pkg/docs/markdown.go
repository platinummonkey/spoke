package docs

import (
	"fmt"
	"strings"
)

// MarkdownExporter exports documentation to Markdown format
type MarkdownExporter struct{}

// NewMarkdownExporter creates a new Markdown exporter
func NewMarkdownExporter() *MarkdownExporter {
	return &MarkdownExporter{}
}

// Export exports documentation to Markdown
func (e *MarkdownExporter) Export(doc *Documentation) string {
	var b strings.Builder

	// Title
	b.WriteString(fmt.Sprintf("# %s\n\n", doc.PackageName))

	// Package info
	if doc.Syntax != "" {
		b.WriteString(fmt.Sprintf("**Syntax:** `%s`\n\n", doc.Syntax))
	}

	if doc.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", doc.Description))
	}

	// Table of contents
	b.WriteString("## Table of Contents\n\n")
	if len(doc.Services) > 0 {
		b.WriteString("- [Services](#services)\n")
	}
	if len(doc.Messages) > 0 {
		b.WriteString("- [Messages](#messages)\n")
	}
	if len(doc.Enums) > 0 {
		b.WriteString("- [Enums](#enums)\n")
	}
	b.WriteString("\n")

	// Services
	if len(doc.Services) > 0 {
		b.WriteString("## Services\n\n")
		for _, svc := range doc.Services {
			e.writeService(&b, svc)
		}
	}

	// Messages
	if len(doc.Messages) > 0 {
		b.WriteString("## Messages\n\n")
		for _, msg := range doc.Messages {
			e.writeMessage(&b, msg, 0)
		}
	}

	// Enums
	if len(doc.Enums) > 0 {
		b.WriteString("## Enums\n\n")
		for _, enum := range doc.Enums {
			e.writeEnum(&b, enum)
		}
	}

	return b.String()
}

// writeService writes a service to markdown
func (e *MarkdownExporter) writeService(b *strings.Builder, svc *ServiceDoc) {
	b.WriteString(fmt.Sprintf("### %s\n\n", svc.Name))

	if svc.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", svc.Description))
	}

	if svc.Deprecated {
		b.WriteString("**⚠️ Deprecated**\n\n")
	}

	// Methods
	if len(svc.Methods) > 0 {
		b.WriteString("#### Methods\n\n")
		for _, method := range svc.Methods {
			e.writeMethod(b, method)
		}
	}
}

// writeMethod writes a method to markdown
func (e *MarkdownExporter) writeMethod(b *strings.Builder, method *MethodDoc) {
	// Method signature
	streaming := ""
	if method.ClientStreaming && method.ServerStreaming {
		streaming = " (bidirectional streaming)"
	} else if method.ClientStreaming {
		streaming = " (client streaming)"
	} else if method.ServerStreaming {
		streaming = " (server streaming)"
	}

	b.WriteString(fmt.Sprintf("##### `%s`%s\n\n", method.Name, streaming))

	if method.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", method.Description))
	}

	if method.Deprecated {
		b.WriteString("**⚠️ Deprecated**\n\n")
	}

	// Request/Response
	b.WriteString("```protobuf\n")
	b.WriteString(fmt.Sprintf("rpc %s (%s) returns (%s)\n",
		method.Name, method.RequestType, method.ResponseType))
	b.WriteString("```\n\n")

	if method.HTTPMethod != "" && method.HTTPPath != "" {
		b.WriteString(fmt.Sprintf("**HTTP:** `%s %s`\n\n", method.HTTPMethod, method.HTTPPath))
	}
}

// writeMessage writes a message to markdown
func (e *MarkdownExporter) writeMessage(b *strings.Builder, msg *MessageDoc, depth int) {
	// Message header
	prefix := strings.Repeat("#", 3+depth)
	b.WriteString(fmt.Sprintf("%s %s\n\n", prefix, msg.Name))

	if msg.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", msg.Description))
	}

	if msg.Deprecated {
		b.WriteString("**⚠️ Deprecated**\n\n")
	}

	// Fields table
	if len(msg.Fields) > 0 {
		b.WriteString("| Field | Type | Label | Description |\n")
		b.WriteString("|-------|------|-------|-------------|\n")

		for _, field := range msg.Fields {
			desc := field.Description
			if field.Deprecated {
				desc = "⚠️ **Deprecated** " + desc
			}
			if field.OneofName != "" {
				desc = fmt.Sprintf("(oneof %s) %s", field.OneofName, desc)
			}

			label := field.Label
			if label == "" {
				label = "-"
			}

			b.WriteString(fmt.Sprintf("| %s | `%s` | %s | %s |\n",
				field.Name, field.Type, label, desc))
		}
		b.WriteString("\n")
	}

	// Nested enums
	for _, enum := range msg.Enums {
		e.writeEnum(b, enum)
	}

	// Nested messages
	for _, nested := range msg.NestedTypes {
		e.writeMessage(b, nested, depth+1)
	}
}

// writeEnum writes an enum to markdown
func (e *MarkdownExporter) writeEnum(b *strings.Builder, enum *EnumDoc) {
	b.WriteString(fmt.Sprintf("### %s\n\n", enum.Name))

	if enum.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", enum.Description))
	}

	if enum.Deprecated {
		b.WriteString("**⚠️ Deprecated**\n\n")
	}

	// Values table
	if len(enum.Values) > 0 {
		b.WriteString("| Name | Number | Description |\n")
		b.WriteString("|------|--------|-------------|\n")

		for _, value := range enum.Values {
			desc := value.Description
			if value.Deprecated {
				desc = "⚠️ **Deprecated** " + desc
			}

			b.WriteString(fmt.Sprintf("| %s | %d | %s |\n",
				value.Name, value.Number, desc))
		}
		b.WriteString("\n")
	}
}

// ExportWithVersion exports documentation with version information
func (e *MarkdownExporter) ExportWithVersion(doc *Documentation, version string) string {
	var b strings.Builder

	// Version header
	b.WriteString(fmt.Sprintf("# %s (Version %s)\n\n", doc.PackageName, version))

	// Add rest of documentation
	exported := e.Export(doc)
	// Skip the first line (title) since we already added version header
	lines := strings.Split(exported, "\n")
	if len(lines) > 2 {
		b.WriteString(strings.Join(lines[2:], "\n"))
	}

	return b.String()
}
