# app 部署（Helm）

命令默认在项目根目录、kubectl 已连接目标集群的同一终端执行，需要 Bash、Helm、kubectl 和 OpenSSL，凭据输入命令在 Bash 或 zsh 中执行。kubectl 与 API server 应相差不超过一个 minor 版本，先用 `kubectl version` 核对。先按 [infra 手册](../infra/README.md)准备共享服务，按[镜像构建](../README.md#构建与发布)构建并推送镜像。

## 部署参数

| 服务 | 部署 values | 唯一运行配置 |
| --- | --- | --- |
| `orderpay-server` | [values](services/orderpay-server.yaml) | [configs](../../../backend/configs/orderpay-server/) |

以下命令以一个服务为例，替换 `SERVICE` 即可更新其他服务。默认 release 是服务名去掉 `-server`，Secret 由服务 values 的 `existingSecret` 指定，初始为 `<service>-runtime`。首次初始化时的 `SECRET_NAME` 须与该服务 `existingSecret` 一致。`NAMESPACE` 控制本次部署的目标 namespace：

```sh
export SERVICE=orderpay-server
export RELEASE="${SERVICE%-server}"
export NAMESPACE=default
kubectl version
kubectl get nodes
```

直接修改 `services/$SERVICE.yaml` 的部署参数和 `backend/configs/$SERVICE/*.yaml` 的运行配置；公共默认值在 [values.yaml](values.yaml)，包括共用的 `env.OTEL_ENDPOINT`；服务 `env` 可覆盖公共值。无需复制或同步运行配置：

| 配置 | 要求 |
| --- | --- |
| 镜像 | `image.repository`、`image.tag` 与已推送镜像一致；启用前端时同步设置 `frontend.image` |
| 依赖地址 | values 的 `env` 提供地址；数据库名和用户名以 `backend/configs/<service>-server/db.yaml` 为准，密码通过 Secret 注入 |
| 监听与探针 | 后端 `server.yaml` 保留本地监听；Chart 根据 `server.configKey` 定位对应配置块，用 values 的 `server.bindHost`、`server.allowRemoteDebug` 和 `ports` 生成 Kubernetes 配置 |
| 前端 / 入口 | 默认关闭，按需设置 `frontend`、`gateway` 或兼容的 `ingress`；`frontend.apiPrefix` 与实际 HTTP API 路径一致，debug 端口仅用于集群内探针和监控 |
| orderpay / SQLite | 单副本；需持久化时先创建 PVC，再设置 `sqlite.existingClaim`；未设置时使用 emptyDir，Pod 删除会丢失数据 |


首次生成时，任一服务使用 PostgreSQL 就会在共享 infra 中启用它。新增提供方后还需检查已保留的 `infra/values.yaml`；默认 infra 提供观测组件。其他依赖需要单独部署或接入已有服务。敏感配置通过 Secret 注入时，应移除 values `env` 中的同名项，避免显式 env 覆盖 Secret。

`CONFIG_PATH` 和 `SERVER_*_PORT` 由 Chart 管理。复用 `latest-amd64` 等标签时，后端和已启用前端的 `pullPolicy` 均设为 `Always`。tracing 默认关闭，需要 trace 时在对应服务的 `tracing.yaml` 启用并检查采集端地址。

## 日常发布

先按[部署参数](#部署参数)选择本次服务。**首次部署先完成[首次初始化](#首次初始化)**；之后安装与更新使用同一套命令。修改镜像 tag 或配置后，打包并检查：

```sh
CHART="$(bash deploy/helm/app/prepare.sh "$SERVICE")"
VALUES="deploy/helm/app/services/$SERVICE.yaml"
helm lint "$CHART" -f "$VALUES" --strict
```

Chart 使用 `values.schema.json` 校验镜像、端口、副本数和开关类型；错误配置会在 lint、预览或部署时直接报错。

需要离线预览时，在部署前执行：

```sh
helm template "$RELEASE" "$CHART" -n "$NAMESPACE" -f "$VALUES"
```

安装或更新，并等待就绪：

```sh
helm upgrade --install "$RELEASE" "$CHART" -n "$NAMESPACE" -f "$VALUES" --wait --timeout 5m
kubectl -n "$NAMESPACE" get deploy,pods,svc -l "app.kubernetes.io/instance=$RELEASE"
```

`prepare.sh` 只将共享 Chart 和后端 YAML 打包到 Git 忽略的 `.fino/cache/helm/$SERVICE/`，不访问集群。缓存可删除，不手动维护；每次发布重新 prepare。确认就绪副本数符合 values，Pod 正常运行后执行[访问与验收](#访问与验收)。需要调整 release、namespace、超时等参数时直接修改 Helm 命令。

ConfigMap 内容变化通过 checksum 自动触发滚动更新。外部 Secret 更新或仅覆盖同名镜像标签时，显式重启并等待就绪：

```sh
kubectl -n "$NAMESPACE" rollout restart deployment -l "app.kubernetes.io/instance=$RELEASE"
kubectl -n "$NAMESPACE" rollout status deployment -l "app.kubernetes.io/instance=$RELEASE" --timeout=5m
```

## 访问与验收

### 临时验收

默认关闭入口，不依赖 Gateway API CRD。使用同一 kubeconfig，在另一终端通过 `kubectl port-forward` 转发实际 Service；名称和端口以 `kubectl -n "$NAMESPACE" get svc` 为准。此方式只用于调试或验收，停止转发后本机地址失效。

### 长期访问

先按 [infra 共享入口](../infra/README.md#共享入口)准备 Gateway、TLS Secret、namespace 授权和客户端 DNS。然后在服务 values 启用 `gateway`，例如仅暴露后端：

```yaml
gateway:
  enabled: true
  parentRef:
    name: apps
    namespace: kube-system
    sectionName: websecure
  routes:
    - name: api
      hostnames: [api.example.com]
      backend: server
```

将示例域名改为实际域名。已启用前端时可将 backend 改为 `frontend`，其 API 代理继续使用服务 values 的 `frontend.apiPrefix`；需要独立入口时增加不同 name 的路由。可选 `pathPrefix` 默认 `/`。普通应用可以只使用一个域名，应用对外地址相关配置由项目自行维护。`ingress.enabled` 必须为 false。

按[日常发布](#日常发布)打包和安装。Helm 等待工作负载就绪后，还需检查路由：

```sh
kubectl -n "$NAMESPACE" get httproute -l "app.kubernetes.io/instance=$RELEASE" -o yaml
```

确认目标 parent 的 `status.parents[].conditions` 包含 `Accepted=True`、`ResolvedRefs=True`，且 observedGeneration 对应当前 generation。HTTPRoute 的条件不在顶层，不能使用普通的 `kubectl wait --for=condition=Accepted httproute/...` 判断。

客户端将域名解析到可达的入口地址并校验证书：内部可使用内网 DNS 或 hosts；公网访问需配置公网可达入口及对应 DNS。hosts 不会创建路由或证书。私有 CA 证书需要在客户端信任；公开可信证书按实际签发方式维护，不在 Chart 中生成密钥。

通过 HTTPS 验收页面、静态资源、API 认证和核心业务流程，并确认应用返回的外部 URL 使用实际域名。原有 Ingress 模式使用 `ingress` 配置，Gateway 关闭；新增模式不会自动迁移已有入口。项目实际域名、验收命令与结果记录在本文中。

## 首次初始化

先准备目标 namespace，再选择要部署的服务执行对应初始化；已有数据库和凭据保留原值。Secret 名以服务 values 的 `existingSecret` 为准，修改 release 名不改变 Secret 名。

```sh
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
```


### orderpay-server

```sh
export SECRET_NAME=orderpay-runtime # 与该服务 values 的 existingSecret 一致
```

分别检查应用数据库和 Secret。共享数据库服务就绪并不代表应用数据库、用户和授权已创建；已有角色也不代表数据库已存在。

SQLite 无需预建数据库角色，需要持久化时先确认配置的 PVC 已存在且可用。

检查已有运行时 Secret 的字段：

```sh
kubectl -n "$NAMESPACE" describe secret "$SECRET_NAME"
```

已有 Secret 时保留原值并补齐缺少字段；只有返回 `NotFound` 时才创建。下面按当前基础设施选择读取必要凭据；项目自有的认证字段也应在同一 Secret 中准备好：

```sh
kubectl -n "$NAMESPACE" create secret generic "$SECRET_NAME" \
  --from-literal=PAGINATION_ENCODED_KEY="$(openssl rand -base64 32)"
```

保留已有分页密钥，避免已发出的分页 token 失效；密码不写入 values 或提交到 Git。


## 维护与回滚

暂停应用及已启用前端，保留 release、配置与外部 Secret；恢复时重新执行[日常发布](#日常发布)，副本数以 values 为准：

```sh
kubectl -n "$NAMESPACE" scale deployment -l "app.kubernetes.io/instance=$RELEASE" --replicas=0
```

```sh
kubectl -n "$NAMESPACE" logs \
  -l "app.kubernetes.io/instance=$RELEASE,app.kubernetes.io/component=server" -c server --tail=100
kubectl -n "$NAMESPACE" get events --sort-by=.lastTimestamp
helm history "$RELEASE" -n "$NAMESPACE"
```

从 history 选择实际 revision 后回滚：

```sh
REVISION=1 # 改为目标版本
helm rollback "$RELEASE" "$REVISION" -n "$NAMESPACE" --wait --timeout 5m
```

回滚恢复镜像引用和 ConfigMap，不恢复外部 Secret、数据库数据或已覆盖的镜像标签。需要可靠回滚时使用独立版本 tag。

不再需要应用时卸载，保留外部依赖、Secret 和已有 PVC：

```sh
helm uninstall "$RELEASE" -n "$NAMESPACE"
```

## 配置与多服务

运行配置只维护 `backend/configs/<service>-server/*.yaml`。Chart 的 `.Files` 无法读取 Chart 外的配置，因此需要 prepare，不直接对源码 `app/` 执行部署。Kubernetes 监听参数由 values 覆盖，本地文件不变。

新增服务时重新生成以补齐服务 values，并在 `.goreleaser.yaml` 中准备构建入口和镜像；服务名转换为 kebab-case 后需唯一，新增基础设施提供方时检查 `infra/values.yaml`。

多服务逐一设置 `SERVICE` 和 `RELEASE`，重复[日常发布](#日常发布)的命令。所有服务的数据库和 Secret 就绪后，也可批量更新；按文件名顺序执行，存在依赖时调整顺序。任一失败即停止，之前成功的 release 保留：

```sh
(
  set -e
  for values in deploy/helm/app/services/*-server.yaml; do
    service="$(basename "$values" .yaml)"
    chart="$(bash deploy/helm/app/prepare.sh "$service")"
    helm lint "$chart" -f "$values" --strict
    helm upgrade --install "${service%-server}" "$chart" -n "$NAMESPACE" -f "$values" --wait --timeout 5m
  done
)
```

此例使用默认 release 命名；已有自定义 release 名时按各自名称单独更新。

`services/` 中的 values 文件就是批量部署清单；移除服务时需清理对应文件，并按需手动卸载原 release，不会自动删除集群资源。

共享 values、服务 values、后端 configs 和本文保留项目修改。新增服务时 Fino 生成缺少的 `services/<service>-server.yaml`，不覆盖已有配置；重新生成刷新共用 Chart 模板和配置打包脚本。旧项目切换时，先在每个服务 values 中补上 `existingSecret`，填入原脚本使用的实际 Secret 名；自定义 release 名继续传给 Helm。确认完成迁移后移除原 `deploy.sh`，生成器不自动删除项目文件。

升级更早的项目时，先将原 `app/values.yaml` 的服务参数移入对应 `services/*.yaml`，再更新共享默认值；原 `app/configs/` 与后端配置合并后移除。检查 `nameOverride` 与原 selector 一致，生成器不自动迁移私有配置。

## 流程依据

采用 Chart + 服务 values + `helm upgrade --install`，安装、升级、预览、回滚均使用 Helm 原生命令，参考 [Helm 官方建议](https://helm.sh/docs/howto/charts_tips_and_tricks/)、[Grafana 部署手册](https://grafana.com/docs/grafana/latest/setup-grafana/installation/helm/)和 [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)。prepare 仅适配本项目的单份外部配置源，限制见 [Helm 文件访问](https://helm.sh/docs/chart_template_guide/accessing_files/)。
