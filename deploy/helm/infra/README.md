# infra 部署（Argo CD）

共享服务由 Argo CD 管理；Helm 仅渲染 Application，不通过 `helm install` 安装 infra。命令默认在项目根目录、kubectl 已连接目标集群的终端执行。直接修改本目录的 [values.yaml](values.yaml)，集群连接和验收记录补充在本文中。

[部署总览与镜像构建](../README.md) · [app 部署](../app/README.md)

## 部署与更新

集群需已安装 Argo CD 和 `local-path` 存储类；kubectl 与 API server 应相差不超过一个 minor 版本。本文命令使用默认的 `argocd`、`infra`、`observability` namespace 和 `infra-` Application 前缀；修改 values 后同步替换命令中的名称、服务地址和仓库清单的 namespace。默认提供 Loki、Tempo、Alloy、Prometheus、Grafana；其他依赖自行准备。

首次同步前检查节点 Ready、Argo CD Pod 就绪，以及存储类和 Application CRD：

```sh
kubectl version
kubectl get nodes
kubectl get storageclass local-path
kubectl -n argocd get pods
kubectl get crd applications.argoproj.io
kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
kubectl -n observability get pvc
kubectl -n observability get secret grafana-admin
```

已有 Secret 时保留原值；只有返回 `NotFound` 且没有旧数据需要恢复时才创建。已有 PVC 但 Secret 丢失时先恢复原凭据；连接或权限错误不能按不存在处理。各服务使用独立密码：

| namespace | Secret | 字段 |
| --- | --- | --- |
| `observability` | `grafana-admin` | `admin-user`、`admin-password` |

需要创建时执行对应命令：

```sh
kubectl -n observability create secret generic grafana-admin --from-literal=admin-user=admin --from-literal=admin-password="$(openssl rand -hex 24)"
```

检查 `values.yaml` 中的组件、namespace、存储、资源和入口配置后，渲染并应用：

```sh
helm lint deploy/helm/infra
helm template infra deploy/helm/infra > /tmp/infra-applications.yaml
kubectl apply -f /tmp/infra-applications.yaml
kubectl -n argocd get applications
```

默认关闭自动同步、自动 prune 和级联删除；应用上面的清单仅创建或更新 Application。先在 Argo CD 查看 Diff，再逐个手动 Sync，不勾选 Prune。顺序为 Loki、Tempo、Prometheus、Alloy、Grafana，每个达到 `Synced / Healthy` 后再同步下一个。

也可通过 kubectl 同步。`APP` 使用 `kubectl get applications` 中的实际名称；默认前缀为 `infra-`，每次只操作一个：

```sh
APP=infra-loki
kubectl -n argocd patch application "$APP" --type=merge -p '{"operation":{"sync":{"prune":false}}}'
```

查询状态，结合 `STARTED` 确认本次操作 `Succeeded` 且为 `Synced / Healthy`：

```sh
kubectl -n argocd get application "$APP" -o custom-columns=NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status,PHASE:.status.operationState.phase,STARTED:.status.operationState.startedAt
```

失败或长时间未就绪时查看原因，处理后再继续：

```sh
kubectl -n argocd get application "$APP" -o jsonpath='{.status.conditions}{"\n"}{.status.operationState.message}{"\n"}'
```

更新本地 `values.yaml` 后，重新 lint、渲染、应用，再同步受影响的 Application；本地文件不会被 Argo CD 自动读取。

## 停止与恢复

按组件操作，namespace 与 instance 见下表。先记录副本数，确认 StatefulSet 的 `persistentVolumeClaimRetentionPolicy.whenScaled` 为 `Retain` 或未设置：

```sh
export INFRA_NAMESPACE=observability
export COMPONENT=grafana
kubectl -n "$INFRA_NAMESPACE" get deploy,sts \
  -l "app.kubernetes.io/instance=$COMPONENT" -o yaml
```

暂停期间不要 Sync；若启用了自动同步与 self-heal，先关闭。

```sh
kubectl -n "$INFRA_NAMESPACE" scale deployment,statefulset \
  -l "app.kubernetes.io/instance=$COMPONENT" --replicas=0
```

恢复原副本数（默认 1）：

```sh
kubectl -n "$INFRA_NAMESPACE" scale deployment,statefulset \
  -l "app.kubernetes.io/instance=$COMPONENT" --replicas=1
kubectl -n "$INFRA_NAMESPACE" rollout status deployment,statefulset \
  -l "app.kubernetes.io/instance=$COMPONENT" --timeout=5m
```

全部停止时先 Alloy、Grafana，再其他组件；恢复按部署顺序。检查实际副本与 Pod 状态，等待退出或就绪。不要删除 namespace 或 PVC；删除 Application、设置 `enabled: false` 或执行 `helm uninstall infra` 均不能代替暂停工作负载。

## 重启与排查

仅重启副本大于 0 的组件，沿用上面的 `INFRA_NAMESPACE`、`COMPONENT`：

```sh
kubectl -n "$INFRA_NAMESPACE" rollout restart deployment,statefulset -l "app.kubernetes.io/instance=$COMPONENT"
kubectl -n "$INFRA_NAMESPACE" rollout status deployment,statefulset -l "app.kubernetes.io/instance=$COMPONENT" --timeout=5m
kubectl -n "$INFRA_NAMESPACE" logs -l "app.kubernetes.io/instance=$COMPONENT" --all-containers=true --tail=100
kubectl -n "$INFRA_NAMESPACE" get events --sort-by=.lastTimestamp
```

## 卸载与重装（保留数据）

