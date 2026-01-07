# Architecture

## Pattern

**Pipeline** orchestrated by a **Service Facade**.

```
                         Service.Convert()
                               │
       ┌───────────┬───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼           ▼
   mdtransform   md2html   htmlinject   html2pdf    assets
```

- **Service Facade** - Single entry point
- **Pipeline** - Chained transformations
- **Dependency Injection** - Components via interfaces

---

## Data Flow

```
Markdown ──▶ mdtransform ──▶ md2html ──▶ htmlinject ──▶ html2pdf ──▶ PDF
                │               │             │              │
           Normalize        Goldmark      CSS inject      Chrome
           Highlights       Tables        Signature       Headless
           Blank lines      Code
```

| Stage           | Transformation | Tool           |
| --------------- | -------------- | -------------- |
| **mdtransform** | MD -> MD       | Regex          |
| **md2html**     | MD -> HTML     | Goldmark       |
| **htmlinject**  | HTML -> HTML   | String replace |
| **html2pdf**    | HTML -> PDF    | Rod (Chrome)   |
