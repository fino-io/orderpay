# Orderpay Package

## 运行默认服务

需要 Go 1.25.13 或更新版本，以及 PATH 中的 `protoc-gen-go`、`protoc-gen-go-grpc`。在项目根目录执行：

```bash
fino doctor --targets=api,service
fino build --targets=api,service
cd backend
cp .env.example .env
go run ./cmd/orderpay-server
```

默认 `fino.yaml` 不启用数据库，示例资源保存在内存中，重启服务后消失。HTTP、gRPC 和调试端口见 `backend/.env`；用 Ctrl-C 停止服务。再次启动无需重新复制 `.env`，已有配置和分页密钥应保留。

## 调用接口

HTTP 路径、方法和请求字段由 `proto/` 中的 RPC、HTTP 绑定与消息定义决定。生成后查看 [API 文档](docs/) 和 [OpenAPI](openapi/)，按对应接口的定义发送请求；HTTP 地址与端口以 `backend/.env` 为准。

README 在创建项目时生成，后续由项目维护，`fino build` 不会覆盖。若在 README 或其他文件中添加调用示例，需随契约变更同步检查。

## 修改与生成

先改 `proto/` 契约，再从项目根目录重新运行 `fino build --targets=api,service`，更新 API 文档、OpenAPI 和生成代码。需要 Go 客户端时，在 targets 中加入 `client`。

`fino create` 生成由项目维护的 [ER 配置](.fino/er.yaml)，已有文件保留，四项默认值为 `fields: keys`、`direction: TB`、`exclude_fields: []`、`relations: {}`。`fino build` 只读取配置并更新 [模型 ER 图](docs/er.md)，不补建或回写配置文件；缺失项使用默认值。

需要 Web 前端时将 `frontend` 加入 `--targets`。目标和配置以本项目的 `fino.yaml` 为准。

Go 依赖版本由各模块的 `go.mod`/`go.sum` 管理。默认不在 `package.yaml` 中固定 `core/go`、`finokit` 和 `gokit` 的版本；首次生成时由 `go mod tidy` 根据代码导入解析依赖，后续生成沿用已有版本。升级时在使用这些依赖的模块中执行 `go get <module>@latest` 和 `go mod tidy`，再测试和编译。需要显式约束版本时，可在 `package.dependencies` 中声明 `module@version`；生成器会先应用这些声明，再执行 `go mod tidy`，并保留已有 replacement。

## Nacos 配置中心

生成的服务默认包含 `backend/infra/nacos/`，Server 和 sidecar 统一调用 `nacos.LoadServiceConfig`，不需要在 `fino.yaml` 中选择配置中心。通过 `backend/.env` 或进程环境提供连接参数：

```dotenv
NACOS_SERVER_ADDR=<HTTP API 主机:端口>
NACOS_NAMESPACE_ID=
NACOS_GROUP=DEFAULT_GROUP
NACOS_DATA_ID=
NACOS_USERNAME=
NACOS_PASSWORD=
```

配置了 `NACOS_SERVER_ADDR` 时优先读取 Nacos；连接、鉴权、读取、解析或初始化失败时，记录告警并读取本地 `configs/<service>/`。地址留空时直接使用本地配置，不尝试连接。如果远端和本地都不可用，启动失败并返回两侧错误。没有额外的启用开关。

API 地址为 `host:port`，不带协议、路径，不能填写控制台地址；SDK gRPC 端口按 API 端口加 1000 计算，部署映射需满足此关系。public 命名空间 ID 留空，其余填写实际 ID。默认 Data ID 是 `<service>.yaml`，Group 为 `DEFAULT_GROUP`；多个服务共享环境文件时保持 `NACOS_DATA_ID` 为空。

将对应服务目录中的 YAML 顶层配置合并后发布到 Nacos。保留 `${SERVER_*}`、`${PAGINATION_ENCODED_KEY}` 等占位符，由部署环境注入值；不上传本地 `.env` 或凭据。远端成功时使用完整远端配置，不叠加本地文件；本地文件仅用于兜底。启动时拉取一次、校验 YAML 并关闭 SDK，以禁用 watcher 的内存快照加载；发布后重启生效。禁用 SDK 自身的磁盘快照，避免它掩盖本地 configs 兜底策略。

已有项目的 `bootstrap.go` 和 `.env.example` 会保留，升级时将启动加载改为 `nacos.LoadServiceConfig("<service>")` 并补齐环境变量；不要覆盖已有业务装配。自有 Worker 或其他入口可调用同一函数。SDK 首次依赖固定到已验证的 Fino module 路径版本，后续生成保留已有要求，`package.dependencies` 中的显式版本优先；不会向 finokit 根模块增加依赖。

## 代码位置

| 位置 | 用途 |
| --- | --- |
| `proto/` | 契约源，先在这里修改 RPC 与消息 |
| `go/` | 生成的 API 与 client，不手工修改 |
| `.fino/er.yaml` | ER 展示与特殊关系映射配置，手工维护 |
| `backend/modules/<service>-server/app/` | RPC 入口、请求映射与业务响应 |
| `backend/modules/<service>-server/domain/` | 用例、领域模型和仓储接口 |
| `backend/modules/<service>-server/infra/` | 仓储与外部依赖实现 |
| `backend/cmd/<service>-server/` | 启动配置与依赖装配 |

生成的自定义 RPC 默认返回 `Unimplemented`。标准 CRUD 是骨架：发布前核对契约中的每个请求字段是否已贯穿 app、domain 和 repo，补全后验证对应请求路径。文件归属和具体约定见 `AGENTS.md` 与 `skills/fino/SKILL.md`。

## 部署

生成服务后，Docker Compose 配置见 [deploy/compose](deploy/compose/README.md)，Helm 业务与 infra 配置见 [deploy/helm](deploy/helm/README.md)。构建发布说明统一维护在 Helm README 中，环境覆盖在 `deploy/helm/environments/` 中维护。