先停止依赖业务并备份数据，保留 namespace、外部 Secret 与 PVC。确认 StatefulSet 的 `whenDeleted`、`whenScaled` 均为 `Retain` 或未设置；若 Application 直接管理 PVC，先为这些 PVC 添加 `argocd.argoproj.io/sync-options: Delete=false` 注解，防止级联删除。

按组件卸载，替换为实际 Application 名；默认 PostgreSQL（如有）、Loki、Tempo、Prometheus 在 Alloy、Grafana 之后卸载：

```sh
export APPLICATION=infra-grafana
kubectl -n argocd patch application "$APPLICATION" --type=merge \
  -p '{"metadata":{"finalizers":["resources-finalizer.argocd.argoproj.io"]}}'
kubectl -n argocd delete application "$APPLICATION" --wait=true --timeout=5m
```

这是级联删除：由 Argo CD 清理工作负载、Service、配置及 RBAC；单独删除没有 finalizer 的 Application 只会移除管理关系。详见 [Argo CD 删除说明](https://argo-cd.readthedocs.io/en/stable/user-guide/app_deletion/)。

卸载后确认相关工作负载已消失、PVC 仍为 Bound、外部 Secret 仍存在。重装时重新执行[部署与更新](#部署与更新)，按原顺序手动 Sync；保持 chart 版本、releaseName、namespace、存储配置和凭据不变，以复用原 PVC。不要重新生成已有密码。

## 共享入口

共享入口只准备一次。优先复用集群现有控制器、GatewayClass 和 Gateway；若由其他团队管理，只记录其名称、namespace、listener、TLS 和允许挂载的应用 namespace。app 卸载时保留共享入口和证书。

### K3s：复用现有 Traefik

[k3s-traefik.yaml](k3s-traefik.yaml) 是可选的集群引导配置，不由本 Chart 渲染或自动应用。它通过 K3s 的 HelmChartConfig 启用 Gateway provider，由原 Traefik release 创建共享 `apps` Gateway；不额外创建 Argo CD Application 或重复安装 Traefik。已有 HelmChartConfig 时先合并，保留原设置。

先核对 Traefik 版本、Gateway API CRD 和 TLS Secret：

```sh
kubectl -n kube-system get deploy traefik
kubectl -n kube-system get helmchartconfig traefik -o yaml
kubectl get crd gateways.gateway.networking.k8s.io httproutes.gateway.networking.k8s.io
kubectl -n kube-system get secret apps-tls
```

`NotFound` 表示对应资源尚未准备；其他错误先处理。CRD 与 controller 版本需兼容，安装由集群维护者负责，不由 app 自动安装。K3s 提供的 CRD 继续由 K3s 管理，不覆盖安装其他版本。

准备包含所用域名 SAN 的证书和私钥，将其存入 Gateway 所在 namespace。已有 Secret 保留，续期时更新原 Secret。以下路径替换为实际文件：

```sh
kubectl -n kube-system create secret tls apps-tls --cert=/path/to/fullchain.pem --key=/path/to/key.pem --dry-run=client -o yaml | kubectl apply -f -
```

为应用 namespace 授权，并应用已检查的 K3s 配置：

```sh
kubectl label namespace "$NAMESPACE" fino.io/gateway=apps
kubectl apply -f deploy/helm/infra/k3s-traefik.yaml
kubectl -n kube-system rollout status deployment/traefik --timeout=2m
kubectl wait --for=condition=Accepted gatewayclass/traefik --timeout=1m
kubectl -n kube-system wait --for=condition=Programmed gateway/apps --timeout=1m
```

`NAMESPACE` 使用已创建的应用 namespace；仅对需要挂载该 Gateway 的 namespace 设置标签。示例 listener `websecure` 使用容器端口 8443，由 Traefik Service 的 443 对外提供 HTTPS；共享 Gateway 没有 HTTP listener，不自动提供 HTTP 到 HTTPS 重定向。已有其他 Ingress 路由仍由原 provider 管理。

Gateway 应为 `Accepted=True`、`Programmed=True`，listener 的 `ResolvedRefs=True`。然后按 [app 手册](../app/README.md#访问与验收)发布 HTTPRoute，检查路由状态和实际域名访问。K3s 配置和证书由集群维护者维护，infra values、路由 values 中不保存私钥。非 K3s 集群由现有 Gateway controller 提供等效入口，无需应用该 K3s 文件。

## 访问与检查

| instance | namespace | 集群内地址 |
| --- | --- | --- |
| `alloy` | `observability` | `alloy.observability.svc.cluster.local:4317/4318`（OTLP） |
| `prometheus` | `observability` | `http://prometheus.observability.svc.cluster.local` |
| `loki` | `observability` | `http://loki.observability.svc.cluster.local:3100` |
| `tempo` | `observability` | `http://tempo.observability.svc.cluster.local:3200` |
| `grafana` | `observability` | `http://grafana.observability.svc.cluster.local` |

```sh
kubectl -n observability get deploy,sts,pods,svc,pvc
kubectl -n observability port-forward svc/grafana 3000:80
```

打开 `http://127.0.0.1:3000`，凭据来自 `grafana-admin`。确认 Pod 就绪、PVC 为 Bound、Grafana 数据源健康，日志、trace 和指标可查询。业务面板在业务未部署时可为空。

`local-path` 数据依赖节点磁盘，备份另行安排。同一日志选择 stdout 或 OTLP 一条采集路径。

`values.yaml`、`argocd-repository.yaml`（如有）和本文首次生成后保留项目修改；重新生成会刷新 Chart 模板。
