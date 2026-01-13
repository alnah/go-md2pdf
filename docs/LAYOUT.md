# Project Layout

```
go-md2pdf/                      # package md2pdf (library)
│
├── service.go                  # New(), Convert(), Close()
├── pool.go                     # ServicePool, ResolvePoolSize()
├── types.go                    # Input, PageSettings, Footer, Signature, Watermark, Cover, TOC, PageBreaks, Options
├── assets.go                   # AssetLoader, TemplateSet, NewAssetLoader(), NewTemplateSet()
├── errors.go                   # Sentinel errors
├── date.go                     # Date formatting (auto:FORMAT)
│
├── mdtransform.go              # MD -> MD (preprocessing)
├── md2html.go                  # MD -> HTML (Goldmark)
├── htmlinject.go               # HTML -> HTML (CSS, watermark, cover, TOC, signature)
├── html2pdf.go                 # HTML -> PDF (Rod/Chrome)
├── process_{unix,windows}.go   # killProcessGroup per platform
│
├── cmd/md2pdf/                 # CLI (md2pdf convert|version|help)
│   ├── main.go                 # Entry point, command dispatch
│   ├── convert.go              # Batch conversion, config merging, asset resolution
│   ├── flags.go                # Flag definitions by category
│   ├── help.go                 # Usage text
│   ├── env.go                  # Environment (Now, Stdout, Stderr, AssetLoader)
│   └── signal_{unix,windows}.go
│
├── internal/
│   ├── assets/                 # Asset loading (styles, templates)
│   ├── config/                 # YAML config, validation
│   ├── dateutil/               # Date format parsing
│   ├── fileutil/               # File utilities (FileExists, IsFilePath, IsURL)
│   └── yamlutil/               # YAML wrapper with limits
│
└── docs/
```

## Conventions

- **Library at root** - `import "github.com/alnah/go-md2pdf"`
- **Files named by transformation** - `mdtransform`, `md2html`, `htmlinject`, `html2pdf`
- **Platform suffix** - `_unix.go`, `_windows.go` for OS-specific code
- **internal/** - Private code (assets, config, utilities)
- **cmd/** - Binaries
