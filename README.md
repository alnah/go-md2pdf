# go-md2pdf

[![Go Reference](https://pkg.go.dev/badge/github.com/alnah/go-md2pdf.svg)](https://pkg.go.dev/github.com/alnah/go-md2pdf)
[![Go Report Card](https://goreportcard.com/badge/github.com/alnah/go-md2pdf)](https://goreportcard.com/report/github.com/alnah/go-md2pdf)
[![Build Status](https://img.shields.io/github/actions/workflow/status/alnah/go-md2pdf/ci.yml?branch=main)](https://github.com/alnah/go-md2pdf/actions)
[![Coverage](https://codecov.io/gh/alnah/go-md2pdf/branch/main/graph/badge.svg)](https://codecov.io/gh/alnah/go-md2pdf)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.txt)

> Convert Markdown to print-ready PDF in **3 lines of code** - with signatures, footers, and custom styling.

## Installation

```bash
go install github.com/alnah/go-md2pdf/cmd/md2pdf@latest
```

<details>
<summary>Other installation methods</summary>

### Homebrew (macOS/Linux)

```bash
brew install alnah/go-md2pdf/go-md2pdf
```

### Docker

```bash
docker pull ghcr.io/alnah/go-md2pdf:latest
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/alnah/go-md2pdf/releases).

</details>

## Features

- **CLI + Library** - Use as `md2pdf` command or import in Go
- **Batch conversion** - Process directories, mirror structure
- **Custom styling** - Embedded themes or your own CSS
- **Signatures** - Name, title, email, photo, links
- **Footers** - Page numbers, dates, status text

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

### With Footer and Signature

```go
pdf, err := svc.Convert(ctx, md2pdf.Input{
    Markdown: content,
    Footer: &md2pdf.Footer{
        ShowPageNumber: true,
        Position:       "center",
        Date:           "2026-01-15",
        Status:         "DRAFT",
    },
    Signature: &md2pdf.Signature{
        Name:  "John Doe",
        Title: "Senior Developer",
        Email: "john@example.com",
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

## CLI Reference

```
md2pdf [flags] <input>

Flags:
  -o, --output       Output file or directory
  -c, --config       Config name (work, personal) or path
      --css          Custom CSS file
      --no-style     Disable default styling
      --no-footer    Disable footer
      --no-signature Disable signature
  -q, --quiet        Only show errors
  -v, --verbose      Show detailed timing
      --version      Show version and exit
```

### Examples

```bash
# Single file with custom output
md2pdf -o report.pdf input.md

# Batch with config
md2pdf --config work ./docs/ -o ./pdfs/

# Custom CSS, no footer
md2pdf --css custom.css --no-footer document.md
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

| Option                  | Type   | Default   | Description              |
| ----------------------- | ------ | --------- | ------------------------ |
| `input.defaultDir`      | string | -         | Default input directory  |
| `output.defaultDir`     | string | -         | Default output directory |
| `css.style`             | string | -         | Embedded style name      |
| `footer.enabled`        | bool   | `false`   | Show footer              |
| `footer.showPageNumber` | bool   | `false`   | Show page numbers        |
| `footer.position`       | string | `"right"` | left, center, right      |
| `footer.date`           | string | -         | Date text                |
| `footer.status`         | string | -         | Status text (DRAFT, etc) |
| `footer.text`           | string | -         | Custom footer text       |
| `signature.enabled`     | bool   | `false`   | Show signature block     |
| `signature.name`        | string | -         | Signer name              |
| `signature.title`       | string | -         | Signer title             |
| `signature.email`       | string | -         | Signer email             |
| `signature.imagePath`   | string | -         | Photo path or URL        |
| `signature.links`       | array  | -         | Links (label, url)       |

<details>
<summary>Example config file</summary>

```yaml
# ~/.config/go-md2pdf/work.yaml

css:
  style: 'fle' # add your styles to internal/assets/styles/, and build or install

footer:
  enabled: true
  showPageNumber: true
  position: 'center'
  date: '2026-01-15'
  status: 'DRAFT'

signature:
  enabled: true
  name: 'John Doe'
  title: 'Developer'
  email: 'john@example.com'
  links:
    - label: 'GitHub'
      url: 'https://github.com/johndoe'
```

</details>

## Project Structure

```
go-md2pdf/
├── service.go          # Public API: New(), Convert(), Close()
├── types.go            # Input, Footer, Signature, Link
├── mdtransform.go      # Markdown preprocessing
├── md2html.go          # Markdown to HTML (Goldmark)
├── htmlinject.go       # CSS/signature injection
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
