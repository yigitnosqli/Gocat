# 🐳 GoCat - Modern Netcat Alternative
# Multi-stage Docker build for optimal image size

# Build stage
FROM golang:1.21-alpine AS builder

# 🏷️ Metadata
LABEL maintainer="Ibrahim <ibrahimsql@proton.me>"
LABEL description="GoCat - Modern netcat alternative written in Go"
LABEL version="1.0.0"

# 🔧 Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# 📁 Set working directory
WORKDIR /build

# 📦 Copy go mod files first for better caching
COPY go.mod go.sum ./

# 📥 Download dependencies
RUN go mod download && go mod verify

# 📋 Copy source code
COPY . .

# 🏗️ Build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG BINARY_PATH

# 🔨 Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-s -w \
             -X main.version=${VERSION} \
             -X main.buildTime=${BUILD_TIME} \
             -X main.gitCommit=${GIT_COMMIT} \
             -X main.builtBy=docker" \
    -o gocat .

# 🧪 Test the binary
RUN ./gocat --help

# Production stage
FROM scratch AS production

# 🏷️ Labels for the final image
LABEL org.opencontainers.image.title="GoCat"
LABEL org.opencontainers.image.description="Modern netcat alternative written in Go"
LABEL org.opencontainers.image.url="https://github.com/ibrahmsql/gocat"
LABEL org.opencontainers.image.source="https://github.com/ibrahmsql/gocat"
LABEL org.opencontainers.image.vendor="Ibrahim"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"

# 📋 Copy necessary files from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# 📁 Create non-root user
USER 65534:65534

# 📦 Copy the binary
COPY --from=builder /build/gocat /usr/local/bin/gocat

# 🔧 Set entrypoint
ENTRYPOINT ["/usr/local/bin/gocat"]

# 📖 Default command
CMD ["--help"]

# 🌐 Expose common ports (documentation only)
EXPOSE 8080 9999

# 🏥 Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/gocat", "version"] || exit 1

# Alternative multi-arch build stage
FROM golang:1.21-alpine AS builder-multiarch

# 🔧 Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# 📁 Set working directory
WORKDIR /build

# 📦 Copy source
COPY . .

# 🏗️ Build arguments for multi-arch
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# 🔨 Build for target architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -a -installsuffix cgo \
    -ldflags="-s -w \
             -X main.version=${VERSION} \
             -X main.buildTime=${BUILD_TIME} \
             -X main.gitCommit=${GIT_COMMIT} \
             -X main.builtBy=docker-multiarch" \
    -o gocat .

# Multi-arch production stage
FROM scratch AS production-multiarch

# 🏷️ Multi-arch labels
LABEL org.opencontainers.image.title="GoCat"
LABEL org.opencontainers.image.description="Modern netcat alternative written in Go (Multi-arch)"
LABEL org.opencontainers.image.url="https://github.com/ibrahmsql/gocat"
LABEL org.opencontainers.image.source="https://github.com/ibrahmsql/gocat"
LABEL org.opencontainers.image.vendor="Ibrahim"
LABEL org.opencontainers.image.licenses="MIT"

# 📋 Copy necessary files
COPY --from=builder-multiarch /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder-multiarch /usr/share/zoneinfo /usr/share/zoneinfo

# 📦 Copy the binary
COPY --from=builder-multiarch /build/gocat /usr/local/bin/gocat

# 👤 Use non-root user
USER 65534:65534

# 🔧 Set entrypoint
ENTRYPOINT ["/usr/local/bin/gocat"]
CMD ["--help"]

# Development stage for debugging
FROM golang:1.21-alpine AS development

# 🔧 Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    bash \
    curl \
    netcat-openbsd \
    tcpdump \
    strace

# 📁 Set working directory
WORKDIR /app

# 📦 Copy source
COPY . .

# 📥 Download dependencies
RUN go mod download

# 🔨 Build with debug info
RUN go build -gcflags="-N -l" -o gocat-debug .

# 🔧 Set entrypoint for development
ENTRYPOINT ["/app/gocat-debug"]
CMD ["--help"]

# Testing stage
FROM builder AS testing

# 🧪 Run tests
RUN go test -v ./...

# 🔍 Run security scan
RUN go vet ./...

# 📊 Generate coverage report
RUN go test -coverprofile=coverage.out ./...

# 🎯 Default to production stage
FROM production
