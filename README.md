# md2pdf

Convert Markdown files to PDF.

## Dependencies

- **Go 1.25+**

## Installation

```bash
go install github.com/alnah/go-md2pdf@latest
```

## Usage

```bash
# Single file
go-md2pdf input.md

# Directory (recursive)
go-md2pdf ./docs/

# With config
go-md2pdf --config work ./docs/

# Output to specific directory
go-md2pdf ./docs/ -o ./pdf/
```

## Configuration

Create a YAML config file in `~/.config/go-md2pdf/` or the current directory:

```yaml
# work.yaml - Example configuration

input:
  defaultDir: '~/Documents/notes' # Optional: fallback when no input specified

output:
  defaultDir: '~/Documents/pdf' # Optional: fallback (default: same as source)

css:
  style: 'fle' # Optional: style name from internal/assets/styles/ (available: "fle")

footer:
  enabled: true # true | false
  position: 'right' # "left" | "center" | "right"
  showPageNumber: true # true | false
  date: '2025-01-15' # Optional: format YYYY-MM-DD
  status: 'DRAFT' # Optional: "DRAFT" | "FINAL" | "v1.0" | any string
  text: '© Jane Doe' # Optional: free-form text

signature:
  enabled: true # true | false
  name: 'Jane Doe'
  title: 'English Teacher'
  email: 'jane.doe@example.com'
  imagePath: '' # Optional: local path or URL to signature image
  links: # Optional: list of clickable links
    - label: 'GitHub'
      url: 'https://github.com/janedoe'
    - label: 'LinkedIn'
      url: 'https://linkedin.com/in/janedoe'

assets:
  basePath: '' # Optional: custom assets path (default: embedded assets)
```

Use with:

```bash
go-md2pdf --config work
```

## Development

### Build

```bash
make build
```

### Run tests

```bash
# Unit tests only
make test

# Integration tests
make test-integration

# All checks (fmt, vet, lint, security, tests)
make check-all
```

### Available make targets

```bash
make help
```
