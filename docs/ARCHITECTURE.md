# Architecture

## Pattern

**Pipeline** orchestrated by a **Converter Facade**, with **ConverterPool** for parallelism.

```
                        Converter.Convert()
                               │
       ┌───────────┬───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼           ▼
   mdtransform   md2html   htmlinject     pdf       assets
   (internal)   (internal)  (internal)   (root)    (internal)
```

- **Converter Facade** - Single entry point, owns browser lifecycle
- **Pipeline** - Chained transformations in `internal/pipeline/`
- **ConverterPool** - Lazy browser init, parallel batch processing
- **Dependency Injection** - Components via interfaces

---

## Package Structure

```
md2pdf/                     # Root: public API + PDF generation
├── types.go                # Public types: Input, Cover, Footer, etc.
├── converter.go            # Converter facade, pipeline orchestration
├── pool.go                 # ConverterPool for parallel processing
├── pdf.go                  # HTML->PDF via headless Chrome (go-rod)
├── cssbuilders.go          # CSS generation (watermark, page breaks)
├── assets.go               # Public AssetLoader interface
├── errors.go               # Public sentinel errors
│
├── cmd/md2pdf/             # CLI application
│   ├── main.go             # Entry point, command routing
│   ├── convert*.go         # Convert command (batch, discovery, params)
│   ├── flags.go            # CLI flag definitions (pflag)
│   ├── completion*.go      # Shell completion generators
│   └── signal_*.go         # OS signal handling (Unix/Windows)
│
└── internal/
    ├── pipeline/           # MD->HTML pipeline (see doc.go)
    │   ├── mdtransform.go  # Preprocessing (normalize, highlights)
    │   ├── md2html.go      # Goldmark conversion
    │   └── htmlinject.go   # CSS/cover/TOC/signature injection
    │
    ├── assets/             # Asset loading (styles, templates)
    │   ├── embedded.go     # Embedded FS loader
    │   ├── filesystem.go   # Filesystem loader
    │   ├── resolver.go     # Custom-first with embedded fallback
    │   └── styles/         # Built-in CSS styles
    │
    ├── config/             # YAML configuration
    ├── dateutil/           # Date format parsing (auto:FORMAT)
    ├── fileutil/           # File utilities
    ├── yamlutil/           # YAML wrapper (goccy/go-yaml)
    └── process/            # Process management (kill Chrome)
```

**Design decision**: PDF generation (`pdf.go`) stays in root rather than `internal/pipeline/` to avoid circular dependencies. It depends on root types (`PageSettings`, `Watermark`) and the clean separation keeps `internal/pipeline/` focused on document structure (MD->HTML) while root handles rendering concerns (HTML->PDF).

---

## Data Flow

```
Markdown ──▶ mdtransform ──▶ md2html ──▶ htmlinject ──▶ pdf ──▶ PDF
                │               │             │           │
           Normalize        Goldmark      Page breaks  Chrome
           Highlights       GFM/TOC IDs   Watermark    Headless
           Blank lines      Footnotes     Cover page   Footer
                                          TOC inject
                                          CSS inject
                                          Signature
```

| Stage           | Transformation | Location                        | Tool            |
| --------------- | -------------- | ------------------------------- | --------------- |
| **mdtransform** | MD -> MD       | `internal/pipeline/`            | Regex           |
| **md2html**     | MD -> HTML     | `internal/pipeline/`            | Goldmark (GFM)  |
| **htmlinject**  | HTML -> HTML   | `internal/pipeline/`            | String/template |
| **pdf**         | HTML -> PDF    | root (`pdf.go`)                 | Rod (Chrome)    |

---

## Injection Order

```
1. Page breaks CSS      ──▶  <head> (lowest priority)
2. Watermark CSS        ──▶  <head>
3. User CSS             ──▶  <head> (highest priority)
4. Cover page           ──▶  after <body>
5. TOC                  ──▶  after cover (or <body>)
6. Signature            ──▶  before </body>
7. Footer               ──▶  Chrome native footer
```

---

## Browser Lifecycle

- Browsers created lazily on first `Acquire()` from pool
- `process.KillProcessGroup()` terminates Chrome + all child processes (GPU, renderer)
- Platform-specific: `syscall.Kill(-pid)` on Unix, `taskkill /T` on Windows
- Implementation in `internal/process/`
