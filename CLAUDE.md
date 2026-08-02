# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a personal polyglot workspace with four language ecosystems: **Go** (primary backend framework), **C++** (low-level data structures, smart pointers, network libraries), **Python** (AI/LLM agents, web scraping), and **JavaScript** (Node.js worker patterns).

## Go — Primary Development Area

The Go module is at `go/` (GOPATH: `/Users/sujiahong/workspace/go`, GOROOT: `/usr/local/go`). Module path is `go.local`.

### Build & Run

```bash
cd go && go run main_go.go
```

### Test

```bash
cd go && go test ./...                          # all tests
cd go && go test ./su_net/...                   # single package
cd go && go test -run TestTcpNetHandler -v ./su_net/...
```

### Architecture: su_* packages

The Go code is a layered backend framework, built bottom-up:

**Infrastructure layer** (no dependencies on other `su_*` packages):
- `su_errors` — Typed error with `Code` (gRPC-style), retryable flag, `Wrap`/`Unwrap` support
- `su_log` — Wraps `go.uber.org/zap` with auto caller info; dual-writes to stdout and file
- `su_context` — Context helpers for request ID, trace ID, user ID, and concurrency-safe metadata
- `su_util` — Sharded goroutine pool (`GoPool`), util functions

**Data Access layer** (`su_da/`):
- `su_da/kafka` — Raw Kafka consumer/producer (sarama)
- `su_da/redis` — Redis client (redigo)
- `su_da/su_sql` — MySQL via sqlx

**Middleware layer**:
- `su_mq` — Message queue abstraction with retry, dead letter, idempotency, metrics hooks; wraps `su_da/kafka`
- `su_net` — Network protocol: 24-byte binary header (`PackLen|PackId|RouteId|HeadUuid`), TCP/WebSocket clients and servers, handler dispatch with 3 modes (manual response, auto response, one-way). Built on [gnet](https://github.com/panjf2000/gnet) for high-performance TCP.
- `su_config` — Environment and config file loading
- `su_metrics` — Metrics key/value reporting
- `su_balance` — Load balancing (hash, random, round-robin)

**Application layer**:
- `su_app` — Module-based application lifecycle: `App` with `Register(Module)` / `Start` / `Stop`. Modules are started in registration order, stopped in reverse. Pre-built modules for config, log, MySQL, Redis, Kafka MQ, metrics.

**Distributed components** (`distributed/`):
- `circuit` — Circuit breaker
- `health` — Health checking
- `lb` — Load balancer
- `limit` — Rate limiting
- `registry` — Service registry
- `rpc` — RPC with JSON codec over custom transport
- `transport` — Transport abstraction

**Standalone Go programs**:
- `examples/` — Runnable network demos: `gnet_tcp_simple`, `gnet_tcp_pressure`, `tcp_net_pressure`, `ws_net_pressure` (load/pressure tests)
- `mcp_redis_server/` — MCP (Model Context Protocol) server exposing Redis
- `deepseek/` — DeepSeek API client (models, chat completions)
- `geo/` — 2D geometry utilities (points, crosspoints)
- `spider/` — Commodity data spider
- `cgo_demo/` — CGo hello-world example
- `proto/` — Protobuf definitions (e.g., `Test` message)

### Key Patterns

- **Handler Registration**: In `su_net`, register handlers by packet ID — `RegisterRequestResponseHandler(rqID, rsID, handler)` auto-replies, `RegisterOneWayHandler(ID, handler)` sends no reply, `RegisterManualResponseHandler(rqID, rsID, handler)` lets the handler decide
- **GoPool sharding**: `SendTask(routeId, fn)` distributes work by `routeId % numWorkers` for deterministic routing
- **Network config**: Constants in `go/su_net/net_config.go` (pool workers, queue sizes, close timeout)

## C++

Header-only libraries and test programs. Key directories:

- `cpp/base/` — Intrusive reference counting: `BaseRef` (atomic refcount, `delete this` on zero), `IntrusivePtr<T>` (full RAII, move semantics, dynamic_cast support), `NormalReleasePtr<T>` (external release function)
- `cpp/data_structure/` — Skip list, Redis-style zset, hash map, circular list, linked list head (Linux kernel style)
- `cpp/net/` — Epoll-based TCP server with non-blocking sockets, event loop, connection management
- `cpp/multi_thread/` — Thread pool, task pool
- `cpp/toolbox/` — File, string, time, UUID utilities
- `cpp/spdlog/` — Vendored spdlog logging library

Build is manual: individual test files are compiled directly (e.g., `g++ -std=c++17 test_intrusive_ptr.cpp -o test_intrusive_ptr`).

## Python

AI/LLM experiments and web scraping:

- `python/HelloAgent/` — Custom LLM agent framework with tools, core loop, and LLM abstraction
- `python/LangGraph/` — LangGraph/LangChain study examples
- `python/dzdp/` — Dianping web scraper
- `python/pchong/` — Multi-site crawler framework (hotels, e-commerce) with requirements management

## JavaScript

Node.js Worker Threads pattern: `main_exec.js` creates a `config_worker.js` Worker that reloads game config on a 5-minute interval.

## Other

- `day_draft/` — Daily dated scratch/draft notes (one directory per date)
- `.claude/settings.local.json` — Project permission allowlist (currently allows `Bash(go run *)`)

## Docker

Development container: `sujiahong/centos7-cpp-dev:v6` with Redis (6379) and MySQL (3306) exposed. Start with:
```bash
docker run -itd --name="dev-os" -p 8379:6379 -p 6306:3306 -v /Users/sujiahong/workspace/import:/workspace --privileged=true --platform linux/amd64 sujiahong/centos7-cpp-dev:v6 /usr/sbin/init
```
