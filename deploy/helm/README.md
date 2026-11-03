# Helm 部署

命令默认在项目根目录执行。app 由 Helm 部署，infra 由 Argo CD 部署；本地 Helm 只负责将 infra Chart 渲染为 Argo CD Application。

| 文档 / 目录 | 内容 |
| --- | --- |
| [app/README.md](app/README.md) | 应用配置、依赖初始化、Helm 部署、验收与维护 |
| [infra/README.md](infra/README.md) | 共享基础设施的 Argo CD 部署、验收与维护 |

首次部署顺序：准备集群与 Argo CD → 按 infra 手册部署共享服务 → 构建并推送镜像 → 按 app 手册部署应用。停止时先停 app，再停 infra。Compose 操作见 [Compose 文档](../compose/README.md)。

所有服务共用 `app/` Chart，每个服务使用独立 release 和一份后端运行配置。

## 服务访问

应用 Service 保持 ClusterIP。临时验收使用 port-forward；长期访问使用域名、HTTPS 和共享 Gateway，应用通过 HTTPRoute 挂载。已有 Ingress 可继续使用，同一 release 的 Ingress 与 Gateway 模式二选一。

共享入口和证书由集群维护者管理，app 只引用入口并管理自己的路由。K3s 自带 Traefik 时通过 HelmChartConfig 扩展，不重复安装控制器；其余 infra 应用仍由 Argo CD 管理。配置与验收分别见 [app 访问](app/README.md#访问与验收)和 [infra 共享入口](infra/README.md#共享入口)。

## 构建与发布

使用项目根目录的 `.goreleaser.yaml`。需要 GoReleaser 2.15+、Go、Node.js/npm 和带 buildx 的 Docker；先确认 `builds`、`dockers_v2.images`、`platforms` 及前端构建 hook 与项目一致。默认平台为 `linux/amd64`，其他架构需同时调整二进制与镜像配置；SQLite 等 CGO 服务还需匹配运行镜像的 Linux musl 编译环境。

发布前检查 `.goreleaser.yaml` 的服务名、构建入口与镜像仓库是否对应实际服务。脚手架默认包含前端构建；需要前端时先生成 `frontend` 目标，纯后端项目先移除前端 hook 和 `dockers_v2` 项，再执行下方命令。包含前端时，Node.js 版本以生成的 `frontend/package.json` 为准，首次使用或 lock 文件变化时先安装依赖（若 hook 已执行 `npm ci` 则无需另行安装）：

```sh
npm ci --prefix frontend
```

snapshot 本地加载需要可访问的 Docker daemon；用 `docker info`、`docker buildx inspect` 检查 daemon 和目标平台。HTTP 仓库需分别配置推送端 Docker Engine、实际推送使用的 BuildKit builder，以及每个拉取镜像的集群节点。Docker 和节点 containerd 的仓库设置分别管理，参见 [Docker 配置](https://docs.docker.com/reference/cli/dockerd/#insecure-registries)、[BuildKit 配置](https://docs.docker.com/build/buildkit/toml-configuration/)与 [K3s 配置](https://docs.k3s.io/installation/private-registry)。

### 本地构建并推送

snapshot 允许未提交修改，构建前后端并将镜像加载到本机 Docker：

```sh
mkdir -p .fino/cache/go-build
goreleaser check --config .goreleaser.yaml
GOCACHE="$PWD/.fino/cache/go-build" goreleaser release --config .goreleaser.yaml --snapshot --clean
```

默认 `dockers_v2` 配置在 snapshot 下将 `latest` 追加平台后缀，得到 `latest-amd64`。先将 `.goreleaser.yaml` 的镜像仓库改为实际地址；下面的 `IMAGE_PREFIX` 应与该配置一致，修改变量本身不会改变构建目标。构建后手动推送：

```sh
export IMAGE_PREFIX=registry.example.com/project
docker push "$IMAGE_PREFIX/orderpay-server:latest-amd64"
docker push "$IMAGE_PREFIX/orderpay-frontend:latest-amd64"
```

只有实际生成并配置了前端时才执行前端镜像的构建、推送命令。不要给上述 snapshot 命令添加 `--skip=publish`，它会跳过 `dockers_v2` 镜像构建。

将对应 `app/services/<service>-server.yaml` 的镜像 repository、tag 与本次产物对齐，再按 [app 部署手册](app/README.md)部署。复用标签时需设置相应镜像的 `pullPolicy: Always` 并重启 Deployment；需可靠回滚时使用独立版本 tag。

### 正式发布

要求干净工作区、当前提交已有合法版本 Git tag，并已登录镜像仓库。按项目 `.goreleaser.yaml` 推送版本镜像：

```sh
GOCACHE="$PWD/.fino/cache/go-build" goreleaser release --config .goreleaser.yaml --clean
```

默认同时发布版本标签和 `latest-amd64`。产物位于 `.fino/release/`，`--clean` 会清理上次产物；[离线包制作](../compose/README.md#离线包)见 Compose 文档。

## 配置与维护

- `backend/configs/<service>-server/*.yaml`：应用运行配置，供本地、Compose 与 Helm 共用。
- `app/values.yaml`：共享部署默认值。
- `app/services/<service>-server.yaml`：服务镜像、地址、资源和入口；每个服务独立 release。
- `infra/values.yaml`：整个项目共用的基础设施。

应用初始化、发布、离线预览、多服务更新和旧项目迁移统一见 [app 手册](app/README.md#日常发布)；共享基础设施见 [infra 手册](infra/README.md)。

后端 configs、共享/服务 values、仓库注册清单和 README 首次生成后由项目维护；Chart 模板和配置打包脚本随 Fino 更新。密码存集群 Secret，临时文件、缓存和发布产物不纳入 Git。
