# myai-novel-go

[中文说明](./README.md)

`myai-novel-go` is a Go rewrite of the `myai-novel` backend, built for AI-assisted novel writing workflows with a stronger focus on concurrency, operational stability, and deployment ergonomics.

It is built on **Gin + GORM** and provides:

- HTTP API service
- Local CLI tool
- Multi-stage chapter workflows
- SQLite / MySQL support
- mock / OpenAI / Anthropic / custom LLM providers
- Docker Compose for quick local deployment

## What this project is

If you only want the short version:

1. **What it is**: a Go backend for AI novel-writing workflows.
2. **How to run it fast**: copy `.env.example`, then run `make run`.
3. **Where to read more**: the full documentation set currently lives mostly in Chinese under `docs/`, starting from [docs/index.md](./docs/index.md).

## Why this rewrite exists

Compared with the original Node.js backend, this project emphasizes:

1. **Real background concurrency** through goroutines and a bounded worker pool.
2. **More predictable database behavior** with GORM pooling, SQLite WAL, and MySQL support.
3. **Explicit LLM rate limiting and timeouts** for safer long-running workflows.
4. **Cleaner service boundaries** so HTTP, CLI, and workflow execution share the same backend services.

## Who this repository is for

This repository is a good fit if you want to:

- stand up an AI novel-writing backend quickly
- run the full workflow locally with a mock provider first
- manage books, chapters, and settings through HTTP or CLI
- migrate from the original Node backend gradually

## Quick start

### 1. Copy environment variables

```bash
cp .env.example .env
```

The default setup already works for local development:

- `DB_CLIENT=sqlite`
- `LLM_PROVIDER=mock`
- `SERVER_PORT=3000`

### 2. Build and run the server

```bash
make server
make run
```

### 3. Check health

```bash
curl http://127.0.0.1:3000/health
```

## Where to go next

### I want to get it running locally

- [Quick start guide](./docs/getting-started.md) *(Chinese)*
- [Configuration](./docs/configuration.md) *(Chinese)*
- [Deployment](./docs/deployment.md) *(Chinese)*

### I want to understand the workflows

- [Workflows](./docs/workflows.md) *(Chinese)*
- [API overview](./docs/api-overview.md) *(Chinese)*
- [CLI guide](./docs/cli.md) *(Chinese)*

### I want to understand the internals

- [Architecture](./docs/architecture.md) *(Chinese)*
- [Migration from Node](./docs/migration-from-node.md) *(Chinese)*
- [Troubleshooting](./docs/troubleshooting.md) *(Chinese)*

### I want a docs entry page

- [Documentation index](./docs/index.md) *(Chinese)*

## Main capabilities

### HTTP API

The server includes:

- health endpoints
- CRUD for books, chapters, and setting resources
- chapter stage read/write endpoints
- synchronous and asynchronous workflows
- task polling and task termination
- auth and user settings endpoints

### CLI

The `novel` CLI supports:

- database init and checks
- book / chapter / setting resource operations
- chapter stage markdown import/export
- `plan` / `draft` / `review` / `repair` / `approve` workflows

### Workflow stages

The core workflow path is:

```text
plan -> draft -> review -> repair -> approve
```

It supports both synchronous execution and asynchronous task submission.

## Default local experience

The repository is intentionally optimized for a "run first, customize later" experience:

- SQLite by default
- mock LLM provider by default
- default address: `127.0.0.1:3000`
- no real model API key required for the first run

That makes it convenient for local evaluation, development, and integration scripts.

## Docker Compose

To run it with containers:

```bash
make docker-up
```

This starts the service with SQLite + mock provider by default.

## Open-source project files

The repository already includes:

- [Contributing guide](./CONTRIBUTING.md)
- [Security policy](./SECURITY.md)
- [Code of conduct](./CODE_OF_CONDUCT.md)
- [MIT License](./LICENSE)

## Current documentation status

The repository now has a structured documentation system, but most detailed docs are currently written in Chinese.

If you plan to open the project to a broader audience, the next high-value step would be to translate the highest-traffic docs first:

- quick start
- configuration
- workflows
- API overview
