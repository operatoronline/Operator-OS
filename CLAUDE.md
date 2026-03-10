# Project: Operator OS

## Overview
Operator OS is an ultra-lightweight, high-performance personal AI Agent framework written in Go. It runs on hardware as inexpensive as $10 with <10MB RAM, bringing continuous intelligence to the edge. The web UI is a React/TypeScript dashboard for managing agents, chat sessions, integrations, billing, and admin controls.

## Tech Stack
- **Backend**: Go 1.25+ (single binary, CGO-free)
- **Frontend**: React 19 + TypeScript + Tailwind CSS v4 + Vite 6
- **State**: Zustand (lightweight stores)
- **Database**: SQLite (embedded, default) / PostgreSQL (optional scaling)
- **Cache**: Redis (optional)
- **Deployment**: Docker Compose, Helm, single binary, Coolify-ready
- **CI**: GoReleaser for multi-arch release builds

## Directory Structure
```
/
├── cmd/operator/           # Go CLI entry point (gateway, agent, onboard)
├── pkg/                    # Go packages (agent, auth, billing, channels, tools…)
├── web/                    # React frontend (Vite + Tailwind v4)
│   ├── src/
│   │   ├── components/     # UI components (layout, chat, agents, billing, admin…)
│   │   ├── pages/          # Route-level pages
│   │   ├── stores/         # Zustand state stores
│   │   ├── services/       # API client + WebSocket manager
│   │   ├── hooks/          # Custom React hooks
│   │   ├── types/          # TypeScript type definitions
│   │   └── styles/         # Additional CSS (hljs themes)
│   └── index.html
├── config/                 # config.example.json
├── docker/                 # Dockerfiles + docker-compose variants
├── deploy/helm/            # Helm chart for Kubernetes
├── workspace/              # Agent identity, soul, memory
├── assets/                 # Logo, GIFs, screenshots
├── docs/                   # Documentation hub
├── services/api/           # Optional containerized API service layer
├── .claude/                # Claude Code skills + commands
├── _start/                 # Starter kit and initialization protocols
├── Makefile                # Build, test, lint, install, Docker targets
├── STATUS.md               # Progress tracker (read first each session)
└── CLAUDE.md               # This file
```

## Quick Start
```bash
# Backend
make deps && make build
./build/operator gateway

# Frontend
cd web && npm install && npm run dev
# Open http://localhost:5173

# Docker (full stack)
docker compose -f docker/docker-compose.full.yml --profile gateway up
```

## Key Commands
```bash
make build           # Build Go binary for current platform
make build-all       # Build for all architectures
make test            # Run Go tests
make lint            # Run golangci-lint
make deps            # Download and verify Go modules
cd web && npm run dev       # Start Vite dev server
cd web && npm run build     # Production build
cd web && npm run typecheck # TypeScript check
cd web && npm run lint      # ESLint
```

## Conventions

### Git
- Use conventional commits: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`
- Update STATUS.md after completing significant work
- Environment variables go in `.env` (never committed)

### Go Backend
- All packages live under `pkg/` — flat, no nested module boundaries
- Tests are colocated (`*_test.go` next to source)
- `CGO_ENABLED=0` by default for static binaries
- Build tags: `stdjson` (default), `whatsapp_native` (optional)
- Config loading: JSON file at `~/.operator/config.json`

### Frontend (web/)
- **Mobile-first**: Design for small screens first, enhance for desktop
- **TypeScript strict**: No `any` types, full type coverage
- **Zustand stores**: One store per domain (chatStore, authStore, billingStore…)
- **Lazy loading**: All pages use `React.lazy()` for code splitting
- **Accessibility**: WCAG 2.1 AA — focus management, ARIA attributes, skip links

## Design System (Non-Negotiable)

### Color Model
- Use **OKLCH** color space exclusively for perceptual harmony
- **80% monochrome** (black, white, grays), **20% functional color**
- One primary hue — applied consistently for key actions and focus states
- Status colors: red (error), green (success), yellow (warning), blue (info)
- Never use colored drop shadows or gradient backgrounds

### Typography
- Primary: DM Sans (400, 500, 600, 700)
- Code: JetBrains Mono (400, 500)

### Spacing & Layout
- 4px base spacing scale with consistent vertical rhythm
- Flexbox for responsive patterns — **no flex-wrap**
- Overflow: truncation, icon-only adaptation, or horizontal scroll
- 44px minimum touch targets on mobile
- Safe area insets for notch devices

### Hard Constraints
- Never: colored drop shadows
- Never: gradient backgrounds (no "AI-style" purple-blue blends)
- Never: redundant or nested containers around the same data
- Never: decorative color outside the 20% functional allocation
- Never: multiple accent colors in the same context
- Never: flex-wrap — use truncation, icon-only states, or horizontal scroll

### Components
- Floating navbar with icon-only logo on mobile
- FAB for quick actions
- User avatar dropdown for account-level settings
- Glass morphism for floating UI elements (bottom tabs, dropdowns)
- Progressive disclosure for optional complexity

## Environment Setup
1. Copy `.env.example` to `.env`
2. Edit `~/.operator/config.json` (or `config/config.example.json` as template)
3. Run `make build && ./build/operator gateway` or `docker compose up`
4. Frontend: `cd web && npm install && npm run dev`

## Session Guidelines
1. Read STATUS.md first to understand current state
2. Work on one focused task per session
3. Commit frequently with clear messages
4. Update STATUS.md before ending session
5. Test changes with `make test` and `cd web && npm run typecheck` before committing
