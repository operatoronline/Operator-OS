# Command: /new-service

Create a new Docker-ready microservice for Operator OS with standard structure.

## Usage
```
/new-service [service-name] [language]
```

## Arguments
- `service-name`: Name of the service (kebab-case, e.g. `webhook-relay`)
- `language`: Programming language (`go` | `node` | `python`)

## Example
```
/new-service webhook-relay go
/new-service analytics-worker node
```

## What It Creates

### Directory Structure
```
services/[service-name]/
├── src/              # (node/python) or cmd/ (go)
│   └── main.[ext]
├── tests/
├── Dockerfile        # Multi-stage (development + production targets)
├── CLAUDE.md         # Service-specific context for Claude
└── README.md
```

### Files Generated

**Dockerfile** — Multi-stage build with:
- `development` target with hot reload (air for Go, nodemon for Node, uvicorn for Python)
- `production` target with minimal Alpine image
- Health check endpoint at `/health`
- Non-root user in production
- Compatible with Operator OS docker-compose patterns

**CLAUDE.md** — Service context including:
- Purpose and responsibility
- Tech stack and dependencies
- API endpoint table
- Environment variables
- Testing and development commands

### Docker Compose Integration
Also updates the root `docker-compose.yml` to add:

```yaml
services:
  [service-name]:
    build:
      context: ./services/[service-name]
      target: development
    volumes:
      - ./services/[service-name]:/app
    environment:
      - ENV=development
    networks:
      - default
```

### Operator OS Conventions
- Service must expose a `/health` endpoint returning `{"status": "ok"}`
- Use structured JSON logging (zerolog for Go, pino for Node)
- Config via environment variables, not files
- All services communicate over the `app-network` Docker network
- Service name in docker-compose matches the directory name

## Post-Creation Steps
1. Update `services/[service-name]/CLAUDE.md` with actual API design
2. Run `docker compose build [service-name]` to build the new service
3. Implement the service logic
4. Add tests
5. Update STATUS.md with new service entry
