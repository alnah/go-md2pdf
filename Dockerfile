# =============================================================================
# Build stage: compile Go binary
# =============================================================================
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Download dependencies first (Docker cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with best practices:
# - CGO_ENABLED=0: static binary
# - trimpath: reproducible builds
# - ldflags -s -w: strip debug info, reduce size
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /go-md2pdf ./cmd/md2pdf

# =============================================================================
# Runtime stage: minimal image with headless Chromium
# =============================================================================
FROM chromedp/headless-shell:stable

# Install CA certificates for HTTPS
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=builder /go-md2pdf /usr/bin/go-md2pdf

# Create non-root user
RUN useradd -r -u 1000 -s /bin/false appuser
USER appuser

# Working directory for files
WORKDIR /data

ENTRYPOINT ["/usr/bin/go-md2pdf"]
