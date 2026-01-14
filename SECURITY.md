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

go-md2pdf processes local Markdown files and generates PDFs using headless Chrome. The main security considerations are:

- File path handling (path traversal)
- HTML/CSS injection in generated output
- Dependencies (go-rod, Goldmark)

The tool does not handle authentication, network requests to external services, or sensitive user data.
