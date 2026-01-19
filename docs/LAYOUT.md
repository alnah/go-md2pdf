# Project Layout

```
go-md2pdf/                      # package md2pdf (library)
│
├── converter.go                # NewConverter(), Convert(), Close() - facade
├── pool.go                     # ConverterPool, ResolvePoolSize()
├── types.go                    # Input, PageSettings, Footer, Signature, Watermark, Cover, TOC, PageBreaks, Options
├── assets.go                   # AssetLoader, TemplateSet, NewAssetLoader(), NewTemplateSet()
├── errors.go                   # Sentinel errors
├── pdf.go                      # HTML -> PDF (Rod/Chrome)
├── cssbuilders.go              # Watermark/PageBreaks CSS (depend on public types)
│
├── cmd/md2pdf/                 # CLI (md2pdf convert|version|help|completion)
│   ├── main.go                 # Entry point, command dispatch
│   ├── convert.go              # Convert command orchestration
│   ├── convert_batch.go        # Batch processing, worker pool
│   ├── convert_params.go       # Parameter builders (cover, signature, footer, etc.)
│   ├── convert_discovery.go    # File discovery, output path resolution
│   ├── flags.go                # Flag definitions by category
│   ├── help.go                 # Usage text
│   ├── env.go                  # Environment (Now, Stdout, Stderr, AssetLoader)
│   ├── completion.go           # Shell completion command, flag/command definitions
│   ├── completion_{bash,zsh,fish,pwsh}.go  # Shell-specific generators
│   └── signal_{unix,windows}.go
│
├── internal/
│   ├── assets/                 # Asset loading (styles, templates)
│   ├── config/                 # YAML config, validation
│   ├── dateutil/               # Date format parsing, ResolveDate()
│   ├── fileutil/               # File utilities (FileExists, IsFilePath, IsURL)
│   ├── pipeline/               # Conversion pipeline components
│   │   ├── mdtransform.go      # MD -> MD (preprocessing)
│   │   ├── md2html.go          # MD -> HTML (Goldmark)
│   │   └── htmlinject.go       # HTML -> HTML (CSS, cover, TOC, signature)
│   ├── process/                # OS-specific process management
│   │   ├── kill_unix.go        # KillProcessGroup (Unix)
│   │   └── kill_windows.go     # KillProcessGroup (Windows)
│   └── yamlutil/               # YAML wrapper with limits
│
└── docs/
```

## Conventions

- **Library at root** - `import "github.com/alnah/go-md2pdf"`
- **Public API only at root** - Converter, Input, types, errors
- **Pipeline in internal/** - mdtransform, md2html, htmlinject
- **Platform suffix** - `_unix.go`, `_windows.go` for OS-specific code
- **internal/** - Private implementation (pipeline, assets, config, utilities)
- **cmd/** - Binaries
