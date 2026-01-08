# Architecture

## Pattern

**Pipeline** orchestrated by a **Service Facade**, with **ServicePool** for parallelism.

```
                         Service.Convert()
                               │
       ┌───────────┬───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼           ▼
   mdtransform   md2html   htmlinject   html2pdf    assets
```

- **Service Facade** - Single entry point, owns browser lifecycle
- **Pipeline** - Chained transformations with context propagation
- **ServicePool** - Lazy browser init, parallel batch processing
- **Dependency Injection** - Components via interfaces

---

## Data Flow

```
Markdown ──▶ mdtransform ──▶ md2html ──▶ htmlinject ──▶ html2pdf ──▶ PDF
                │               │             │              │
           Normalize        Goldmark      CSS inject      Chrome
           Highlights       GFM tables    Signature       Headless
           Blank lines      Footnotes     Footer opts
```

| Stage           | Transformation | Tool           |
| --------------- | -------------- | -------------- |
| **mdtransform** | MD -> MD       | Regex          |
| **md2html**     | MD -> HTML     | Goldmark (GFM) |
| **htmlinject**  | HTML -> HTML   | String/template|
| **html2pdf**    | HTML -> PDF    | Rod (Chrome)   |

---

## Browser Lifecycle

- Browsers created lazily on first `Acquire()` from pool
- `killProcessGroup()` terminates Chrome + all child processes (GPU, renderer)
- Platform-specific: `syscall.Kill(-pid)` on Unix, `taskkill /T` on Windows
