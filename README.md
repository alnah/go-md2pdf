# go-md2pdf

[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/alnah/go-md2pdf)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen)](https://goreportcard.com/report/github.com/alnah/go-md2pdf)
[![Build Status](https://img.shields.io/github/actions/workflow/status/alnah/go-md2pdf/ci.yml?branch=main)](https://github.com/alnah/go-md2pdf/actions)
[![Coverage](https://img.shields.io/codecov/c/github/alnah/go-md2pdf)](https://codecov.io/gh/alnah/go-md2pdf)
[![License](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE.txt)

> Markdown to print-ready PDF via CLI or Go library - cover pages, TOC, signatures, footers, watermarks, custom CSS, and parallel batch processing.

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
md2pdf convert document.md                # Single file
md2pdf convert ./docs/ -o ./output/       # Batch convert
md2pdf convert -c work document.md        # With config
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
    svc, err := md2pdf.New()
    if err != nil {
        log.Fatal(err)
    }
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
        ClientName:   "Client Corp",       // extended metadata
        ProjectName:  "Project Alpha",
        DocumentType: "Technical Report",
        DocumentID:   "DOC-2025-001",
    },
})
```

### With Table of Contents

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    TOC: &md2pdf.TOC{
        Title:    "Contents",
        MinDepth: 2, // Start at h2 (skip document title)
        MaxDepth: 3, // Include up to h3
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
        Name:         "John Doe",
        Title:        "Senior Developer",
        Email:        "john@example.com",
        Organization: "Acme Corp",
        Phone:        "+1 555-0123",  // extended metadata
        Department:   "Engineering",
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

### With Custom Assets

Override embedded CSS styles and HTML templates by loading from a custom directory:

```go
import "github.com/alnah/go-md2pdf/internal/assets"

// Create asset resolver with custom directory (falls back to embedded)
loader, err := assets.NewAssetResolver("/path/to/assets")
if err != nil {
    log.Fatal(err)
}

svc, err := md2pdf.New(md2pdf.WithAssetLoader(loader))
if err != nil {
    log.Fatal(err)
}
```

Expected directory structure:

```
/path/to/assets/
├── styles/
│   └── technical.css    # Override embedded style
└── templates/
    ├── cover.html       # Override cover page template
    └── signature.html   # Override signature block template
```

Available embedded styles: `technical`, `creative`, `academic`, `corporate`, `legal`, `invoice`, `manuscript`

Missing files fall back to embedded defaults silently.

### With Service Pool (Parallel Processing)

For batch conversion, use `ServicePool` to process multiple files in parallel:

```go
package main

import (
    "context"
    "log"
    "sync"

    "github.com/alnah/go-md2pdf"
)

func main() {
    // Create pool with 4 workers (each has its own browser instance)
    pool := md2pdf.NewServicePool(4)
    defer pool.Close()

    files := []string{"doc1.md", "doc2.md", "doc3.md", "doc4.md"}
    var wg sync.WaitGroup

    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()

            svc := pool.Acquire()
            if svc == nil {
                log.Printf("failed to acquire service: %v", pool.InitError())
                return
            }
            defer pool.Release(svc)

            content, _ := os.ReadFile(f)
            pdf, err := svc.Convert(context.Background(), md2pdf.Input{
                Markdown: string(content),
            })
            if err != nil {
                log.Printf("convert %s: %v", f, err)
                return
            }
            os.WriteFile(f+".pdf", pdf, 0644)
        }(file)
    }
    wg.Wait()
}
```

Use `md2pdf.ResolvePoolSize(0)` to auto-calculate optimal pool size based on CPU cores.

## CLI Reference

```
md2pdf convert <input> [flags]

Input/Output:
  -o, --output <path>       Output file or directory
  -c, --config <name>       Config file name or path
  -w, --workers <n>         Parallel workers (0 = auto)

Author:
      --author-name <s>     Author name
      --author-title <s>    Author professional title
      --author-email <s>    Author email
      --author-org <s>      Organization name
      --author-phone <s>    Author phone number
      --author-address <s>  Author postal address
      --author-dept <s>     Author department

Document:
      --doc-title <s>       Document title ("" = auto from H1)
      --doc-subtitle <s>    Document subtitle
      --doc-version <s>     Version string
      --doc-date <s>        Date (see Date Formats section)
      --doc-client <s>      Client name
      --doc-project <s>     Project name
      --doc-type <s>        Document type
      --doc-id <s>          Document ID/reference
      --doc-desc <s>        Document description

Page:
  -p, --page-size <s>       Page size: letter, a4, legal
      --orientation <s>     Orientation: portrait, landscape
      --margin <f>          Margin in inches (0.25-3.0)

Footer:
      --footer-position <s> Position: left, center, right
      --footer-text <s>     Custom footer text
      --footer-page-number  Show page numbers
      --footer-doc-id       Show document ID in footer
      --no-footer           Disable footer

Cover:
      --cover-logo <path>   Logo path or URL
      --cover-dept          Show author department on cover
      --no-cover            Disable cover page

Signature:
      --sig-image <path>    Signature image path
      --no-signature        Disable signature block

Table of Contents:
      --toc-title <s>       TOC heading text
      --toc-depth <n>       Max heading depth (1-6)
      --no-toc              Disable table of contents

Watermark:
      --wm-text <s>         Watermark text
      --wm-color <s>        Watermark color (hex)
      --wm-opacity <f>      Watermark opacity (0.0-1.0)
      --wm-angle <f>        Watermark angle in degrees
      --no-watermark        Disable watermark

Page Breaks:
      --break-before <s>    Break before headings: h1,h2,h3
      --orphans <n>         Min lines at page bottom (1-5)
      --widows <n>          Min lines at page top (1-5)
      --no-page-breaks      Disable page break features

Styling:
      --css <path>          External CSS file
      --no-style            Disable CSS styling

Output Control:
  -q, --quiet               Only show errors
  -v, --verbose             Show detailed timing
```

### Examples

```bash
# Single file with custom output
md2pdf convert -o report.pdf input.md

# Batch with config
md2pdf convert -c work ./docs/ -o ./pdfs/

# Custom CSS, no footer
md2pdf convert --css custom.css --no-footer document.md

# A4 landscape with 1-inch margins
md2pdf convert -p a4 --orientation landscape --margin 1.0 document.md

# With watermark
md2pdf convert --wm-text "DRAFT" --wm-opacity 0.15 document.md

# Override document title
md2pdf convert --doc-title "Final Report" document.md

# Page breaks before H1 and H2 headings
md2pdf convert --break-before h1,h2 document.md
```

### Docker

```bash
# Convert a single file
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf convert document.md

# Convert with output path
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf convert -o output.pdf input.md

# Batch convert directory
docker run --rm -v $(pwd):/data ghcr.io/alnah/go-md2pdf convert ./docs/ -o ./pdfs/
```

## Configuration

Config files are loaded from `~/.config/go-md2pdf/` or current directory.
Supported formats: `.yaml`, `.yml`

| Option                  | Type   | Default      | Description                              |
| ----------------------- | ------ | ------------ | ---------------------------------------- |
| `output.defaultDir`     | string | -            | Default output directory                 |
| `css.style`             | string | -            | Embedded style name                      |
| `assets.basePath`       | string | -            | Custom assets directory (styles, templates) |
| `author.name`           | string | -            | Author name (used by cover, signature)   |
| `author.title`          | string | -            | Author professional title                |
| `author.email`          | string | -            | Author email                             |
| `author.organization`   | string | -            | Organization name                        |
| `author.phone`          | string | -            | Contact phone number                     |
| `author.address`        | string | -            | Postal address (multiline via YAML `\|`) |
| `author.department`     | string | -            | Department name                          |
| `document.title`        | string | -            | Document title ("" = auto from H1)       |
| `document.subtitle`     | string | -            | Document subtitle                        |
| `document.version`      | string | -            | Version string (used in cover, footer)   |
| `document.date`         | string | -            | Date (see [Date Formats](#date-formats)) |
| `document.clientName`   | string | -            | Client/customer name                     |
| `document.projectName`  | string | -            | Project name                             |
| `document.documentType` | string | -            | Document type (e.g., "Specification")    |
| `document.documentID`   | string | -            | Document ID (e.g., "DOC-2025-001")       |
| `document.description`  | string | -            | Brief document summary                   |
| `input.defaultDir`      | string | -            | Default input directory                  |
| `page.size`             | string | `"letter"`   | letter, a4, legal                        |
| `page.orientation`      | string | `"portrait"` | portrait, landscape                      |
| `page.margin`           | float  | `0.5`        | Margin in inches (0.25-3.0)              |
| `cover.enabled`         | bool   | `false`      | Show cover page                          |
| `cover.logo`            | string | -            | Logo path or URL                         |
| `cover.showDepartment`  | bool   | `false`      | Show author.department on cover          |
| `toc.enabled`           | bool   | `false`      | Show table of contents                   |
| `toc.title`             | string | -            | TOC title (empty = no title)             |
| `toc.minDepth`          | int    | `2`          | Min heading depth (1-6, skips H1)        |
| `toc.maxDepth`          | int    | `3`          | Max heading depth (1-6)                  |
| `footer.enabled`        | bool   | `false`      | Show footer                              |
| `footer.showPageNumber` | bool   | `false`      | Show page numbers                        |
| `footer.position`       | string | `"right"`    | left, center, right                      |
| `footer.text`           | string | -            | Custom footer text                       |
| `footer.showDocumentID` | bool   | `false`      | Show document.documentID in footer       |
| `signature.enabled`     | bool   | `false`      | Show signature block                     |
| `signature.imagePath`   | string | -            | Photo path or URL                        |
| `signature.links`       | array  | -            | Links (label, url)                       |
| `watermark.enabled`     | bool   | `false`      | Show watermark                           |
| `watermark.text`        | string | -            | Watermark text (required if enabled)     |
| `watermark.color`       | string | `"#888888"`  | Watermark color (hex)                    |
| `watermark.opacity`     | float  | `0.1`        | Watermark opacity (0.0-1.0)              |
| `watermark.angle`       | float  | `-45`        | Watermark rotation (degrees)             |
| `pageBreaks.enabled`    | bool   | `false`      | Enable page break features               |
| `pageBreaks.beforeH1`   | bool   | `false`      | Page break before H1 headings            |
| `pageBreaks.beforeH2`   | bool   | `false`      | Page break before H2 headings            |
| `pageBreaks.beforeH3`   | bool   | `false`      | Page break before H3 headings            |
| `pageBreaks.orphans`    | int    | `2`          | Min lines at page bottom (1-5)           |
| `pageBreaks.widows`     | int    | `2`          | Min lines at page top (1-5)              |

<details>
<summary>Example config file</summary>

```yaml
# ~/.config/go-md2pdf/work.yaml

# Input/Output directories
input:
  defaultDir: './docs/markdown' # Default input when no arg provided

output:
  defaultDir: './docs/pdf' # Default output when no -o flag

# Shared author info (used by cover and signature)
author:
  name: 'John Doe'
  title: 'Senior Developer'
  email: 'john@example.com'
  organization: 'Acme Corp'
  phone: '+1 555-0123'
  address: |
    123 Main Street
    San Francisco, CA 94102
  department: 'Engineering'

# Shared document metadata (used by cover and footer)
document:
  title: '' # "" = auto from H1 or filename
  subtitle: 'Internal Document'
  version: 'v1.0'
  # Date formats:
  #   - Literal: '2025-01-11'
  #   - Auto (ISO): 'auto' -> 2025-01-11
  #   - Auto with format: 'auto:DD/MM/YYYY' -> 11/01/2025
  #   - Auto with preset: 'auto:long' -> January 11, 2025
  # Presets: iso, european, us, long
  # Tokens: YYYY, YY, MMMM, MMM, MM, M, DD, D
  # Escaping: [text] -> literal text
  date: 'auto'
  clientName: 'Client Corp'
  projectName: 'Project Alpha'
  documentType: 'Technical Specification'
  documentID: 'DOC-2025-001'
  description: 'Technical documentation for Project Alpha'

# Page layout
page:
  size: 'a4'           # letter (default), a4, legal
  orientation: 'portrait' # portrait (default), landscape
  margin: 0.75         # inches, 0.25-3.0 (default: 0.5)

# Styling
css:
  # Available styles:
  #   - technical: system-ui, clean borders, GitHub syntax highlighting
  #   - creative: colorful headings, badges, bullet points
  #   - academic: Georgia/Times serif, 1.8 line height, academic tables
  #   - corporate: Arial/Helvetica, blue accents, business style
  #   - legal: Times New Roman, double line height, wide margins
  #   - invoice: Arial, optimized tables, minimal cover
  #   - manuscript: Courier New mono, scene breaks, simplified cover
  style: 'technical'

assets:
  basePath: '' # "" = use embedded assets

# Cover page
cover:
  enabled: true
  logo: '/path/to/logo.png' # path or URL
  showDepartment: true      # show author.department on cover

# Table of contents
toc:
  enabled: true
  title: 'Table of Contents'
  minDepth: 2 # 1-6 (default: 2, skips H1)
  maxDepth: 3 # 1-6 (default: 3)

# Footer
footer:
  enabled: true
  position: 'center'     # left, center, right (default: right)
  showPageNumber: true
  showDocumentID: true   # show document.documentID in footer
  text: ''               # optional custom text

# Signature block
signature:
  enabled: true
  imagePath: '/path/to/signature.png'
  links:
    - label: 'GitHub'
      url: 'https://github.com/johndoe'
    - label: 'LinkedIn'
      url: 'https://linkedin.com/in/johndoe'

# Watermark
watermark:
  enabled: false
  text: 'DRAFT'      # DRAFT, CONFIDENTIAL, SAMPLE, PREVIEW, etc.
  color: '#888888'   # hex color (default: #888888)
  opacity: 0.1       # 0.0-1.0 (default: 0.1, recommended: 0.05-0.15)
  angle: -45         # -90 to 90 (default: -45 = diagonal)

# Page breaks
pageBreaks:
  enabled: true
  beforeH1: true
  beforeH2: false
  beforeH3: false
  orphans: 2 # min lines at page bottom, 1-5 (default: 2)
  widows: 2  # min lines at page top, 1-5 (default: 2)
```

</details>

### Date Formats

The `document.date` field supports auto-generation with customizable formats:

| Syntax        | Example           | Output          |
| ------------- | ----------------- | --------------- |
| `auto`        | `auto`            | 2026-01-09      |
| `auto:FORMAT` | `auto:DD/MM/YYYY` | 09/01/2026      |
| `auto:preset` | `auto:long`       | January 9, 2026 |

**Presets:** `iso` (YYYY-MM-DD), `european` (DD/MM/YYYY), `us` (MM/DD/YYYY), `long` (MMMM D, YYYY)

**Tokens:** `YYYY`, `YY`, `MMMM` (January), `MMM` (Jan), `MM`, `M`, `DD`, `D`

**Escaping:** Use brackets for literal text: `auto:[Date:] YYYY-MM-DD` → "Date: 2026-01-09"

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

See: [BSD-3-Clause](LICENSE.txt).
