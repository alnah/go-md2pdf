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

See [LAYOUT.md](LAYOUT.md) for the complete project layout.

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
