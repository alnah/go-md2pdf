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

# Install CA certificates, fontconfig, emoji, and download quality fonts
# - Inter: Modern sans-serif, close to Apple's San Francisco
# - JetBrains Mono: Excellent monospace for code
# - fonts-noto-color-emoji: Color emoji support
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    fontconfig \
    fonts-noto-color-emoji \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/* \
    # Download and install Inter font
    && curl -sL https://github.com/rsms/inter/releases/download/v4.0/Inter-4.0.zip -o /tmp/inter.zip \
    && unzip -q /tmp/inter.zip -d /tmp/inter \
    && mkdir -p /usr/share/fonts/truetype/inter \
    && cp /tmp/inter/Inter.ttc /usr/share/fonts/truetype/inter/ \
    && rm -rf /tmp/inter /tmp/inter.zip \
    # Download and install JetBrains Mono font
    && curl -sL https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip -o /tmp/jbmono.zip \
    && unzip -q /tmp/jbmono.zip -d /tmp/jbmono \
    && mkdir -p /usr/share/fonts/truetype/jetbrains-mono \
    && cp /tmp/jbmono/fonts/ttf/*.ttf /usr/share/fonts/truetype/jetbrains-mono/ \
    && rm -rf /tmp/jbmono /tmp/jbmono.zip \
    # Rebuild font cache
    && fc-cache -fv \
    # Remove curl/unzip (no longer needed)
    && apt-get purge -y curl unzip \
    && apt-get autoremove -y

# Copy binary
COPY --from=builder /go-md2pdf /usr/bin/go-md2pdf

# Create non-root user with home directory (rod needs it for cache)
RUN useradd -r -u 1000 -m -s /bin/false appuser

# Rod configuration:
# - ROD_BROWSER_BIN: use pre-installed headless Chrome
# - ROD_NO_SANDBOX: Docker lacks kernel capabilities for Chrome sandboxing
ENV ROD_BROWSER_BIN=/headless-shell/headless-shell \
    ROD_NO_SANDBOX=1

USER appuser

# Working directory for files
WORKDIR /data

ENTRYPOINT ["/usr/bin/go-md2pdf"]
