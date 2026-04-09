# Macada

An open-source, self-hosted implementation inspired by [Anthropic's Managed Agents API](https://platform.claude.com/docs/en/managed-agents/overview). Define agents with custom system prompts, tools, and MCP servers, then run them in isolated sandboxed environments with full session lifecycle management.

## Architecture

```
                  Browser
                    │
                    ▼
              ┌───────────┐
              │  Frontend  │  Next.js 15 / React 19
              └─────┬─────┘
                    │
                    ▼
              ┌───────────┐     ┌─────────────────────┐
              │  Backend   │────▶│  Runtime (per-session)│
              │ Control    │◀────│  Sandbox container    │
              │ Plane      │     └─────────────────────┘
              └──┬────┬────┘
                 │    │
          ┌──────┘    └──────┐
          ▼                  ▼
    ┌──────────┐       ┌──────────┐
    │PostgreSQL│       │  Redis   │
    │  16      │       │  7       │
    └──────────┘       └──────────┘
```

- **Control Plane** — Go API server managing agents, sessions, environments, skills, and events
- **Runtime** — Per-session sandboxed container running the agent loop (built from `Dockerfile.runtime`)
- **Frontend** — Dashboard for monitoring agents, sessions, environments, and analytics
- **PostgreSQL** — Persistent storage for all resources
- **Redis** — Pub/sub for real-time SSE event streaming

## Tech Stack

| Layer    | Technology                          |
|----------|-------------------------------------|
| Backend  | Go, chi, PostgreSQL 16, Redis 7     |
| Frontend | Next.js 15, React 19, Tailwind CSS 4|
| Infra    | Docker Compose, multi-stage builds  |

## Prerequisites

- Docker & Docker Compose
- An Anthropic API key

## Quick Start

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env and set your ANTHROPIC_API_KEY

# 2. Start all services
make up

# 3. Open the dashboard
# http://localhost:3000
```

The backend runs database migrations automatically on startup.

## Configuration

All configuration is done via environment variables. See `.env.example`:

| Variable              | Default | Description                                  |
|-----------------------|---------|----------------------------------------------|
| `ANTHROPIC_API_KEY`   | —       | **Required.** Your Anthropic API key         |
| `BACKEND_PORT`        | 8080    | Backend API port                             |
| `FRONTEND_PORT`       | 3000    | Frontend dashboard port                      |
| `EXTERNAL_DB_PORT`    | 15432   | PostgreSQL port exposed to host (for GUI tools) |
| `EXTERNAL_REDIS_PORT` | 16379   | Redis port exposed to host (for debugging)   |

## API

Base URL: `http://localhost:8080/v1`

| Method   | Endpoint                                   | Description              |
|----------|--------------------------------------------|--------------------------|
| `POST`   | `/environments`                            | Create environment       |
| `GET`    | `/environments`                            | List environments        |
| `GET`    | `/environments/{id}`                       | Get environment          |
| `POST`   | `/environments/{id}`                       | Update environment       |
| `DELETE` | `/environments/{id}`                       | Delete environment       |
| `POST`   | `/skills`                                  | Create skill             |
| `GET`    | `/skills`                                  | List skills              |
| `GET`    | `/skills/{id}`                             | Get skill                |
| `POST`   | `/skills/{id}`                             | Update skill             |
| `DELETE` | `/skills/{id}`                             | Delete skill             |
| `POST`   | `/agents`                                  | Create agent             |
| `GET`    | `/agents`                                  | List agents              |
| `GET`    | `/agents/{id}`                             | Get agent                |
| `POST`   | `/agents/{id}`                             | Update agent             |
| `POST`   | `/agents/{id}/archive`                     | Archive agent            |
| `POST`   | `/sessions`                                | Create session           |
| `GET`    | `/sessions`                                | List sessions            |
| `GET`    | `/sessions/{id}`                           | Get session              |
| `POST`   | `/sessions/{id}/archive`                   | Archive session          |
| `POST`   | `/sessions/{id}/events`                    | Send event to session    |
| `GET`    | `/sessions/{id}/events`                    | List session events      |
| `GET`    | `/sessions/{id}/events/stream`             | SSE event stream         |
| `GET`    | `/analytics/usage`                         | Usage analytics          |
| `GET`    | `/analytics/logs`                          | Analytics logs           |

## Development

### Full stack (Docker)

```bash
make up       # Start all services
make down     # Stop all services
```

### Backend only

```bash
cd backend
make dev          # Run with hot reload
make test         # Run tests with race detection
make migrate-up   # Run database migrations
make tidy         # go mod tidy
```

### Frontend only

```bash
cd frontend
npm install
npm run dev       # http://localhost:3000
```

## Project Structure

```
Macada/
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

## License

[MIT](LICENSE)
