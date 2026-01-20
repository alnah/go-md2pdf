# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 1.x     | Yes       |
| < 1.0   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately:

1. **Do not** open a public GitHub issue
2. Email the maintainer or use GitHub's private vulnerability reporting
3. Include steps to reproduce the issue

You can expect a response within 7 days. If confirmed, a fix will be released as soon as possible.

## Scope

go-md2pdf processes Markdown files and generates PDFs using headless Chrome.

### Security Considerations

- **File path handling**: Asset names are validated to prevent path traversal attacks.
- **HTML/CSS injection**: User-provided content is escaped in generated output.
- **Dependencies**: go-rod (browser automation), Goldmark (markdown parsing).

### Network Requests

The tool may make network requests in these cases:

- **Cover logo**: When `cover.logo` is a URL (e.g., `https://example.com/logo.png`)
- **Markdown images**: When image sources are URLs (e.g., `![img](https://...)`)
- **Chromium download**: On first run, go-rod may download Chromium from Google's mirrors if not present

These requests are initiated by the headless Chrome browser, not the Go code directly.

### Data Handling

- The tool does not handle authentication
- The tool does not store or transmit user data
- Temporary files (HTML, screenshots) are created in the system temp directory and cleaned up after conversion
