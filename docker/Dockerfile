# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies including ICU for Unicode support
RUN apk add --no-cache gcc musl-dev icu-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with ICU support for proper Unicode case conversion
RUN CGO_ENABLED=1 GOOS=linux go build -tags "icu" -o biblio-catalog .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies including ICU libraries for Unicode support
RUN apk add --no-cache ca-certificates tzdata icu-libs

# Copy binary from builder
COPY --from=builder /app/biblio-catalog .

# Copy static files and templates
COPY --from=builder /app/web ./web
COPY --from=builder /app/internal/db/genres_en.tsv ./internal/db/genres_en.tsv
COPY --from=builder /app/internal/db/genres_ru.tsv ./internal/db/genres_ru.tsv

# Create directories for database and books
RUN mkdir -p /db /books

ENV OPDS_DATABASE_PATH=/db/opds.db
ENV OPDS_SERVER_HOST=0.0.0.0
ENV OPDS_SERVER_PORT=9988

# Expose port
EXPOSE 9988

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9988/api/setup/check || exit 1

# Run the server
CMD ["./biblio-catalog"]
