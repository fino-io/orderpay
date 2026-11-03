# Issue Workspace

.fino/issues 是当前项目 Issue 的唯一事实来源。每个问题使用一个 `ISSUE-*.md` 文件；Issues 页面和 AI 都直接读写这些文件。

本 README 是项目级工作协议模板。创建项目后，只在需要时补充当前项目的服务、端口和依赖；不要复制其他项目的具体服务名、端口或历史 Issue。

## AI 快速接入

将本 README 和目标 ISSUE 文件一起交给 AI：先确认项目配置和依赖，启动最小运行环境，复现并记录证据，再修改代码；完成后把测试命令、结果、修改文件和验证结论写回原 Issue 文件。未验证通过前不要改为 `resolved`；暂不处理的问题使用 `backlog`。

## 快速启动

先读取 `fino.yaml`、`backend/configs/*-server/*.yaml` 和 `backend/cmd/*-server`，确认目标服务、端口及依赖。只启动目标服务和它声明的依赖；已有进程或基础设施直接复用。

| 服务 | 启动入口 | 依赖 |
| --- | --- | --- |
| `<service-name>` | `backend/cmd/<service>-server` | 按配置确认 |
| Issues | `frontend` | 无，只读写 `.fino/issues` |

每个新启动的服务使用独立终端。项目生成后再把表格替换为真实服务清单。

1. 启动目标服务声明的基础设施：

       docker compose -f deploy/compose/compose.infra.yaml --env-file deploy/compose/.env up -d --wait <required-infra>

   只填写目标服务实际需要的组件。不要默认启动 MinIO 或 `otel-collector`；只有项目配置或目标服务明确依赖时才启动。

2. 启动后端服务。服务名以 backend/cmd/*-server 为准：

       cd backend
       go run ./cmd/<service>-server

3. 需要页面调试时启动 Issues：

       cd frontend
       VITE_APP_ID=<app-id> VITE_API_BASE=http://127.0.0.1:<api-port> npm run dev -- --host 127.0.0.1 --port <frontend-port>

访问实际配置的 Issues 地址，选择项目目录后读取 `.fino/issues`。Issues 不应成为后端 Issue 复现的必需依赖。

## 服务与验证

| 服务 | HTTP | gRPC | Debug |
| --- | ---: | ---: | ---: |
| `<service-name>` | `<http-port>` | `<grpc-port>` | `<debug-port>` |
| Issues | `<frontend-port>` | — | — |

- 就绪检查：按服务配置请求 `/readyz`；不要假设所有服务使用相同端口。
- pprof：仅在服务声明 debug 端口时访问 `/debug/pprof/`。
- 日志：本地进程 stdout；Compose 服务使用 `docker compose logs -f <service>`。
- 后端验证：`cd backend && go test ./...`。
- 前端验证：`cd frontend && npm run build`。
- 接口或页面：按目标 Issue 的复现命令回归，并记录实际响应或页面结果。

## 收尾

测试和验证完成后，只停止本次由 AI 启动的业务服务或 Issues。未由 AI 启动的基础设施不要停止；AI 自己启动的基础设施也要确认没有其他任务使用后再停止。

## 状态与录入规则

| 状态 | 含义 |
| --- | --- |
| backlog | 暂不处理，保留为待排期事项 |
| reported | 已记录，尚未完成初步确认 |
| reproducing | 正在复现和收集证据 |
| fixing | 正在修复 |
| verifying | 正在验证修复 |
| resolved | 已验证通过 |

- 文件名与 frontmatter 的 id 保持一致，例如 ISSUE-014.md。
- frontmatter 固定包含：id、type、title、status、priority、labels、created_at、updated_at。
- type 使用 `task`、`bug` 或 `feature`，分别表示任务、缺陷和需求。
- priority 使用 P0、P1、P2、P3；时间使用 ISO 8601。
- 正文固定包含并按顺序保留：Summary、Reproduction、Environment、Expected、Actual、Test Command、Test Result、AI Analysis、Fix、Verification。
- 保留真实命令、日志、修改文件和验证结论，不要只写结论。

## Issue 示例：ISSUE-EXAMPLE.md

下面示例只用于说明格式，不要把它作为实际 Issue 文件提交：

    ---
    id: ISSUE-EXAMPLE
    type: bug
    title: 列表接口分页参数未生效
    status: reported
    priority: P2
    labels:
      - api
      - pagination
    created_at: 2026-01-01T09:00:00Z
    updated_at: 2026-01-01T09:00:00Z
    ---

    # 列表接口分页参数未生效

    ## Summary
    列表接口忽略 page_size，始终返回默认数量的数据。

    ## Reproduction
    1. 启动目标服务及其依赖。
    2. 请求列表接口并传入 page_size=2。
    3. 记录响应数量和服务日志。

    ## Environment
    - Service: <service-name>
    - Endpoint: <method> <url>
    - Date: 2026-01-01

    ## Expected
    返回不超过 2 条数据，并提供下一页游标（如有）。

    ## Actual
    返回默认数量的数据，page_size 未生效。

    ## Test Command
    curl -fsS '<url>?page_size=2'

    ## Test Result
    实际返回 <count> 条数据；日志显示请求已收到 page_size=2。

    ## AI Analysis
    <定位到参数解析、应用层或仓储层的原因>

    ## Fix
    <记录修改文件和修复方式>

    ## Verification
    <记录回归命令、结果以及是否可以标记 resolved>
