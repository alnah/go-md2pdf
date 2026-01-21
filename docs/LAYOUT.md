# Project Layout

```
go-md2pdf/                      # package md2pdf (library)
│
├── doc.go                      # Package documentation (godoc)
├── converter.go                # NewConverter(), Convert(), Close() - facade
├── pool.go                     # ConverterPool, ResolvePoolSize()
├── types.go                    # Input, PageSettings, Footer, Signature, Watermark, Cover, TOC, PageBreaks, Options
├── assets.go                   # AssetLoader, TemplateSet, NewAssetLoader(), NewTemplateSet()
├── errors.go                   # Sentinel errors
├── pdf.go                      # HTML -> PDF (Rod/Chrome)
├── cssbuilders.go              # Watermark/PageBreaks CSS (depend on public types)
├── example_test.go             # Runnable examples for godoc (Example*, ExampleConverterPool, etc.)
│
├── cmd/md2pdf/                 # CLI (md2pdf convert|doctor|version|help|completion)
│   ├── main.go                 # Entry point, command dispatch
│   ├── exit_codes.go           # Semantic exit codes (0-4) and exitCodeFor()
│   ├── convert.go              # Convert command orchestration
│   ├── convert_batch.go        # Batch processing, worker pool
│   ├── convert_params.go       # Parameter builders (cover, signature, footer, etc.)
│   ├── convert_discovery.go    # File discovery, output path resolution
│   ├── doctor.go               # Doctor command (system diagnostics)
│   ├── flags.go                # Flag definitions by category
│   ├── help.go                 # Usage text
│   ├── env.go                  # Environment (Now, Stdout, Stderr, AssetLoader)
│   ├── env_config.go           # Environment variable configuration
│   ├── completion.go           # Shell completion command, flag/command definitions
│   ├── completion_{bash,zsh,fish,pwsh}.go  # Shell-specific generators
│   └── signal_{unix,windows}.go
│
├── internal/
│   ├── assets/                 # Asset loading (styles, templates)
│   │   ├── assets.go           # Loader interface and factory
│   │   ├── embedded.go         # Embedded assets (go:embed)
│   │   ├── filesystem.go       # Filesystem-based loader
│   │   ├── resolver.go         # Asset resolution logic
│   │   ├── templateset.go      # Template set management
│   │   ├── validation.go       # Asset validation
│   │   ├── styles/             # Embedded CSS styles
│   │   │   ├── default.css
│   │   │   ├── technical.css
│   │   │   ├── creative.css
│   │   │   ├── academic.css
│   │   │   ├── corporate.css
│   │   │   ├── legal.css
│   │   │   ├── invoice.css
│   │   │   └── manuscript.css
│   │   └── templates/default/  # Default HTML templates
│   │       ├── cover.html
│   │       └── signature.html
│   ├── config/                 # YAML config, validation
│   ├── dateutil/               # Date format parsing, ResolveDate()
│   ├── fileutil/               # File utilities (FileExists, IsFilePath, IsURL)
│   ├── hints/                  # Actionable error message hints
│   ├── pipeline/               # Conversion pipeline components
│   │   ├── mdtransform.go      # MD -> MD (preprocessing)
│   │   ├── md2html.go          # MD -> HTML (Goldmark)
│   │   ├── htmlinject.go       # HTML -> HTML (CSS, cover, TOC, signature)
│   │   └── pathrewrite.go      # Rewrite relative paths for SourceDir
│   ├── process/                # OS-specific process management
│   │   ├── kill_unix.go        # KillProcessGroup (Unix)
│   │   └── kill_windows.go     # KillProcessGroup (Windows)
│   └── yamlutil/               # YAML wrapper with limits
│
├── examples/                   # Example markdown files and generated PDFs
│
└── docs/                       # Documentation
```

## Root Configuration Files

```
go-md2pdf/
├── go.mod                      # Module definition, dependencies
├── go.sum                      # Dependency checksums
├── Makefile                    # Build, test, lint commands
├── Dockerfile                  # Container build
├── README.md                   # User documentation
├── CONTRIBUTING.md             # Contributor guide
├── SECURITY.md                 # Security policy
├── CODE_OF_CONDUCT.md          # Community guidelines
└── LICENSE.txt                 # BSD-3-Clause license
```

## Conventions

- **Library at root** - `import "github.com/alnah/go-md2pdf"`
- **Public API only at root** - Converter, Input, types, errors
- **Pipeline in internal/** - mdtransform, md2html, htmlinject
- **Platform suffix** - `_unix.go`, `_windows.go` for OS-specific code
- **internal/** - Private implementation (pipeline, assets, config, utilities)
- **cmd/** - Binaries

## Test Conventions

| Pattern                     | Purpose                              | Example                        |
| --------------------------- | ------------------------------------ | ------------------------------ |
| `*_test.go`                 | Unit tests (same package)            | `converter_test.go`            |
| `*_integration_test.go`     | Integration tests (require browser)  | `converter_integration_test.go`|
| `*_bench_test.go`           | Benchmarks                           | `pool_bench_test.go`           |
| `example_test.go`           | Runnable examples for godoc          | `example_test.go`              |

- Unit tests: `make test` - fast, no external dependencies
- Integration tests: `make test-integration` - require Chrome, use `-tags=integration`
- Benchmarks: `make bench` - use `-tags=bench`
- Examples: `go test -run Example` - appear on pkg.go.dev

## Embedded Styles

| Style          | Target Use Case                                |
| -------------- | ---------------------------------------------- |
| `default`      | Minimal, neutral baseline                      |
| `technical`    | System fonts, GitHub syntax highlighting       |
| `creative`     | Colorful headings, visual flair                |
| `academic`     | Serif fonts, academic formatting               |
| `corporate`    | Arial/Helvetica, blue accents, business style  |
| `legal`        | Times New Roman, double spacing                |
| `invoice`      | Optimized tables, minimal cover                |
| `manuscript`   | Courier New mono, scene breaks                 |
