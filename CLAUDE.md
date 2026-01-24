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
| Frontend | React 18, TypeScript, Vite 7, Tailwind CSS v4 |
| Database | SQLite (default), MySQL, PostgreSQL |
| API | Protocol Buffers with buf, dual gRPC + Connect |

## Commands (Run on EC2)

All commands below are run on the EC2 dev server after SSH'ing in.

### Backend
```bash
cd /home/ec2-user/memos-ai

# Development
go run ./cmd/memos/ --mode dev --data ./data

# Build
go build -o memos ./cmd/memos/

# Tests
go test ./...                              # All tests
go test ./store/...                        # Store layer tests
go test -v ./plugin/llm/...                # LLM plugin tests
go test -v -run TestMemoCreation ./store/test/...  # Single test

# Lint
golangci-lint run

# Format imports
goimports -w .
```

### Frontend
```bash
cd /home/ec2-user/memos-ai/web

pnpm install                     # Install dependencies
pnpm dev                         # Development server (port 3001, proxies to :8081)
pnpm build                       # Production build
pnpm lint                        # TypeScript + Biome check
pnpm lint:fix                    # Auto-fix lint issues
pnpm format                      # Format with Biome
pnpm release                     # Build to server/router/frontend/dist
```

### Protocol Buffers
```bash
cd /home/ec2-user/memos-ai/proto
buf generate                     # Regenerate Go and TypeScript from .proto
buf lint                         # Lint proto definitions
buf breaking --against .git#main # Check for breaking changes
```

## Architecture

### Backend Structure

```
cmd/memos/              # Entry point (Cobra CLI, profile setup)
server/
├── server.go           # Echo HTTP server, healthz, background runners
├── auth/               # Authentication (JWT, PAT, session)
└── router/
    └── api/v1/         # gRPC + Connect service implementations
        ├── v1.go               # Service registration, gateway + Connect setup
        ├── acl_config.go       # Public endpoints whitelist
        ├── connect_interceptors.go  # Auth, logging, recovery
        └── *_service.go        # Individual services (memo, user, etc.)
store/
├── driver.go           # Driver interface (all DB operations)
├── store.go            # Store wrapper with caching (TTL 10min, max 1000 items)
└── db/                 # Driver implementations (sqlite/, mysql/, postgres/)
plugin/
├── llm/                # LLM provider abstraction (OpenAI, Anthropic, Gemini, Ollama)
│   ├── provider.go     # Interface + factory
│   ├── tag_service.go  # Tag suggestions (15min cache, 60 req/min rate limit)
│   └── crypto.go       # AES-256-GCM encryption for API keys
└── storage/            # S3, local storage backends
proto/                  # Generated protobuf code
```

### Frontend Structure (`web/`)

```
src/
├── components/         # React components
├── contexts/           # React Context (AuthContext, ViewContext, MemoFilterContext)
├── hooks/              # React Query hooks (useMemoQueries, useUserQueries, etc.)
├── lib/
│   ├── query-client.ts # React Query v5 config (staleTime: 30s, gcTime: 5min)
│   └── connect.ts      # Connect RPC client setup
├── pages/              # Page components
└── types/proto/        # Generated TypeScript from .proto
```

### Key Patterns

- **Store Interface:** All DB operations go through `store.Driver` interface
- **Dual API Protocol:** Connect RPC for browsers (`/memos.api.v1.*`), gRPC-Gateway for REST (`/api/v1/*`)
- **State Management:** React Query v5 for server state, React Context for client state
- **Authentication:** JWT Access Tokens (15-min) + Personal Access Tokens (long-lived), both via `Authorization: Bearer`

### Data Flow

```
Frontend (React Query) → Connect RPC → Service Layer → Store Interface → Database Driver
```

## Key Workflows

### Adding a New API Endpoint

1. Define in `proto/api/v1/*_service.proto` (messages + RPC method)
2. Regenerate: `cd proto && buf generate`
3. Implement in `server/router/api/v1/*_service.go`
4. If public, add to `server/router/api/v1/acl_config.go`
5. Create frontend hook in `web/src/hooks/use*Queries.ts`

### Database Schema Changes

1. Create migrations in `store/migration/{sqlite,mysql,postgres}/{version}/NN__description.sql`
2. Update `store/migration/{driver}/LATEST.sql`
3. If new table, add methods to `store/driver.go` and implement in each driver

## Production Deployment

