# Docker Compose

本目录分别提供业务 `compose.yaml` 和共享平台 `compose.infra.yaml`。先启动共享平台，再启动业务；两者独立更新和停止。

本文命令默认在项目根目录执行。Helm 部署及构建发布说明见 [Helm README](../helm/README.md)。

## 配置与启动

生成器仅提供 `.env.example`；首次使用时自行复制为 `.env` 并调整配置。后续生成不会创建或修改 `.env`。

```sh
cp deploy/compose/.env.example deploy/compose/.env
```

`.env.example` 首次生成时预填随机分页密钥，复制为 `.env` 即可使用。重建保留已有示例；已有 `.env` 应保留原 `PAGINATION_ENCODED_KEY`，避免已发出的分页 token 失效。

- 平台依赖：`deploy/compose/compose.infra.yaml`
- 业务服务：`deploy/compose/compose.yaml`
- 部署环境变量与服务端口：`deploy/compose/.env`

infra provider 来源于当前服务的 `fino.yaml`；未配置的 provider 不生成对应依赖。PostgreSQL 使用官方 `postgres:16.15-bookworm` 镜像。显式配置 SQLite 时不生成外部数据库容器。当前平台还提供 Loki、Tempo、Grafana Alloy、Prometheus 和 Grafana。Alloy 统一采集 Docker 日志并接收 OTLP；Tempo 使用本地卷存储 trace，Grafana 统一查询日志、指标和链路。服务接入约定如下：

- Docker 网络：`infra`（业务 Compose 中为 `external: true`）
- 服务地址：业务依赖地址见 `compose.yaml` 的 environment；观测服务为 `loki:3100`、`tempo:3200`、`alloy:4317/4318`、`prometheus:9090`、`grafana:3000`
- 平台配置：镜像和共享凭据位于 `.env`；修改后应以同一份配置启动平台和依赖它的服务

平台端口仅绑定到 `127.0.0.1`，可安全供本机工具使用：Redis、MinIO API/Console、NATS 客户端/监控、Loki、Alloy UI/OTLP、Prometheus 和 Grafana 的默认端口分别为 `6379`、`9000/9001`、`4222/8222`、`3100`、`12345/4317/4318`、`9090`、`3000`。Tempo 只在 Docker 网络内提供服务。端口可在 `.env` 中统一调整。

Grafana 默认关闭匿名访问和自助注册。启动前请在 `.env` 中为 `GRAFANA_ADMIN_PASSWORD` 设置唯一强密码。

Grafana 通过前端同源路径提供访问，默认入口为 `http://127.0.0.1:8082/monitor/`；平台预置服务指标和集中日志面板。


两份 Compose 已通过顶层 `name` 固定项目名：平台为 `infra`，业务为 `orderpay`，日常命令无需指定 `-p`。同一台机器上的共享平台由同一份 Compose 配置与凭据统一管理；其他服务只启动自己的业务 Compose，并确认平台包含所需组件。

先启动一次共享平台：

```sh
docker compose \
  -f "deploy/compose/compose.infra.yaml" \
  --env-file "deploy/compose/.env" \
  up -d --wait
```

当前业务 Compose 会在服务启动前直接使用 SQLite 本地文件，不会创建或停止共享平台：

```sh
docker compose \
  -f "deploy/compose/compose.yaml" \
  --env-file "deploy/compose/.env" \
  up -d --wait
```

## 调试

业务 Compose 已将后端 HTTP、gRPC 和 Debug 端口映射到宿主机 `127.0.0.1`，可直接使用本机工具访问，无需额外配置。端口由 `.env` 中的 `SERVER_HTTP_PORT`、`SERVER_GRPC_PORT` 和 `SERVER_DEBUG_PORT` 指定。

业务服务输出 JSON 到 stdout，带有 `fino.logs=true` 标签的容器由 Alloy 自动发现并写入 Loki；Docker 本地日志限制为单文件 10 MiB、最多 3 个文件。Grafana 中可使用预置日志面板集中查询，命令行仍可使用 `docker compose logs -f <service>`。

