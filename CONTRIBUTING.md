# Contributing

Contributions are welcome! go-md2pdf aims to be the best Markdown-to-PDF tool for professional documents.

## What I'm Looking For

**Actively seeking:**

- New CSS styles (see Contributing Styles below)
- Documentation improvements
- Bug fixes with tests

**Requires discussion first:**

- New features: open an issue before coding
- Architectural changes
- New dependencies

## Setup

```bash
git clone https://github.com/alnah/go-md2pdf.git
cd go-md2pdf
git lfs install  # Required for example PDFs
git lfs pull     # Download example PDFs
```

## How to Contribute

### Bug Reports

Please open an issue with:

- OS and Go version
- Minimal reproduction steps
- Expected vs actual behavior

### Feature Requests

Open an issue to discuss before implementing. This ensures alignment and avoids wasted effort.

### Pull Requests

1. Reference an existing issue (small fixes like typos are exempt)
2. Follow existing code patterns
3. Add tests for new functionality
4. Run `make check-all` before submitting

### Contributing Styles

New CSS styles in `internal/assets/styles/` are especially welcome. Follow the established structure (see `technical.css` as reference):

1. CSS Variables: Central configuration
2. Reset and base styles
3. Typography: Headings and hierarchy
4. Text formatting
5. Components: Blockquotes
6. Components: Lists (including task lists)
7. Components: Tables
8. Code and syntax highlighting (Chroma classes)
9. Images and media
10. Footnotes
11. Signature block
12. Chrome PDF specific rules (`@media all`)
13. Cover page
14. Table of contents
15. Print settings (`@page`, `@media print`)

Requirements:

- Use CSS variables for colors and spacing
- Include `-webkit-print-color-adjust: exact` for color preservation
- Test with actual PDF generation before submitting

## Code of Conduct

Be respectful. Technical disagreements are fine; personal attacks are not.

## Questions?

Open a discussion or issue. Happy to help newcomers get started.
