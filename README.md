# Memos AI

A fork of [usememos/memos](https://github.com/usememos/memos) with AI-powered features for developers.

[![Live Instance](https://img.shields.io/badge/🚀-memo.formsight.ai-blue?style=flat-square)](https://memo.formsight.ai)
[![Upstream](https://img.shields.io/badge/upstream-usememos/memos-green?style=flat-square)](https://github.com/usememos/memos)

## Overview

Memos AI extends the excellent [Memos](https://github.com/usememos/memos) note-taking application with AI capabilities. Bring your own LLM API keys and unlock semantic search, intelligent tagging, and code-aware features.

**What's Different from Upstream Memos?**

| Feature | Memos AI | Upstream Memos |
|---------|----------|----------------|
| **Semantic Search** | ✅ Find by intent, not just keywords | ❌ Keyword search only |
| **Auto-Tagging** | ✅ AI suggests tags based on content | ❌ Manual tagging |
| **Code Intelligence** | ✅ Auto-detect languages, syntax highlighting | ⚠️ Basic highlighting |
| **BYOK** | ✅ OpenAI, Anthropic, Gemini, Ollama | ❌ N/A |

## AI Features

- **🔍 Semantic Search** — Find memos by meaning, not just exact keywords
- **🏷️ Auto-Tagging** — AI analyzes content and suggests relevant tags
- **💻 Code Intelligence** — Automatic language detection and enhanced syntax highlighting
- **🔑 Bring Your Own Key (BYOK)** — Use your preferred LLM provider:
  - OpenAI (GPT-4, GPT-3.5)
  - Anthropic (Claude)
  - Google (Gemini)
  - Ollama (local models)

## Core Memos Features

All the features you love from the original Memos:

- **🔒 Privacy-First** — Self-hosted, zero telemetry, your data stays yours
- **📝 Markdown Native** — Full markdown support with plain text storage
- **⚡ Fast** — Go backend + React frontend, optimized for performance
- **🐳 Easy Deployment** — Docker, binaries, or build from source
- **🔗 Developer-Friendly** — REST and gRPC APIs

## Quick Start

### Docker

```bash
docker run -d \
  --name memos-ai \
  -p 5230:5230 \
  -v ~/.memos:/var/opt/memos \
  ghcr.io/praveensehgal/memos-ai:latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/praveensehgal/memos-ai.git
cd memos-ai

# Build backend
go build -o memos ./cmd/memos/

# Build frontend
cd web && pnpm install && pnpm run release && cd ..

# Run
./memos --data /path/to/data
```

## Documentation

| Document | Description |
|----------|-------------|
| [PRD.md](./docs/PRD.md) | Product Requirements — features, user stories, timeline |
| [ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Technical architecture, data models, system design |
| [API.md](./docs/API.md) | API reference for AI endpoints |
| [SETUP.md](./docs/SETUP.md) | Installation and configuration guide |
| [USER-GUIDE.md](./docs/USER-GUIDE.md) | End-user documentation |

## Tech Stack

- **Backend:** Go 1.25+
- **Frontend:** React, TypeScript, Vite, Tailwind CSS
- **Database:** SQLite (default), MySQL, PostgreSQL
- **API:** Connect RPC (gRPC-compatible)

## Development

```bash
# Backend development
go run ./cmd/memos/ --mode dev --data ./data

# Frontend development
cd web
pnpm install
pnpm run dev
```

## Roadmap

| Phase | Focus |
|-------|-------|
| Sprint 1 | Infrastructure & LLM Integration |
| Sprint 2 | Auto-Tagging, Semantic Search, Code Intelligence |
| Sprint 3 | UI/UX Polish |
| Sprint 4-6 | Beta Testing & Launch |

## Acknowledgments

This project is built on top of [usememos/memos](https://github.com/usememos/memos), an excellent open-source note-taking application. Huge thanks to the Memos team and community for creating such a solid foundation.

## License

MIT License — same as upstream Memos.

---

**[Live Instance](https://memo.formsight.ai)** • **[Documentation](./docs/)** • **[Upstream Memos](https://github.com/usememos/memos)**