Compose 内应用通过 `alloy:4318` 发送 OTLP 日志和 trace，宿主机进程使用 `OTEL_ENDPOINT=127.0.0.1:4318`。服务 tracing 默认关闭；启用后 Alloy 将 trace 批量转发给 Tempo，Tempo 使用本地持久卷存储，并按 `${TEMPO_RETENTION}` 清理。Prometheus 同时采集 Tempo 与 Alloy 指标。生产环境应按流量下调各服务 `tracing.sampleRatio`。

Loki 使用单体 TSDB 和本地持久卷，日志保留 30 天。该配置面向单机、小规模部署；需要跨主机高可用时应改用对象存储和集群部署。

### 本地前端调试

如果后端服务已在宿主机启动，可运行 Vite 开发服务器（首次执行使用 `npm ci`）：

```sh
cd frontend && \
VITE_APP_ID=1 \
VITE_API_BASE=http://127.0.0.1:31101 npm run dev -- --host 127.0.0.1 --port 32180
```

访问 `http://127.0.0.1:32180/`。`VITE_API_BASE` 用于将浏览器请求指向宿主机后端；Compose 默认使用同源 API。

## 停止

停止当前业务服务：

```sh
docker compose \
  -f "deploy/compose/compose.yaml" \
  --env-file "deploy/compose/.env" \
  down
```

平台由所有微服务共享；确认没有服务正在使用后再停止。追加 `-v` 会清理共享持久化卷：

```sh
docker compose \
  -f "deploy/compose/compose.infra.yaml" \
  --env-file "deploy/compose/.env" \
  down
```

## 离线包

生成离线包前，先完成 [构建与发布](../helm/README.md#构建与发布)。下列操作均从项目根目录执行。

```sh
PACKAGE_NAME="package-$(date +%Y%m%d-%H%M%S)"
PACKAGE_DIR="${PWD}/.fino/release/${PACKAGE_NAME}"

mkdir -p "${PACKAGE_DIR}/deploy" "${PACKAGE_DIR}/backend" "${PACKAGE_DIR}/images"

for compose in compose.infra.yaml compose.yaml; do
  docker compose \
    -f "deploy/compose/${compose}" \
    --env-file "deploy/compose/.env" \
    config --images
done | sort -u > "${PACKAGE_DIR}/images/images.txt"

xargs docker image save \
  -o "${PACKAGE_DIR}/images/images.tar" \
  < "${PACKAGE_DIR}/images/images.txt"

cp -R deploy/compose "${PACKAGE_DIR}/deploy/"
cp -R backend/configs "${PACKAGE_DIR}/backend/"

COPYFILE_DISABLE=1 tar -C "${PWD}/.fino/release" \
  -czf "${PACKAGE_DIR}.tgz" "${PACKAGE_NAME}"
(cd .fino/release && shasum -a 256 "${PACKAGE_NAME}.tgz" > "${PACKAGE_NAME}.tgz.sha256")
```

目标机导入并启动：

```sh
shasum -a 256 -c package-*.tgz.sha256
tar -xzf package-*.tgz
docker image load -i package-*/images/images.tar

cd package-*/
docker compose -f "deploy/compose/compose.infra.yaml" --env-file "deploy/compose/.env" up -d --wait
docker compose -f "deploy/compose/compose.yaml" --env-file "deploy/compose/.env" up -d --wait
```

## 约束

- 后端 HTTP、gRPC 和 Debug 端口仅绑定宿主机 `127.0.0.1`；生成前端时，`frontend` 提供对外入口。
- 平台与业务分别启动，不能将两个 Compose 文件作为同一个 `-f` 合并运行，因为 `infra` 网络在平台侧创建、业务侧以外部网络接入。
- Compose 从 `bootstrap/caddy` 只读挂载 Caddy 配置，提供静态文件、SPA fallback、压缩和 API 反向代理。
- 部署模板资产由 `fino build` 刷新；环境 `.env`、外部 values 和后端业务配置保留。`.goreleaser.yaml` 是项目发布配置，可按需要扩展。