**CRITICAL: Memos runs as a NATIVE BINARY with EMBEDDED FRONTEND, not Docker.**

The Go binary embeds frontend files via `//go:embed`. This means:
- Frontend changes require rebuilding the Go binary
- `pnpm release` alone is NOT enough - you must also rebuild Go

| Property | Value |
|----------|-------|
| **URL** | https://memo.formsight.ai |
| **Binary** | `/home/ec2-user/memos-ai/memos` |
| **Data** | `/home/ec2-user/memos-ai/data` |
| **Port** | 5230 (nginx proxies from 443) |

### Full Deployment Workflow (Frontend + Backend)

```bash
# 1. Sync local changes to EC2
rsync -avz --checksum --exclude='.git' --exclude='node_modules' --exclude='data' \
  -e "ssh -i ~/.ssh/formsight-key.pem" \
  /path/to/local/memos-ai/ \
  ec2-user@54.92.152.180:/home/ec2-user/memos-ai/

# 2. SSH to EC2
ssh -i ~/.ssh/formsight-key.pem ec2-user@54.92.152.180
cd /home/ec2-user/memos-ai

# 3. Build frontend (outputs to server/router/frontend/dist/)
cd web && pnpm install && pnpm release && cd ..

# 4. Build Go binary (embeds frontend files)
GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go build -o memos-new ./cmd/memos/

# 5. Deploy (stop old, swap binary, start new)
killall -9 memos
cp memos memos-backup
cp memos-new memos
nohup ./memos --port 5230 --data ./data > memos.log 2>&1 &

# 6. Verify
sleep 3 && ps aux | grep '[m]emos'
```

### Frontend-Only Changes (Still Requires Go Rebuild!)

Even for CSS/JS changes, you MUST rebuild the Go binary:
```bash
cd web && pnpm release && cd ..
GOTOOLCHAIN=auto go build -o memos-new ./cmd/memos/
# Then deploy as above
```

### Check Running Process
```bash
ps aux | grep memos | grep -v grep
ss -tlnp | grep 5230
```

### View Logs
```bash
tail -f /home/ec2-user/memos-ai/memos.log
```

## Key Files

| File | Purpose |
|------|---------|
| `memos-ai/AGENTS.md` | Comprehensive codebase guide (600+ lines) - **read this first for deep work** |
| `memos-ai/proto/api/v1/*.proto` | API definitions |
| `memos-ai/store/driver.go` | Database driver interface |
| `memos-ai/server/router/api/v1/acl_config.go` | Public endpoints whitelist |
| `memos-ai/plugin/llm/provider.go` | LLM provider interface |

## Common Issues & Lessons Learned

### Frontend Changes Not Showing After Deploy

**Root Cause:** The Go binary embeds frontend files at compile time via `//go:embed dist/*` in `server/router/frontend/frontend.go`.

**Solution:** After running `pnpm release`, you MUST rebuild the Go binary:
```bash
cd web && pnpm release && cd ..
GOTOOLCHAIN=auto go build -o memos-new ./cmd/memos/
# Then deploy the new binary
```

**How to verify:** Check Network tab in browser DevTools - the JS filename hash should change after deploying.

### Go Build Fails with "go.mod requires go >= 1.25"

```bash
# Always use GOTOOLCHAIN when building
GOTOOLCHAIN=auto GOSUMDB=sum.golang.org go build -o memos ./cmd/memos/
```

### rsync Creates Files in Wrong Directories

When syncing individual files, rsync may place them in unexpected locations. Always sync entire directories:
```bash
# GOOD - sync directory
rsync -avz ... /local/path/to/dir/ user@host:/remote/path/to/dir/

# BAD - may create stray files
rsync -avz ... /local/file1.tsx /local/file2.tsx user@host:/remote/path/
```

### API Keys Not Working After Save

**Root Cause:** Backend was overwriting stored API keys with empty strings when users saved settings without re-entering keys.

**Fix Applied:** `preserveExistingAPIKeys()` function in `server/router/api/v1/instance_service.go`.

### Anthropic Provider Not Registered

If AI features show "LLM not configured" with Anthropic keys:
- Verify `plugin/llm/anthropic.go` exists
- Verify `plugin/llm/config.go` registers Anthropic provider in `LoadFromProto()`

## Related Resources

- Upstream: https://github.com/usememos/memos
- Reference AI PR: https://github.com/usememos/memos/pull/5022
- AI Discussion: https://github.com/orgs/usememos/discussions/5000
