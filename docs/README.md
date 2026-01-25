# Spoke Documentation

This directory contains the Hugo-based documentation for Spoke.

## Local Development

### Prerequisites

- Hugo 0.121.0 or later ([installation guide](https://gohugo.io/installation/))
- Git

### Install Hugo

**macOS:**
```bash
brew install hugo
```

**Linux:**
```bash
# Download and install Hugo extended
wget https://github.com/gohugoio/hugo/releases/download/v0.121.0/hugo_extended_0.121.0_linux-amd64.deb
sudo dpkg -i hugo_extended_0.121.0_linux-amd64.deb
```

**Windows:**
```bash
choco install hugo-extended
```

### Install Theme

```bash
cd docs
git clone https://github.com/alex-shpak/hugo-book themes/book
```

### Run Development Server

```bash
cd docs
hugo server -D
```

Open http://localhost:1313 in your browser.

The server will automatically reload when you make changes to the content.

## Building for Production

Build the static site:

```bash
cd docs
hugo --minify
```

The built site will be in `docs/public/`.

## Project Structure

```
docs/
├── hugo.toml                 # Hugo configuration
├── content/                  # Markdown content
│   ├── _index.md            # Home page
│   ├── getting-started/     # Getting started guides
│   ├── guides/              # User guides
│   ├── tutorials/           # Step-by-step tutorials
│   ├── examples/            # Code examples
│   ├── architecture/        # Architecture documentation
│   └── deployment/          # Deployment guides
├── static/                  # Static files (images, etc.)
├── themes/                  # Hugo themes
│   └── book/               # Hugo Book theme
└── public/                 # Generated site (git-ignored)
```

## Content Organization

### Front Matter

All content files should have front matter:

```yaml
---
title: "Page Title"
weight: 1                    # Order in navigation
bookFlatSection: false       # Show subsections
bookCollapseSection: false   # Collapse by default
---
```

### Navigation

Navigation is automatically generated from:
1. Directory structure
2. `weight` values in front matter (lower = higher in menu)
3. `title` values

## Writing Documentation

### Markdown Features

The Hugo Book theme supports:

**Code blocks with syntax highlighting:**

````markdown
```go
func main() {
    fmt.Println("Hello, World!")
}
```
````

**Admonitions:**

```markdown
{{< hint info >}}
This is an info hint.
{{< /hint >}}

{{< hint warning >}}
This is a warning.
{{< /hint >}}

{{< hint danger >}}
This is dangerous!
{{< /hint >}}
```

**Buttons:**

```markdown
{{< button href="https://example.com" >}}Click me{{< /button >}}
```

**Columns:**

```markdown
{{< columns >}}
Left column content
<--->
Right column content
{{< /columns >}}
```

**Tabs:**

```markdown
{{< tabs "uniqueid" >}}
{{< tab "Tab 1" >}}
Content for tab 1
{{< /tab >}}
{{< tab "Tab 2" >}}
Content for tab 2
{{< /tab >}}
{{< /tabs >}}
```

### Best Practices

1. **Use descriptive titles**: Clear, concise titles help users find content
2. **Add front matter**: Always include title and weight
3. **Link between pages**: Use relative links `[text](/section/page/)`
4. **Include code examples**: Show, don't just tell
5. **Use proper headings**: Start with `##` (not `#` which is the title)
6. **Test locally**: Always test with `hugo server` before committing

## Deployment

Documentation is automatically deployed to GitHub Pages when changes are pushed to `main` branch.

### GitHub Pages Setup

1. Go to repository Settings → Pages
2. Source: GitHub Actions
3. The workflow in `.github/workflows/hugo.yml` handles the build and deployment

### Manual Deployment

If you need to deploy manually:

```bash
cd docs
hugo --minify

# The site is in public/ directory
# Deploy public/ to your hosting provider
```

## Contributing

When adding new documentation:

1. Create content in the appropriate section
2. Add front matter with title and weight
3. Test locally with `hugo server`
4. Commit and push to main branch
5. Verify deployment at https://platinummonkey.github.io/spoke/

## Troubleshooting

### Theme Not Found

```bash
cd docs
git clone https://github.com/alex-shpak/hugo-book themes/book
```

### Hugo Version Mismatch

Ensure you're using Hugo Extended version 0.121.0 or later:

```bash
hugo version
```

### Build Errors

Check for:
- Missing front matter in content files
- Invalid markdown syntax
- Broken links

Run with verbose output:

```bash
hugo server --verbose
```

## Resources

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Hugo Book Theme](https://github.com/alex-shpak/hugo-book)
- [Markdown Guide](https://www.markdownguide.org/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
