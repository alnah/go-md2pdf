# go-md2pdf

[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/alnah/go-md2pdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/alnah/go-md2pdf)](https://goreportcard.com/report/github.com/alnah/go-md2pdf)
[![Build Status](https://img.shields.io/github/actions/workflow/status/alnah/go-md2pdf/ci.yml?branch=main)](https://github.com/alnah/go-md2pdf/actions)
[![Coverage](https://codecov.io/gh/alnah/go-md2pdf/branch/main/graph/badge.svg)](https://codecov.io/gh/alnah/go-md2pdf)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.txt)

> Convert Markdown to print-ready PDF in **3 lines of code** - with cover pages, table of contents, signatures, footers, watermarks, and custom styling.

## Installation

```bash
go install github.com/alnah/go-md2pdf/cmd/md2pdf@latest
```

<details>
<summary>Other installation methods</summary>

### Docker

```bash
docker pull ghcr.io/alnah/go-md2pdf:latest
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/alnah/go-md2pdf/releases).

</details>

## Features

- **CLI + Library** - Use as `md2pdf` command or import in Go
- **Batch conversion** - Process directories with parallel workers
- **Cover pages** - Title, subtitle, logo, author, organization, date, version
- **Table of contents** - Auto-generated from headings with configurable depth
- **Custom styling** - Embedded themes or your own CSS
- **Page settings** - Size (letter, A4, legal), orientation, margins
- **Signatures** - Name, title, email, photo, links
- **Footers** - Page numbers, dates, status text
- **Watermarks** - Diagonal background text (BRAND, etc.)

## Quick Start

### CLI

```bash
md2pdf document.md                        # Single file
md2pdf ./docs/ -o ./output/               # Batch convert
md2pdf --config work document.md          # With config
```

### Library

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/alnah/go-md2pdf"
)

func main() {
    svc := md2pdf.New()
    defer svc.Close()

    pdf, err := svc.Convert(context.Background(), md2pdf.Input{
        Markdown: "# Hello World\n\nGenerated with go-md2pdf.",
    })
    if err != nil {
        log.Fatal(err)
    }

    os.WriteFile("output.pdf", pdf, 0644)
}
```

## Library Usage

### With Cover Page

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Cover: &md2pdf.Cover{
        Title:        "Project Report",
        Subtitle:     "Q4 2025 Analysis",
        Author:       "John Doe",
        AuthorTitle:  "Senior Analyst",
        Organization: "Acme Corp",
        Date:         "2025-12-15",
        Version:      "v1.0",
        Logo:         "/path/to/logo.png", // or URL
    },
})
```

### With Table of Contents

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    TOC: &md2pdf.TOC{
        Title:    "Contents",
        MaxDepth: 3, // Include h1, h2, h3
    },
})
```

### With Footer

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Footer: &md2pdf.Footer{
        ShowPageNumber: true,
        Position:       "center",
        Date:           "2025-12-15",
        Status:         "DRAFT",
    },
})
```

### With Signature

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Signature: &md2pdf.Signature{
        Name:  "John Doe",
        Title: "Senior Developer",
        Email: "john@example.com",
    },
})
```

### With Watermark

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Watermark: &md2pdf.Watermark{
        Text:    "CONFIDENTIAL",
        Color:   "#888888",
        Opacity: 0.1,
        Angle:   -45,
    },
})
```

### With Custom CSS

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    CSS:      customCSS,
})
```

### With Page Settings

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Page: &md2pdf.PageSettings{
        Size:        md2pdf.PageSizeA4,
        Orientation: md2pdf.OrientationLandscape,
        Margin:      1.0, // inches
    },
})
```

### With Page Breaks

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    PageBreaks: &md2pdf.PageBreaks{
        BeforeH1: true, // Page break before H1 headings
        BeforeH2: true, // Page break before H2 headings
        Orphans:  3,    // Min 3 lines at page bottom
        Widows:   3,    // Min 3 lines at page top
    },
})
```

