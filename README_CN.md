# Shadiff - 影子流量语义对拍工具

[English](README.md)

## 项目简介

Shadiff 是一个影子流量语义对拍工具，用于跨框架/跨语言的 API 迁移验证。通过**录制-回放-对拍**三段式流程：以反向代理方式透明采集老 API 的完整行为（输入、输出、数据库副作用），然后将相同的输入回放到新 API，语义级比较两边的行为差异。

## 核心特性

- **HTTP 反向代理录制** — 通过 `httputil.ReverseProxy` 透明采集流量，记录完整的请求/响应对及时间信息
- **数据库协议代理** — TCP 级别黑盒采集，支持 MySQL（COM_QUERY）、PostgreSQL（Simple/Extended Query）和 MongoDB（OP_MSG Wire Protocol）
- **并发回放引擎** — 基于 Worker Pool 的回放，支持可配置的并发数，请求变换（host/header 替换）
- **语义级 JSON 对比** — 递归结构化比较，支持路径追踪（如 `body.data.items[0].name`）
- **可配置规则系统** — 支持忽略时间戳、UUID、数值容差、数组顺序等，通过 YAML 规则配置
- **多格式报告** — 终端彩色输出、JSON、HTML 报告，附带汇总统计
- **会话管理** — 完整的会话生命周期管理，JSONL 流式存储

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.24 | 主语言 |
| Cobra | v1.9 | CLI 框架 |
| slog | 标准库 | 结构化日志 + 日志轮转 |
| JSONL | - | 流式记录存储 |

## 项目结构

```
shadiff/
├── main.go                            # CLI 入口
├── go.mod                             # Go 1.24 模块
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
│   └── version.go                     # shadiff version
├── internal/
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
│   │       ├── mysql.go               # MySQL 协议代理（COM_QUERY 解析）
│   │       ├── postgres.go            # PostgreSQL 协议代理（Simple/Extended Query）
│   │       └── mongo.go               # MongoDB 协议代理（OP_MSG Wire Protocol）
│   ├── storage/                       # 存储层
│   │   ├── store.go                   # SessionStore/RecordStore/DiffStore 接口
│   │   └── filestore.go              # 文件系统实现（JSONL）
│   ├── replay/                        # 回放引擎
│   │   ├── engine.go                  # 回放编排器
│   │   ├── worker.go                  # 并发 Worker Pool
│   │   └── transform.go              # 请求变换（host/header 替换）
│   ├── diff/                          # 语义对拍引擎
│   │   ├── engine.go                  # 对拍编排器，按序号配对记录
│   │   ├── json.go                    # JSON 结构化递归 diff
│   │   ├── db.go                      # SQL 数据库对比（MySQL/PostgreSQL）
│   │   ├── mongo.go                   # MongoDB 操作对比
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

- **Go** >= 1.24

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

## 使用方法

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

# 多数据库代理
shadiff record -t http://old-api:8080 -l :18080 \
  --db-proxy mysql://:13306->:3306 \
  --db-proxy mongo://:27018->:27017 -s "full-migration"
```

将流量指向 `localhost:18080` 而非老 API。所有请求、响应和数据库操作都会被记录。

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
```

### 3. 对比结果

对录制和回放的行为进行语义对比：

```bash
# 基本对比
shadiff diff -s "migration-v1"

# 使用自定义规则（忽略时间戳、UUID）
shadiff diff -s "migration-v1" -r rules.yaml --ignore-order
```

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

| 配置块 | 说明 |
|--------|------|
| `capture` | 代理设置（监听地址、超时） |
| `replay` | 回放设置（并发数、延迟、超时） |
| `diff` | 对比设置（默认规则、忽略模式） |
| `storage` | 存储设置（数据目录） |
| `log` | 日志设置（级别、目录、轮转） |

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
        ├── pidfile                    # 守护进程 PID 文件（仅守护模式）
        └── daemon.log                 # 守护进程日志输出（仅守护模式）
```

## DB 代理格式

`--db-proxy` 格式：`<type>://<listen_addr>-><target_addr>`

支持类型：`mysql`、`postgres`、`mongo`，可多次指定。

## 文档

- **开发指南**：`CLAUDE.md` — 架构说明 + 工程规约
- **路线图**：`plan/` — 开发阶段和进度
