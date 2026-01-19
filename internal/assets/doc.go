// Package assets provides CSS styles and HTML templates for PDF generation.
//
// # Loader Architecture
//
// The package implements a layered loading system:
//
//	AssetLoader (interface)
//	    │
//	    ├── EmbeddedLoader    - loads from go:embed filesystem (default styles)
//	    ├── FilesystemLoader  - loads from custom directory on disk
//	    └── AssetResolver     - combines both with custom-first fallback
//
// EmbeddedLoader provides built-in styles (default, technical, corporate, etc.)
// and templates embedded at compile time.
//
// FilesystemLoader allows users to provide custom assets from a directory,
// with path traversal protection and symlink resolution.
//
// AssetResolver is the primary loader used by the converter. It tries the
// custom FilesystemLoader first, falling back to EmbeddedLoader if the asset
// is not found. This enables overriding specific assets while keeping defaults.
//
// # Directory Structure
//
// Assets are organized by type:
//
//	{basePath}/
//	├── styles/
//	│   └── {name}.css           # CSS styles (e.g., technical.css)
//	└── templates/
//	    └── {name}/
//	        ├── cover.html       # Cover page template
//	        └── signature.html   # Signature block template
//
// # Security
//
// Asset names are validated to prevent path traversal attacks.
// FilesystemLoader resolves symlinks and verifies paths stay within basePath.
package assets
