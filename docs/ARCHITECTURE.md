# Architecture

## Pattern

**Pipeline** orchestrated by a **Service Facade**, with **ServicePool** for parallelism.

```
                         Service.Convert()
                               │
       ┌───────────┬───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼           ▼
   mdtransform   md2html   htmlinject   html2pdf    assets
   (internal)   (internal)  (internal)   (root)    (internal)
```

- **Service Facade** - Single entry point, owns browser lifecycle
- **Pipeline** - Chained transformations in `internal/pipeline/`
- **ServicePool** - Lazy browser init, parallel batch processing
- **Dependency Injection** - Components via interfaces

---

## Data Flow

```
Markdown ──▶ mdtransform ──▶ md2html ──▶ htmlinject ──▶ html2pdf ──▶ PDF
                │               │             │              │
           Normalize        Goldmark      Page breaks    Chrome
           Highlights       GFM/TOC IDs   Watermark      Headless
           Blank lines      Footnotes     Cover page     Footer
                                          TOC inject
                                          CSS inject
                                          Signature
```

| Stage           | Transformation | Location                     | Tool            |
| --------------- | -------------- | ---------------------------- | --------------- |
| **mdtransform** | MD -> MD       | `internal/pipeline/`         | Regex           |
| **md2html**     | MD -> HTML     | `internal/pipeline/`         | Goldmark (GFM)  |
| **htmlinject**  | HTML -> HTML   | `internal/pipeline/`         | String/template |
| **html2pdf**    | HTML -> PDF    | root (depends on PageSettings) | Rod (Chrome)  |

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
