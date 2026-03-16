# syntax=docker/dockerfile:1

# ====================
# Builder
# ====================
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -tags stdjson \
    -o /bin/operator \
    ./cmd/operator

# ====================
# Development
# ====================
FROM golang:1.25-alpine AS development
WORKDIR /app

RUN apk add --no-cache git
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV ENV=development
ENV PORT=18790

EXPOSE 18790

CMD ["air", "-c", ".air.toml"]

# ====================
# Production
# ====================
FROM alpine:3.21 AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget

# Create non-root user
RUN addgroup --system --gid 1001 operator && \
    adduser --system --uid 1001 --ingroup operator operator

# Copy binary
COPY --from=builder /bin/operator /bin/operator

# Create workspace directory
RUN mkdir -p /home/operator/.operator/workspace && \
    chown -R operator:operator /home/operator/.operator

USER operator
WORKDIR /home/operator

ENV PORT=18790

EXPOSE 18790

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:18790/health || exit 1

ENTRYPOINT ["/bin/operator"]
CMD ["gateway"]
