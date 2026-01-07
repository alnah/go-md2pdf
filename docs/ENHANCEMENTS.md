# Future Enhancements

Features planned for future versions.

---

## Benchmarks

| Benchmark                 | Description                     |
| ------------------------- | ------------------------------- |
| `BenchmarkMarkdownToHTML` | Markdown -> HTML conversion     |
| `BenchmarkHTMLToPDF`      | HTML -> PDF conversion (Chrome) |
| `BenchmarkFullConversion` | Complete MD -> PDF pipeline     |

**Usage:**

```bash
go test -bench=. -benchmem ./...
```

**Metrics to track:**

- ns/op: time per operation
- B/op: memory allocations
- allocs/op: number of allocations

---

## gRPC Server

## Expose md2pdf as a microservice via gRPC.

## PDF Metadata

| Field    | Source      | Description       |
| -------- | ----------- | ----------------- |
| Title    | Frontmatter | Document title    |
| Subject  | Frontmatter | Topic/description |
| Keywords | Frontmatter | Search keywords   |
| Author   | Config      | Default author    |

---

## Page Settings

| Setting     | Current Value | Future Options      |
| ----------- | ------------- | ------------------- |
| Size        | US Letter     | A4, Letter, Legal   |
| Orientation | Portrait      | Portrait, Landscape |
| Margins     | 0.5 inch      | Customizable        |

---

## Visual Enhancements

| Feature               | Description                           |
| --------------------- | ------------------------------------- |
| **Watermark**         | Background text (DRAFT, CONFIDENTIAL) |
| **Cover page**        | Title, logo, author, date             |
| **Table of contents** | Auto-generated TOC                    |

---

## Typography

| Feature                  | Description          |
| ------------------------ | -------------------- |
| **Page breaks (Maybe?)** | Force before h1/h2   |
| **Widow/orphan control** | Avoid isolated lines |
| **Custom fonts**         | Embed fonts          |

---

## Implementation

| Feature    | Approach                           |
| ---------- | ---------------------------------- |
| Metadata   | YAML frontmatter -> `<meta>` tags  |
| Page size  | `html2pdf` parameters              |
| Watermark  | CSS `@page { background }`         |
| Cover page | Injected HTML + `page-break-after` |
| TOC        | Goldmark extension or post-process |
