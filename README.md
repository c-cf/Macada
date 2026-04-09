<div align="center">

# Macada

**The open-source Managed Agents platform.**

Define agents with custom system prompts, tools, and MCP servers.<br/>
Run them in isolated sandbox environments with full session lifecycle management.

[![CI](https://github.com/cchu-code/managed-agents/actions/workflows/ci.yml/badge.svg)](https://github.com/cchu-code/managed-agents/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](https://nextjs.org/)

[Quick Start](#quick-start) · [Architecture](#architecture) · [Contributing](#development)

</div>

---

## What is Macada?

Macada is a self-hosted implementation of [Anthropic's Managed Agents API](https://platform.claude.com/docs/en/managed-agents/overview). It gives you a full control plane for managing AI agents — define them once, run them anywhere, and inspect every step.

Each session runs inside its own sandboxed container. Events stream in real time. Skills persist across runs. You get a dashboard, a REST API, and complete ownership of your data.

No vendor lock-in. No usage caps. Just your agents, running the way you designed them.

## Features

- **Agent definitions** — configure system prompts, tools, and MCP servers in YAML. Version and update agents without interrupting active sessions.
- **Isolated sandboxes** — every session spawns its own container from `Dockerfile.runtime`. Agents can't interfere with each other or the host.
- **Real-time event streaming** — watch sessions unfold live via SSE. Every tool call, message, and error is captured and queryable.
- **Skills library** — save reusable capabilities across your agent fleet. Skills accumulate and compound over time.
- **Environment management** — separate network policies, secrets, and constraints per environment. Move agents from development to production without code changes.
- **Analytics dashboard** — token usage, latency, session counts, and cost breakdown. Built-in, no third-party required.
- **Full REST API** — everything the dashboard does, the API does too. Automate agent workflows from your own tooling.

---

## Quick Start

**Prerequisites:** Docker & Docker Compose, an Anthropic API key.

```bash
# 1. Clone and configure
git clone https://github.com/cchu-code/managed-agents.git
cd managed-agents
cp .env.example .env
# Edit .env — set ANTHROPIC_API_KEY, ADMIN_SECRET, JWT_SECRET

# 2. Start all services
make up

# 3. Open the dashboard
open http://localhost:3000
```

The backend runs database migrations automatically on startup. The first time you visit, register an account via the UI or bootstrap your first workspace via the admin endpoint.

### Environment variables

| Variable              | Default | Description                                           |
|-----------------------|---------|-------------------------------------------------------|
| `ANTHROPIC_API_KEY`   | —       | **Required.** Your Anthropic API key                  |
| `ADMIN_SECRET`        | —       | **Required.** Secret for the bootstrap admin endpoint |
| `JWT_SECRET`          | —       | **Required.** Secret for signing user login tokens    |
| `BACKEND_PORT`        | `8080`  | Backend API port                                      |
| `FRONTEND_PORT`       | `3000`  | Frontend dashboard port                               |
| `EXTERNAL_DB_PORT`    | `15432` | PostgreSQL port exposed to host (for GUI tools)       |
| `EXTERNAL_REDIS_PORT` | `16379` | Redis port exposed to host (for debugging)            |

---

## Architecture

```
                  Browser
                    │
                    ▼
              ┌───────────┐
              │  Frontend  │  Next.js 15 / React 19
              └─────┬─────┘
                    │ REST / SSE
                    ▼
              ┌───────────┐     ┌───────────────────────┐
              │  Backend   │────▶│  Runtime (per-session) │
              │ Control    │◀────│  Sandbox container     │
              │  Plane     │     └───────────────────────┘
              └──┬────┬────┘
                 │    │
          ┌──────┘    └──────┐
          ▼                  ▼
    ┌──────────┐       ┌──────────┐
    │PostgreSQL│       │  Redis   │
    │    16    │       │    7     │
    └──────────┘       └──────────┘
```

| Layer        | Stack                                       |
|--------------|---------------------------------------------|
| Frontend     | Next.js 15, React 19, Tailwind CSS 4        |
| Backend      | Go, chi router, sqlc, PostgreSQL 16, Redis 7|
| Agent Runtime| Per-session Docker containers               |
| Infra        | Docker Compose, multi-stage builds          |

**Control Plane** — Go API server managing agents, sessions, environments, skills, and events.

**Runtime** — Each session spawns a dedicated container running the agent loop (`Dockerfile.runtime`). Containers are isolated from each other and from the host network.

**Frontend** — Dashboard for creating agents, monitoring sessions, browsing events, and viewing analytics.

---

## Development

**Prerequisites:** [Go](https://go.dev/) 1.21+, [Node.js](https://nodejs.org/) 20+, [Docker](https://www.docker.com/)

### Full stack

```bash
make up       # Build and start all services (Docker)
make down     # Stop all services
```

### Backend only

```bash
make backend-dev          # Run with live reload
make backend-test         # Run tests with race detection
make backend-migrate-up   # Apply database migrations
make backend-tidy         # go mod tidy
```

### Frontend only

```bash
make frontend-install     # Install dependencies
make frontend-dev         # Start dev server at http://localhost:3000
make frontend-build       # Production build
```

### Project structure

```
macada/
├── backend/
│   ├── cmd/
│   │   ├── server/        # Control plane entrypoint
│   │   └── runtime/       # Sandbox runtime entrypoint
│   ├── internal/
│   │   ├── api/           # HTTP handlers & router
│   │   ├── config/        # App configuration
│   │   ├── domain/        # Domain models & repository interfaces
│   │   ├── infra/         # PostgreSQL & Redis implementations
│   │   ├── runtime/       # Agent loop, prompt builder, context management
│   │   ├── sandbox/       # Container orchestration & deployment
│   │   └── service/       # Business logic
│   ├── pkg/               # Shared libraries (SSE, pagination)
│   ├── Dockerfile         # Control plane image
│   └── Dockerfile.runtime # Sandbox runtime image
├── frontend/
│   └── src/
│       ├── app/           # Next.js App Router pages
│       ├── components/    # Shared UI components
│       └── lib/           # API client, types, utilities
├── docker-compose.yml
├── Makefile
└── .env.example
```

---

## License

[MIT](LICENSE)
