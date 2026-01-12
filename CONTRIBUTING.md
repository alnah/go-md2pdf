# Contributing

This is a personal tool built for my own workflow. While I share it publicly,
I have limited bandwidth for maintenance and feature development.

## Bug Reports

Issues are welcome, especially for:
- Bugs on platforms I don't use daily (Windows, Linux)
- Edge cases in PDF generation
- Documentation improvements

Please include: OS, Go version, and minimal reproduction steps.

## Feature Requests

I'm unlikely to implement features I don't personally need. Feel free to open
an issue to discuss, but understand that "won't implement" is a likely outcome
for features outside my use case.

## Pull Requests

PRs must reference an existing issue where the approach was discussed and agreed upon.
This avoids wasted effort on both sides.

Small fixes (typos, broken links) are exempt from this rule.

## Contributing Styles

New CSS styles in `internal/assets/styles/` are welcome. They must follow the
established structure (see `technical.css` or `creative.css` as reference):

1. CSS Variables - Central configuration
2. Reset and base styles
3. Typography - Headings and hierarchy
4. Text formatting
5. Components - Blockquotes
6. Components - Lists (including task lists)
7. Components - Tables
8. Code and syntax highlighting (Chroma classes)
9. Images and media
10. Footnotes and signature block
11. Chrome PDF specific rules (`@media all`)
12. Cover page
13. Table of contents
14. Print settings (`@page`, `@media print`)

Requirements:
- Use CSS variables for colors and spacing (easier theming)
- Include `-webkit-print-color-adjust: exact` for color preservation
- Test with actual PDF generation before submitting
- Name should reflect the style's purpose (e.g., `academic.css`, `corporate.css`)
