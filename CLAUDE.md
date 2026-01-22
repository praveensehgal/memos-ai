# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Memos AI is a fork of [usememos/memos](https://github.com/usememos/memos) with AI-powered features. The main codebase is in `memos-ai/`.

**AI Features:** Semantic search, auto-tagging, code intelligence, BYOK (OpenAI, Anthropic, Gemini, Ollama)

## Development Environment

**All development is done on EC2, not locally.** The local machine is only used for running Claude Code.

| Property | Value |
|----------|-------|
| **Server** | 54.92.152.180 (us-east-1) |
| **Instance** | m7i-flex.large (2 vCPU, 8GB RAM) |
| **Code Path** | `/home/ec2-user/memos-ai` |
| **SSH Key** | `~/.ssh/formsight-key.pem` |

### Connect to Dev Server
```bash
ssh -i ~/.ssh/formsight-key.pem ec2-user@54.92.152.180
cd /home/ec2-user/memos-ai
```

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25, Echo v4, Connect RPC |
| Frontend | React 18, TypeScript, Vite 7, Tailwind CSS |
| Database | SQLite (default), MySQL, PostgreSQL |
| API | Protocol Buffers with buf, dual gRPC + Connect |

## Commands (Run on EC2)

All commands below are run on the EC2 dev server after SSH'ing in.

### Backend
```bash
cd /home/ec2-user/memos-ai

# Development (hot reload with air)
go run ./cmd/memos/ --mode dev --data ./data

# Build
go build -o memos ./cmd/memos/

# Tests
go test ./...
go test ./store/...              # Store layer tests
go test -v ./plugin/llm/...      # LLM plugin tests

# Lint
golangci-lint run
```

### Frontend
```bash
cd /home/ec2-user/memos-ai/web

pnpm install                     # Install dependencies
pnpm dev                         # Development server (port 3001)
pnpm build                       # Production build
pnpm lint                        # TypeScript + Biome check
pnpm lint:fix                    # Auto-fix lint issues
pnpm format                      # Format with Biome
```

### Protocol Buffers
```bash
cd /home/ec2-user/memos-ai/proto
buf generate                     # Regenerate from .proto files
buf lint                         # Lint proto definitions
```

### Release Build
```bash
cd /home/ec2-user/memos-ai/web
pnpm release                     # Build to server/router/frontend/dist
```

## Architecture

### Backend (`memos-ai/`)

```
cmd/memos/          # Entry point
server/             # gRPC + Connect service implementations
  ├── route/        # HTTP routes (health, RSS, robots.txt)
  └── service/      # Service layer (memo_service.go, user_service.go)
store/              # Database abstraction (Driver interface)
  ├── db/           # Driver implementations (sqlite/, mysql/, postgres/)
  └── *.go          # Store interfaces (memo.go, user.go, tag.go)
plugin/
  ├── llm/          # LLM provider abstraction
  │   ├── provider.go       # Interface + factory
  │   ├── openai.go         # OpenAI implementation
  │   ├── anthropic.go      # Anthropic implementation
  │   ├── gemini.go         # Google Gemini
  │   ├── ollama.go         # Local Ollama
  │   └── tag_service.go    # Tag suggestions with caching
  └── storage/      # S3, local storage backends
internal/           # Internal utilities
proto/              # Generated protobuf code
```

### Frontend (`memos-ai/web/`)

```
src/
  ├── components/   # React components
  │   ├── kit/      # UI primitives (Button, Dialog)
  │   └── memo*.tsx # Memo-related components
  ├── pages/        # Route pages
  ├── store/        # Zustand stores
  ├── hooks/        # React Query hooks + custom hooks
  └── grpcweb/      # Generated Connect client
```

### Key Patterns

- **Store Interface:** All DB operations go through `store.Driver` interface, implementations in `store/db/`
- **Dual API Protocol:** Services implement both gRPC and Connect RPC via `connectrpc.com/connect`
- **State Management:** React Query v5 for server state, Zustand for client state
- **LLM Abstraction:** `plugin/llm/Provider` interface with factory pattern, API keys encrypted with AES-256-GCM

### Data Flow

```
Frontend (React Query) → Connect RPC → Service Layer → Store Interface → Database Driver
```

## Production Deployment

Production runs as a Docker container on the same EC2 instance as development.

| Property | Value |
|----------|-------|
| **URL** | https://memo.formsight.ai |
| **Container** | `memos` |
| **Data Volume** | /home/ec2-user/memos-data:/var/opt/memos |

### Container Management
```bash
docker logs memos -f            # View logs
docker restart memos            # Restart

# Update to latest
docker pull ghcr.io/praveensehgal/memos-ai:latest
docker stop memos && docker rm memos
docker run -d --name memos --restart unless-stopped \
  -p 5230:5230 \
  -v /home/ec2-user/memos-data:/var/opt/memos \
  ghcr.io/praveensehgal/memos-ai:latest
```

### Nginx & SSL
- Config: `/etc/nginx/conf.d/memo.formsight.ai.conf`
- Cert: `/etc/letsencrypt/live/memo.formsight.ai/`

## Key Files

| File | Purpose |
|------|---------|
| `memos-ai/AGENTS.md` | Comprehensive codebase guide (600+ lines) |
| `memos-ai/docs/PRD.md` | Product requirements |
| `memos-ai/docs/ARCHITECTURE.md` | Technical architecture |
| `memos-ai/proto/api/v1/*.proto` | API definitions |
| `memos-ai/store/memo.go` | Memo store interface |
| `memos-ai/plugin/llm/provider.go` | LLM provider interface |
| `memos-ai/plugin/llm/tag_service.go` | Tag suggestion service |

## Sprint Status

- **Sprint 1 (Completed):** CI/CD, Docker builds, LLM provider layer, API key encryption, tag suggestions (MEMOS-1 through MEMOS-9)
- **Sprint 2 (In Progress):**
  - MEMOS-22: AI Settings page (completed) - Admin UI for LLM provider selection, API keys, feature toggles
  - MEMOS-10: Tag suggestion UI component (not started) - Show AI suggestions in memo editor
  - Server integration needed to wire `plugin/llm` to API endpoints

## Related Resources

- Upstream: https://github.com/usememos/memos
- Reference AI PR: https://github.com/usememos/memos/pull/5022
- AI Discussion: https://github.com/orgs/usememos/discussions/5000
