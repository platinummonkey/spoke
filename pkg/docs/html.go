package docs

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// HTMLExporter exports documentation to HTML format
type HTMLExporter struct {
	template *template.Template
}

// NewHTMLExporter creates a new HTML exporter
func NewHTMLExporter() *HTMLExporter {
	tmpl := template.Must(template.New("docs").Funcs(template.FuncMap{
		"escape":     template.HTMLEscapeString,
		"markdown":   markdownToHTML,
		"anchor":     toAnchor,
		"hasContent": hasContent,
	}).Parse(htmlTemplate))

	return &HTMLExporter{
		template: tmpl,
	}
}

// Export exports documentation to HTML
func (e *HTMLExporter) Export(doc *Documentation) (string, error) {
	return e.ExportWithVersion(doc, "")
}

// ExportWithVersion exports documentation with version info
func (e *HTMLExporter) ExportWithVersion(doc *Documentation, version string) (string, error) {
	data := struct {
		*Documentation
		Version string
	}{
		Documentation: doc,
		Version:       version,
	}

	var buf bytes.Buffer
	err := e.template.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// toAnchor converts a name to an HTML anchor
func toAnchor(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// hasContent checks if a string has content
func hasContent(s string) bool {
	return strings.TrimSpace(s) != ""
}

// markdownToHTML converts simple markdown to HTML
func markdownToHTML(text string) template.HTML {
	if text == "" {
		return ""
	}

	// Simple markdown conversion
	text = template.HTMLEscapeString(text)

	// Bold
	text = strings.ReplaceAll(text, "**", "<strong>")

	// Code blocks
	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				result = append(result, "</code></pre>")
			} else {
				result = append(result, "<pre><code>")
			}
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			result = append(result, line)
		} else {
			// Inline code
			line = strings.ReplaceAll(line, "`", "<code>")
			result = append(result, "<p>"+line+"</p>")
		}
	}

	return template.HTML(strings.Join(result, "\n"))
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .PackageName }} - API Documentation</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            background: #2c3e50;
            color: white;
            padding: 30px 0;
            margin-bottom: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        header h1 {
            margin-bottom: 10px;
        }
        .badge {
            display: inline-block;
            padding: 4px 8px;
            background: #3498db;
            color: white;
            border-radius: 3px;
            font-size: 0.85em;
            font-weight: 600;
        }
        nav {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        nav h2 {
            margin-bottom: 15px;
            color: #2c3e50;
        }
        nav ul {
            list-style: none;
        }
        nav ul li {
            margin: 8px 0;
        }
        nav a {
            color: #3498db;
            text-decoration: none;
            transition: color 0.2s;
        }
        nav a:hover {
            color: #2980b9;
            text-decoration: underline;
        }
        .content {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h2 {
            color: #2c3e50;
            margin: 30px 0 20px 0;
            padding-bottom: 10px;
            border-bottom: 2px solid #3498db;
        }
        h3 {
            color: #34495e;
            margin: 25px 0 15px 0;
        }
        h4 {
            color: #7f8c8d;
            margin: 20px 0 10px 0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th {
            background: #ecf0f1;
            padding: 12px;
            text-align: left;
            font-weight: 600;
            border-bottom: 2px solid #bdc3c7;
        }
        td {
            padding: 12px;
            border-bottom: 1px solid #ecf0f1;
        }
        tr:hover {
            background: #f8f9fa;
        }
        code {
            background: #f8f9fa;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
            font-size: 0.9em;
            color: #e74c3c;
        }
        pre {
            background: #2c3e50;
            color: #ecf0f1;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            margin: 15px 0;
        }
        pre code {
            background: none;
            color: #ecf0f1;
            padding: 0;
        }
        .deprecated {
            color: #e74c3c;
            font-weight: 600;
        }
        .method-signature {
            background: #ecf0f1;
            padding: 15px;
            border-left: 4px solid #3498db;
            margin: 15px 0;
            font-family: monospace;
        }
        .streaming {
            color: #9b59b6;
            font-size: 0.85em;
            font-style: italic;
        }
        .label {
            display: inline-block;
            padding: 2px 8px;
            background: #95a5a6;
            color: white;
            border-radius: 3px;
            font-size: 0.8em;
            margin-left: 5px;
        }
        .label.repeated {
            background: #3498db;
        }
        .label.optional {
            background: #f39c12;
        }
        .label.required {
            background: #e74c3c;
        }
        .search-box {
            width: 100%;
            padding: 12px;
            border: 2px solid #ecf0f1;
            border-radius: 5px;
            font-size: 1em;
            margin-bottom: 20px;
        }
        .search-box:focus {
            outline: none;
            border-color: #3498db;
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>{{ .PackageName }}</h1>
            {{ if .Syntax }}
            <span class="badge">{{ .Syntax }}</span>
            {{ end }}
            {{ if .Version }}
            <span class="badge">Version {{ .Version }}</span>
            {{ end }}
        </div>
    </header>

    <div class="container">
        <nav>
            <h2>Table of Contents</h2>
            <input type="text" class="search-box" placeholder="Search documentation..." id="search">
            <ul>
                {{ if .Services }}
                <li><a href="#services">Services</a></li>
                {{ end }}
                {{ if .Messages }}
                <li><a href="#messages">Messages</a></li>
                {{ end }}
                {{ if .Enums }}
                <li><a href="#enums">Enums</a></li>
                {{ end }}
            </ul>
        </nav>

        <div class="content">
            {{ if hasContent .Description }}
            <p>{{ .Description }}</p>
            {{ end }}

            {{ if .Services }}
            <h2 id="services">Services</h2>
            {{ range .Services }}
            <h3 id="{{ anchor .Name }}">{{ .Name }}</h3>
            {{ if .Deprecated }}<p class="deprecated">⚠️ Deprecated</p>{{ end }}
            {{ if hasContent .Description }}<p>{{ .Description }}</p>{{ end }}

            {{ if .Methods }}
            <h4>Methods</h4>
            {{ range .Methods }}
            <h4 id="{{ anchor .Name }}">{{ .Name }}</h4>
            {{ if .Deprecated }}<p class="deprecated">⚠️ Deprecated</p>{{ end }}
            {{ if hasContent .Description }}<p>{{ .Description }}</p>{{ end }}

            <div class="method-signature">
                rpc {{ .Name }} ({{ .RequestType }}) returns ({{ .ResponseType }})
                {{ if .ClientStreaming }}{{ if .ServerStreaming }}
                <span class="streaming">(bidirectional streaming)</span>
                {{ else }}
                <span class="streaming">(client streaming)</span>
                {{ end }}{{ else }}{{ if .ServerStreaming }}
                <span class="streaming">(server streaming)</span>
                {{ end }}{{ end }}
            </div>
            {{ end }}
            {{ end }}
            {{ end }}
            {{ end }}

            {{ if .Messages }}
            <h2 id="messages">Messages</h2>
            {{ range .Messages }}
            {{ template "message" . }}
            {{ end }}
            {{ end }}

            {{ if .Enums }}
            <h2 id="enums">Enums</h2>
            {{ range .Enums }}
            {{ template "enum" . }}
            {{ end }}
            {{ end }}
        </div>
    </div>

    <script>
        // Simple search functionality
        document.getElementById('search').addEventListener('input', function(e) {
            const searchTerm = e.target.value.toLowerCase();
            const content = document.querySelector('.content');
            const sections = content.querySelectorAll('h3, h4');

            sections.forEach(section => {
                const text = section.textContent.toLowerCase();
                const parent = section.parentElement;

                if (text.includes(searchTerm) || searchTerm === '') {
                    parent.style.display = '';
                } else {
                    parent.style.display = 'none';
                }
            });
        });
    </script>
</body>
</html>

{{ define "message" }}
<h3 id="{{ anchor .Name }}">{{ .Name }}</h3>
{{ if .Deprecated }}<p class="deprecated">⚠️ Deprecated</p>{{ end }}
{{ if hasContent .Description }}<p>{{ .Description }}</p>{{ end }}

{{ if .Fields }}
<table>
    <thead>
        <tr>
            <th>Field</th>
            <th>Type</th>
            <th>Number</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        {{ range .Fields }}
        <tr>
            <td>
                <code>{{ .Name }}</code>
                {{ if .Repeated }}<span class="label repeated">repeated</span>{{ end }}
                {{ if .Optional }}<span class="label optional">optional</span>{{ end }}
                {{ if .Required }}<span class="label required">required</span>{{ end }}
            </td>
            <td><code>{{ .Type }}</code></td>
            <td>{{ .Number }}</td>
            <td>
                {{ if .Deprecated }}<span class="deprecated">⚠️ Deprecated</span> {{ end }}
                {{ .Description }}
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>
{{ end }}

{{ if .Enums }}
{{ range .Enums }}
{{ template "enum" . }}
{{ end }}
{{ end }}

{{ if .NestedTypes }}
{{ range .NestedTypes }}
{{ template "message" . }}
{{ end }}
{{ end }}
{{ end }}

{{ define "enum" }}
<h3 id="{{ anchor .Name }}">{{ .Name }}</h3>
{{ if .Deprecated }}<p class="deprecated">⚠️ Deprecated</p>{{ end }}
{{ if hasContent .Description }}<p>{{ .Description }}</p>{{ end }}

{{ if .Values }}
<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Number</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        {{ range .Values }}
        <tr>
            <td><code>{{ .Name }}</code></td>
            <td>{{ .Number }}</td>
            <td>
                {{ if .Deprecated }}<span class="deprecated">⚠️ Deprecated</span> {{ end }}
                {{ .Description }}
            </td>
        </tr>
        {{ end }}
    </tbody>
</table>
{{ end }}
{{ end }}
`