## CLI Reference

```
md2pdf [flags] <input>

Flags:
  -o, --output            Output file or directory
  -c, --config            Config name (work, personal) or path
  -w, --workers           Number of parallel workers (default: auto)
  -p, --page-size         Page size: letter, a4, legal (default: letter)
      --orientation       Page orientation: portrait, landscape (default: portrait)
      --margin            Page margin in inches (default: 0.5)
      --css               Custom CSS file
      --no-style          Disable default styling
      --no-footer         Disable footer
      --no-signature      Disable signature
      --no-cover          Disable cover page
      --no-toc            Disable table of contents
      --no-watermark      Disable watermark
      --no-page-breaks    Disable page break features
      --cover-title       Override cover page title
      --break-before      Page breaks before headings: h1,h2,h3 (comma-separated)
      --orphans           Min lines at page bottom (1-5, default: 2)
      --widows            Min lines at page top (1-5, default: 2)
      --watermark-text    Watermark text (e.g., BRAND)
      --watermark-color   Watermark color in hex (default: #888888)
      --watermark-opacity Watermark opacity 0.0-1.0 (default: 0.1)
      --watermark-angle   Watermark rotation in degrees (default: -45)
  -q, --quiet             Only show errors
  -v, --verbose           Show detailed timing
      --version           Show version and exit
```

### Examples

```bash
# Single file with custom output
md2pdf -o report.pdf input.md

# Batch with config
md2pdf --config work ./docs/ -o ./pdfs/

# Custom CSS, no footer
md2pdf --css custom.css --no-footer document.md

# A4 landscape with 1-inch margins
md2pdf -p a4 --orientation landscape --margin 1.0 document.md

# With watermark
md2pdf --watermark-text "BRAND" --watermark-opacity 0.15 document.md

# Override cover title
md2pdf --cover-title "Final Report" document.md

# Page breaks before H1 and H2 headings
md2pdf --break-before h1,h2 document.md
```

### Docker

```bash
# Convert a single file
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf document.md

# Convert with output path
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf -o output.pdf input.md

# Batch convert directory
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf ./docs/ -o ./pdfs/
```

## Configuration

Config files are loaded from `~/.config/go-md2pdf/` or current directory.
Supported formats: `.yaml`, `.yml`

