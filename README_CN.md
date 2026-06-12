# Shadiff - 影子流量语义对拍工具

[English](README.md)

[![CI](https://github.com/zzqDeco/shadiff/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/zzqDeco/shadiff/actions/workflows/ci.yml)

## 项目简介

Shadiff 是一个影子流量语义对拍工具，用于跨框架/跨语言的 API 迁移验证。通过**录制-回放-对拍**三段式流程：以反向代理方式透明采集老 API 的完整行为（输入、输出、数据库副作用），然后将相同的输入回放到新 API，语义级比较两边的行为差异。

## 核心特性

- **HTTP 反向代理录制** — 通过 `httputil.ReverseProxy` 透明采集流量，记录完整的请求/响应对及时间信息
- **数据库协议代理** — TCP 级别黑盒采集，支持 MySQL（COM_QUERY）、PostgreSQL（Simple/Extended Query）、MongoDB（OP_MSG Wire Protocol）和 Redis（RESP 命令）
- **并发回放引擎** — 基于 Worker Pool 的回放，支持可配置的并发数，请求变换（host/header 替换）
- **语义级 JSON 对比** — 递归结构化比较，支持路径追踪（如 `body.data.items[0].name`）
- **可配置规则系统** — 支持忽略时间戳、UUID、数值容差、数组顺序等，通过 YAML 规则配置
- **多格式报告** — 终端彩色输出、JSON、HTML 报告，附带汇总统计
- **会话管理** — 完整的会话生命周期管理，JSONL 流式存储

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25 | 主语言 |
| Cobra | v1.9 | CLI 框架 |
| slog | 标准库 | 结构化日志 + 日志轮转 |
| JSONL | - | 流式记录存储 |

## 项目结构

```
shadiff/
├── main.go                            # CLI 入口
├── go.mod                             # Go 1.25 模块
├── CLAUDE.md                          # 开发者指南
├── cmd/                               # CLI 命令
│   ├── root.go                        # Cobra 根命令，全局 flags
│   ├── record.go                      # shadiff record
│   ├── record_stop.go                 # shadiff record stop
│   ├── record_status.go              # shadiff record status
│   ├── replay.go                      # shadiff replay
│   ├── diff.go                        # shadiff diff
│   ├── report.go                      # shadiff report
│   ├── session.go                     # shadiff session (list/show/delete)
│   ├── doctor.go                      # shadiff doctor
│   └── version.go                     # shadiff version
├── internal/
│   ├── dbtype/                        # 支持的 DB 代理类型注册表
│   ├── model/                         # 核心数据模型
│   │   ├── session.go                 # 录制会话
│   │   ├── record.go                  # 单条行为记录（请求+响应+副作用）
│   │   ├── request.go                 # HTTP 请求/响应模型
│   │   ├── sideeffect.go             # 副作用模型（DB 操作、外部调用）
│   │   └── diff.go                    # 差异结果模型
│   ├── config/                        # 配置管理
│   │   ├── config.go                  # 配置类型定义 + DefaultConfig()
│   │   └── store.go                   # JSON 文件存储（~/.shadiff/config.json）
│   ├── capture/                       # 流量采集层
│   │   ├── proxy.go                   # HTTP 反向代理（httputil.ReverseProxy）
│   │   ├── recorder.go               # 统一录制器，组装 Record 并持久化
│   │   └── dbhook/                    # 数据库协议代理
│   │       ├── hook.go                # DBHook 接口定义
│   │       ├── tcp_proxy.go           # 共享透明 TCP 代理生命周期
│   │       ├── mysql.go               # MySQL 协议代理（COM_QUERY 解析）
│   │       ├── postgres.go            # PostgreSQL 协议代理（Simple/Extended Query）
│   │       ├── mongo.go               # MongoDB 协议代理（OP_MSG Wire Protocol）
│   │       └── redis.go               # Redis 协议代理（RESP 命令解析）
│   ├── storage/                       # 存储层
│   │   ├── store.go                   # SessionStore/RecordStore/DiffStore 接口
│   │   └── filestore.go              # 文件系统实现（JSONL）
│   ├── replay/                        # 回放引擎
│   │   ├── engine.go                  # 回放编排器
│   │   ├── worker.go                  # 并发 Worker Pool
│   │   └── transform.go              # 请求变换（host/header 替换）
│   ├── diff/                          # 语义对拍引擎
│   │   ├── engine.go                  # 对拍编排器，按序号配对记录
│   │   ├── sideeffects.go             # 副作用 comparer 注册表
│   │   ├── json.go                    # JSON 结构化递归 diff
│   │   ├── db.go                      # SQL 数据库对比（MySQL/PostgreSQL）
│   │   ├── mongo.go                   # MongoDB 操作对比
│   │   ├── redis.go                   # Redis 命令对比
│   │   └── rules.go                   # 对拍规则 + 内置匹配器
│   ├── reporter/                      # 报告生成
│   │   ├── reporter.go                # Reporter 接口 + 工厂
│   │   ├── terminal.go                # 终端彩色输出
│   │   ├── json.go                    # JSON 格式
│   │   └── html.go                    # HTML 报告（内嵌模板）
│   ├── daemon/                        # 守护进程管理
│   │   ├── pidfile.go                 # PID 文件读写/检查
│   │   ├── process_unix.go            # Unix 进程分离 + 信号
│   │   └── process_windows.go         # Windows 进程分离 + 信号
│   └── logger/                        # 结构化日志
│       └── logger.go                  # slog + 日志轮转
├── plan/                              # 开发路线图
└── logs/                              # 运行日志（gitignored）
```

## 快速开始

### 环境要求

- **Go** >= 1.25

### 安装

```bash
go install github.com/zzqDeco/shadiff@latest
```

或从源码构建：

```bash
git clone https://github.com/zzqDeco/shadiff.git
cd shadiff
go build -o shadiff .
```

## 开发分支流转

- `main` 是稳定发布基线，也是当前默认分支。
- `dev` 是日常集成分支。
- `master` 是已弃用的历史分支，不再承接新工作。
- 日常开发使用从 `dev` 切出的短期工作分支。
- 功能、修复、文档、重构和测试类 PR 先合入 `dev`。
- 当 `dev` 稳定后，再单独发起 `dev -> main` 的 PR。

GitHub Actions 会在 `main` 和 `dev` 的 push / pull request 上运行 `go test ./...` 和 `go build -o shadiff .`。匹配 `v*.*.*` 的发布标签会构建 Linux、macOS、Windows 的 amd64/arm64 压缩包，校验压缩包内容和版本元数据，并生成 SHA-256 校验和。

构建出 `dist/` 后可以在本地校验发布资产：

```bash
bash scripts/verify-release-assets.sh dist v0.1.1
```

Docker-backed 数据库集成测试是可选测试，不包含在默认单元测试命令中：

```bash
go test -v -tags integration ./internal/integration -count=1 -timeout=20m
```

### 官方 E2E Demo

运行可复现的 Docker Compose demo，使用真实 CLI 跑完整 `record -> replay -> diff -> report`，并覆盖 HTTP、MySQL、PostgreSQL、MongoDB 和 Redis side effects：

```bash
./examples/e2e/run.sh --assert
./examples/e2e/run.sh --assert --summary
./examples/e2e/run.sh --assert --binary /path/to/shadiff --summary-file /tmp/shadiff-e2e-summary.json
```

Demo 会把隔离产物写到 `examples/e2e/.work/<run-id>/`，包括 `diff.json`、`report.html` 和可选 summary JSON。端口、预期差异、release binary 验收方式和排障说明见 `examples/e2e/README.md`。

## 使用方法

### 环境诊断

在运行集成测试、官方 E2E demo 或 release binary 验收前，可以先运行只读诊断：

```bash
shadiff doctor
shadiff doctor --format json
shadiff doctor --strict --e2e
```

`doctor` 会检查配置有效性、数据/日志目录可见性、支持的 DB proxy 类型、Docker / Docker Compose 可用性，以及可选的官方 E2E 端口占用情况。缺少可选工具会显示为 warning；error 会让命令失败，`--strict` 会让 warning 也失败。

### 1. 录制流量

启动反向代理，采集老 API 的流量：

```bash
# 基本 HTTP 录制
shadiff record -t http://old-api:8080 -l :18080 -s "migration-v1"

# 带 MySQL 协议代理
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 -s "mysql-migration"

# 以后台守护进程运行
shadiff record -D -t http://old-api:8080 -l :18080 -s "bg-session"

# 带 MongoDB 协议代理
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mongo://:27018->:27017 -s "mongo-migration"

# 带 Redis 协议代理
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy redis://:16379->:6379 -s "redis-migration"

# 多数据库代理
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 \
  --db-proxy mongo://:27018->:27017 \
  --db-proxy redis://:16379->:6379 -s "full-migration"
```

将流量指向 `localhost:18080` 而非老 API。所有请求、响应和数据库操作都会被记录。
当录制启用 DB 代理时，Shadiff 会在每个请求 scope 关闭前先 flush DB hook 投递的副作用，尽量减少“窗口内发生、稍后送达”的副作用丢失。

#### 守护进程模式

以后台方式运行录制，通过 `stop` 和 `status` 管理：

```bash
# 启动守护进程
shadiff record -D -t http://localhost:8080 -l :18080 -s "long-run"

# 查看状态
shadiff record status
shadiff record status -s "long-run"

# 停止守护进程
shadiff record stop -s "long-run"
```

### 2. 回放流量

将录制的流量回放到新 API：

```bash
shadiff replay -s "migration-v1" -t http://new-api:9090 -c 5
shadiff replay -s "migration-v1" -t http://new-api:9090 \
  --db-proxy mysql://:13307->:3306
```

当 replay 启用 `--db-proxy` 时，DB side effect 会写入 `replay-records.jsonl`，并且回放必须保持串行（`--concurrency 1`）。
如果未传 `--db-proxy`，replay 会回退使用配置文件中的 `replay.dbProxies`。
回放在每条请求窗口收口前也会先 flush DB-hook telemetry，让语义 diff 更稳定地拿到窗口内的 SQL、Mongo 和 Redis 副作用。

### 3. 对比结果

对录制和回放的行为进行语义对比：

```bash
# 基本对比
shadiff diff -s "migration-v1"

# 使用自定义规则（忽略时间戳、UUID）
shadiff diff -s "migration-v1" -r rules.yaml --ignore-order

# 面向脚本/CI 的 JSON 输出
shadiff diff -s "migration-v1" -o json

# 将 CI JSON 写入文件
shadiff diff -s "migration-v1" -o json --output-file diff.json

# 存在未忽略差异时让 CI 失败
shadiff diff -s "migration-v1" --fail-on diff
```

`--fail-on` 支持 `none`（默认）、`diff`、`error`。使用 `diff` 可在存在任意未忽略差异时失败；使用 `error` 只在存在未忽略的 error 级差异时失败。

### 4. 生成报告

```bash
# 终端输出（默认）
shadiff report -s "migration-v1"

# HTML 报告
shadiff report -s "migration-v1" -f html -o report.html

# JSON 报告
shadiff report -s "migration-v1" -f json -o report.json
```

### 5. 管理会话

```bash
shadiff session list
shadiff session show <session-id>
shadiff session delete <session-id>
```

## 配置说明

应用配置存储于 `~/.shadiff/config.json`：

优先级：

```text
CLI flag > config.json > 内置默认值
```

如果配置文件不存在，Shadiff 会在首次运行时自动创建。也可以通过 `--config /path/to/config.json` 指定其他配置文件。

| 配置块 | 说明 |
|--------|------|
| `capture` | `listenAddr`、`maxBodySize`、`excludePaths`、`dbProxies` |
| `replay` | `concurrency`、`timeout`、`retryCount`、`delayMs`、`dbProxies` |
| `diff` | `ignoreHeaders`、`ignoreOrder`、`maxDiffs`、`rules`、`rulesFile` |
| `storage` | `dataDir`、`maxSessions` |
| `log` | `level`、`logDir` |

示例：

```json
{
  "capture": {
    "listenAddr": ":18080",
    "maxBodySize": 1048576,
    "excludePaths": ["/healthz"],
    "dbProxies": [
      {
        "type": "mysql",
        "listenAddr": ":13306",
        "targetAddr": "127.0.0.1:3306"
      }
    ]
  },
  "replay": {
    "concurrency": 5,
    "timeout": "30s",
    "retryCount": 1,
    "delayMs": 100,
    "dbProxies": [
      {
        "type": "mysql",
        "listenAddr": ":13307",
        "targetAddr": "127.0.0.1:3306"
      }
    ]
  },
  "diff": {
    "ignoreOrder": true,
    "maxDiffs": 500,
    "rulesFile": "rules.yaml"
  },
  "storage": {
    "dataDir": "D:/shadiff-data",
    "maxSessions": 100
  },
  "log": {
    "level": "info",
    "logDir": "D:/shadiff-data/logs"
  }
}
```

补充说明：

- `capture.maxBodySize` 会截断录制下来的请求/响应 body 预览，但会保留原始 `bodyLen`。
- 当请求 body 的内联预览被截断时，Shadiff 会在 session 目录下保存完整请求体，并在 replay 时自动优先使用该 artifact。
- `capture.excludePaths` 会继续代理匹配路径，但不会录制这些 HTTP 请求。
- `capture.dbProxies` 与 `--db-proxy` 使用同一格式。
- `diff.rulesFile` 支持 JSON、YAML、YML。
- `storage.maxSessions` 会在创建新会话前清理最旧的非录制中会话。

## 数据存储

所有持久化数据存储于 `~/.shadiff/` 目录：

```
~/.shadiff/
├── config.json                        # 全局配置
├── logs/                              # 日志文件
└── sessions/
    └── {session-id}/
        ├── session.json               # 会话元数据
        ├── records.jsonl              # 录制的行为记录（JSONL 流式）
        ├── replay-records.jsonl       # 回放结果
        ├── diff-results.json          # 对拍结果
        ├── artifacts/
        │   └── request-bodies/        # 用于忠实 replay 的完整请求体 artifact
        ├── pidfile                    # 守护进程 PID 文件（仅守护模式）
        └── daemon.log                 # 守护进程日志输出（仅守护模式）
```

## DB 代理格式

`--db-proxy` 格式：`<type>://<listen_addr>-><target_addr>`

支持类型：`mysql`、`postgres`、`mongo`、`redis`，可多次指定。

## Side-effect 存储说明

从 `v0.4.0` 开始，当前唯一的数据库副作用存储约定是 typed JSON payload，例如 `database.sql`、`database.mongo` 和 `database.redis`。`v0.3.x` 及更早版本录制的 session 使用旧 flat 字段，当前 diff/report 流程不做迁移，也不支持读取旧格式。

## 文档

- **开发指南**：`CLAUDE.md` — 架构说明 + 工程规约
- **路线图**：`plan/` — 开发阶段和进度
