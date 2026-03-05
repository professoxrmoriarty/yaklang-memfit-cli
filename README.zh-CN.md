# yaklang-memfit-cli

[English](./README.md) | [简体中文](./README.zh-CN.md)

`yaklang-memfit-cli` 是一个面向 Yaklang 的终端 AI Agent 客户端。
它通过连接 `yaklang ai-http-gateway`，在命令行中提供基于 TUI 的交互式工作流，支持创建会话、实时流式查看输出，以及在会话中持续发送后续输入。

## 项目能力

- 连接 Yaklang AI 网关（默认：`http://127.0.0.1:8089/agent`）
- 创建并管理 Agent 运行会话（run/session）
- 基于 SSE 实时接收运行事件
- 在运行中持续推送用户输入/事件
- 支持运行时更新 AI 配置（provider/model/review policy/focus mode）
- 支持在 TUI 内取消当前运行

## 环境要求

- Go `1.24.1+`
- 可用的 Yaklang `ai-http-gateway` 服务
- 可访问的 API 前缀（默认 `/agent`）

## 快速开始

```bash
# 1) 进入项目目录
cd /Users/z3/Code/yaklang-aitui

# 2) 直接运行
go run ./cmd
```

默认连接参数：

- Host: `127.0.0.1`
- Port: `8089`
- Prefix: `/agent`

最终基础地址：

```text
http://127.0.0.1:8089/agent
```

## CLI 参数

```bash
go run ./cmd \
  -host 127.0.0.1 \
  -port 8089 \
  -prefix /agent \
  -token <JWT_TOKEN>
```

- `-host`：网关主机地址
- `-port`：网关端口
- `-prefix`：API 路由前缀
- `-token`：可选 JWT Bearer Token

## 构建可执行文件

```bash
go build -o yaklang-memfit-cli ./cmd
./yaklang-memfit-cli -host 127.0.0.1 -port 8089 -prefix /agent
```

## 内置斜杠命令

- `/help` 显示命令帮助
- `/setting` 查看当前 AI 设置
- `/provider` 查看/选择 provider
- `/model` 查看/选择 model
- `/focus_mode` 查看/选择 focus mode
- `/set <key> <value>` 更新设置项
- `/review_policy <auto|ai|manual|ai-auto>` 更新审查策略
- `/new` 开始新会话
- `/cancel` 取消当前运行中的会话
- `/clear` 清屏
- `/exit` 或 `/quit` 退出程序

## 高层工作流

1. 通过 `POST /session` 创建会话
2. 通过 `GET /run/{run_id}/events` 建立 SSE 事件流
3. 通过 `POST /run/{run_id}` 触发运行
4. 通过 `POST /run/{run_id}/events/push` 推送后续输入
5. 如需中断，通过 `POST /run/{run_id}/cancel` 取消运行

## 项目结构

- `cmd/main.go`：CLI 入口与参数处理
- `client.go`：网关 HTTP/SSE 客户端
- `tui.go`：Bubble Tea TUI 实现
- `models.go`：请求/响应模型定义
- `event_content.go`：事件内容解析辅助函数

## 说明

- 本仓库聚焦终端优先（terminal-first）的 AI Agent 体验。
- 交互方式以“流式输出 + 迭代输入”为核心。
- 如果网关开启鉴权，请通过 `-token` 传入令牌。
