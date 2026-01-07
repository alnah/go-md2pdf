# Project Layout

```
go-md2pdf/                      # package md2pdf (library)
│
├── service.go                  # New(), Convert(), Close()
├── types.go                    # Input, Footer, Signature
├── errors.go                   # Sentinel errors
│
├── mdtransform.go              # MD -> MD
├── md2html.go                  # MD -> HTML
├── htmlinject.go               # HTML -> HTML
├── html2pdf.go                 # HTML -> PDF
├── fileutil.go                 # Temp files
│
├── cmd/md2pdf/                 # CLI
│   ├── main.go
│   ├── convert.go
│   └── serve.go                # gRPC (future)
│
├── internal/
│   ├── assets/                 # CSS, templates
│   ├── config/                 # CLI config
│   └── yamlutil/               # YAML wrapper
│
├── proto/                      # gRPC (future)
│   └── md2pdf.proto
│
└── docs/
```

## Conventions

- **Library at root** - `import "github.com/alnah/go-md2pdf"`
- **Files named by transformation** - `mdtransform`, `md2html`, `htmlinject`, `html2pdf`
- **internal/** - Private code (assets, config)
- **cmd/** - Binaries
