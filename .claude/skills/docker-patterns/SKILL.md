# Skill: Docker Development Patterns for Operator OS

## Description
Best practices for Docker-based local development and deployment of Operator OS services.

## Operator OS Docker Architecture

Operator OS uses multiple Docker Compose configurations:
- `docker/docker-compose.yml` — Minimal Alpine-based (agent + gateway)
- `docker/docker-compose.full.yml` — Full-featured with Node.js 24 (MCP support)
- `docker/docker-compose.services.yml` — Supporting services (DB, cache)
- `docker-compose.managed.yml` — All-in-one managed deployment
- `docker-compose.yml` (root) — Development stack (web + api + db + redis)

## Dockerfile Patterns

### Go Service (Backend)
```dockerfile
# Base — static binary, no CGO
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/service

# Production — scratch or distroless
FROM alpine:3.21 AS production
RUN adduser -D -u 1001 appuser
COPY --from=builder /bin/service /bin/service
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/bin/service"]
```

### Node.js Service (Frontend / Workers)
```dockerfile
FROM node:20-alpine AS base
WORKDIR /app
RUN apk add --no-cache libc6-compat

FROM base AS deps
COPY package.json package-lock.json* ./
RUN npm ci --legacy-peer-deps

FROM base AS development
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NODE_ENV=development
CMD ["npx", "vite", "--host", "0.0.0.0"]

FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM nginx:alpine AS production
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
```

### Target Selection
```yaml
services:
  web:
    build:
      context: ./web
      target: development  # Switch to "production" for prod builds
```

## Docker Compose Patterns

### Health Checks (Required for Dependencies)
```yaml
services:
  db:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  api:
    depends_on:
      db:
        condition: service_healthy
```

### Volume Mounts for Development
```yaml
services:
  web:
    volumes:
      - ./web:/app              # Source code (hot reload)
      - /app/node_modules       # Preserve container node_modules
```

### Environment Variables
```yaml
services:
  api:
    env_file:
      - .env              # Shared secrets
    environment:
      - ENV=development    # Service-specific overrides
```

## Operator OS Specific Patterns

### Gateway + Agent Profiles
The existing docker-compose files use profiles to select mode:
```bash
docker compose -f docker/docker-compose.yml --profile gateway up
docker compose -f docker/docker-compose.yml run --rm operator-agent
```

### Config Volume Mount
The Go binary reads `config.json` at startup:
```yaml
volumes:
  - ./docker/data/config.json:/root/.operator/config.json
```

### Multi-Arch Builds
Operator OS supports x86_64, ARM64, ARMv7, RISC-V, LoongArch:
```bash
make build-all  # Builds for all supported architectures
```

## Security Checklist
- [x] Non-root user in production images
- [x] No secrets in Dockerfiles (use env vars or mounted files)
- [x] Minimal base images (Alpine preferred, scratch for Go)
- [x] Health checks defined on all services
- [ ] Read-only root filesystem where possible
- [ ] Resource limits in compose (memory, CPU)
- [ ] Network segmentation (frontend vs backend networks)