| Option                  | Type   | Default      | Description                                    |
| ----------------------- | ------ | ------------ | ---------------------------------------------- |
| `input.defaultDir`      | string | -            | Default input directory                        |
| `output.defaultDir`     | string | -            | Default output directory                       |
| `css.style`             | string | -            | Embedded style name                            |
| `page.size`             | string | `"letter"`   | letter, a4, legal                              |
| `page.orientation`      | string | `"portrait"` | portrait, landscape                            |
| `page.margin`           | float  | `0.5`        | Margin in inches (0.25-3.0)                    |
| `cover.enabled`         | bool   | `false`      | Show cover page                                |
| `cover.title`           | string | -            | Cover title (auto: H1 or filename)             |
| `cover.subtitle`        | string | -            | Cover subtitle                                 |
| `cover.logo`            | string | -            | Logo path or URL                               |
| `cover.author`          | string | -            | Author name (fallback: signature.name)         |
| `cover.authorTitle`     | string | -            | Author title (fallback: signature.title)       |
| `cover.organization`    | string | -            | Organization name                              |
| `cover.date`            | string | -            | Date ("auto" for today, fallback: footer.date) |
| `cover.version`         | string | -            | Version (fallback: footer.status)              |
| `toc.enabled`           | bool   | `false`      | Show table of contents                         |
| `toc.title`             | string | -            | TOC title (empty = no title)                   |
| `toc.maxDepth`          | int    | `3`          | Heading depth (1-6)                            |
| `footer.enabled`        | bool   | `false`      | Show footer                                    |
| `footer.showPageNumber` | bool   | `false`      | Show page numbers                              |
| `footer.position`       | string | `"right"`    | left, center, right                            |
| `footer.date`           | string | -            | Date text                                      |
| `footer.status`         | string | -            | Status text (DRAFT, etc)                       |
| `footer.text`           | string | -            | Custom footer text                             |
| `signature.enabled`     | bool   | `false`      | Show signature block                           |
| `signature.name`        | string | -            | Signer name                                    |
| `signature.title`       | string | -            | Signer title                                   |
| `signature.email`       | string | -            | Signer email                                   |
| `signature.imagePath`   | string | -            | Photo path or URL                              |
| `signature.links`       | array  | -            | Links (label, url)                             |
| `watermark.enabled`     | bool   | `false`      | Show watermark                                 |
| `watermark.text`        | string | -            | Watermark text (required if enabled)           |
| `watermark.color`       | string | `"#888888"`  | Watermark color (hex)                          |
| `watermark.opacity`     | float  | `0.1`        | Watermark opacity (0.0-1.0)                    |
| `watermark.angle`       | float  | `-45`        | Watermark rotation (degrees)                   |
| `pageBreaks.enabled`    | bool   | `false`      | Enable page break features                     |
| `pageBreaks.beforeH1`   | bool   | `false`      | Page break before H1 headings                  |
| `pageBreaks.beforeH2`   | bool   | `false`      | Page break before H2 headings                  |
| `pageBreaks.beforeH3`   | bool   | `false`      | Page break before H3 headings                  |
| `pageBreaks.orphans`    | int    | `2`          | Min lines at page bottom (1-5)                 |
| `pageBreaks.widows`     | int    | `2`          | Min lines at page top (1-5)                    |

<details>
<summary>Example config file</summary>

```yaml
# ~/.config/go-md2pdf/work.yaml

css:
  style: 'nord' # add your styles to internal/assets/styles/, and build or install

page:
  size: 'a4'
  orientation: 'portrait'
  margin: 0.75 # inches (0.5in = 12.7mm, 1in = 25.4mm)

cover:
  enabled: true
  # title: auto-detected from H1 or filename
  subtitle: 'Internal Document'
  logo: '/path/to/company-logo.png'
  organization: 'Acme Corp'
  date: 'auto' # resolves to YYYY-MM-DD

toc:
  enabled: true
  title: 'Table of Contents'
  maxDepth: 3

footer:
  enabled: true
  showPageNumber: true
  position: 'center'
  status: 'DRAFT'

signature:
  enabled: true
  name: 'John Doe'
  title: 'Developer'
  email: 'john@example.com'
  links:
    - label: 'GitHub'
      url: 'https://github.com/johndoe'

watermark:
  enabled: true
  text: 'DRAFT'
  color: '#888888'
  opacity: 0.1
  angle: -45

pageBreaks:
  enabled: true
  beforeH1: true
  beforeH2: false
  beforeH3: false
  orphans: 2
  widows: 2
```

</details>

## Project Structure

```
go-md2pdf/
├── service.go          # Public API: New(), Convert(), Close()
├── pool.go             # ServicePool for parallel processing
├── types.go            # Input, Footer, Signature, Watermark, Cover, TOC, PageBreaks
├── mdtransform.go      # Markdown preprocessing
├── md2html.go          # Markdown to HTML (Goldmark)
├── htmlinject.go       # CSS/signature/cover/TOC injection
├── html2pdf.go         # HTML to PDF (headless Chrome)
├── cmd/md2pdf/         # CLI binary
└── internal/           # Assets, config, utilities
```

## Documentation

Full API documentation: [pkg.go.dev/github.com/alnah/go-md2pdf](https://pkg.go.dev/github.com/alnah/go-md2pdf)

## Requirements

- Go 1.25+
- Chrome/Chromium (downloaded automatically on first run)

## Contributing

See: [CONTRIBUTING.md](CONTRIBUTING.md).

## License

See: [MIT](LICENSE.txt).
