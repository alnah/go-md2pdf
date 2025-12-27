# md2pdf

Convert Markdown files to PDF.

## Dependencies

### Runtime

- **Chrome** or **Chromium** - Required for PDF generation (headless browser rendering)
- **Pandoc** - Required for Markdown to HTML conversion

### Development

- **Go 1.23+**
- **Chrome** or **Chromium** - Required for integration tests

## Installation

```bash
go install github.com/alnah/md2pdf@latest
```

## Usage

```bash
md2pdf input.md output.pdf
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

# Integration tests (requires Pandoc and Chrome)
make test-integration

# All checks (fmt, vet, lint, security, tests)
make check-all
```

### Available make targets

```bash
make help
```
