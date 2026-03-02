# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Shadiff 是一个影子流量语义对拍工具，用于跨框架/跨语言的 API 迁移验证。通过黑盒录制-回放-对拍三段式流程，验证新旧 API 行为是否一致。

## Build & Development Commands

```bash
go build -o shadiff .         # 编译
go run . version               # 运行 version 命令
go run . record --help          # 查看 record 命令帮助
go test ./...                   # 运行所有测试
```

## Tech Stack

- **Language:** Go 1.24
- **CLI Framework:** cobra
- **Storage:** 文件系统 (JSONL), ~/.shadiff/
- **Logging:** slog + 日志轮转

## Architecture

### 核心工作流

```
record (录制) → replay (回放) → diff (对拍) → report (报告)
```

### 包结构

| 包 | 职责 |
|---|---|
| `cmd/` | CLI 命令 (cobra) |
| `internal/model/` | 核心数据模型: Session, Record, SideEffect, DiffResult |
| `internal/config/` | 配置管理 (~/.shadiff/config.json) |
| `internal/logger/` | 结构化 slog 日志 + 轮转 |
| `internal/capture/` | HTTP 反向代理 + DB 协议代理采集 |
| `internal/capture/dbhook/` | 数据库协议代理 (MySQL/PostgreSQL/MongoDB) |
| `internal/storage/` | JSONL 文件存储 |
| `internal/replay/` | 流量回放引擎 |
| `internal/diff/` | 语义对拍引擎 |
| `internal/reporter/` | 报告生成 (terminal/JSON/HTML) |

### 数据存储

所有持久数据存储在 `~/.shadiff/`:
- `config.json` — 全局配置
- `sessions/{id}/session.json` — 会话元数据
- `sessions/{id}/records.jsonl` — 录制记录 (JSONL 流式)
- `sessions/{id}/replay-records.jsonl` — 回放记录
- `sessions/{id}/diff-results.json` — 对拍结果

## Key Conventions

- Go 代码遵循标准 Go 惯例 (gofmt, effective Go)
- 注释使用中文，标识符使用英文
- 配置管理参考 starxo 的 config.Store 模式 (线程安全 JSON 读写)
- 日志使用 slog + 日志轮转，参考 starxo logger 模式

## Engineering Conventions

### Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`

Scopes: `model`, `config`, `capture`, `dbhook`, `storage`, `replay`, `diff`, `reporter`, `logger`, `cmd`

### Development Workflow

1. **Plan** — 在 `plan/` 创建计划文档
2. **Implement** — 在 feature 分支上实现
3. **Verify** — 端到端测试
4. **Commit** — Conventional Commits 格式

### Testing

- Go tests: `*_test.go` alongside source
- Test file naming: `<source>_test.go`
