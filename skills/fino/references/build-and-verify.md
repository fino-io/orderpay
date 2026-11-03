# 构建与验证

用于目标选择、生成、工具链、运行故障、发布及 skill 打包。按改动选择下面的检查，不把全部流程当作每次任务的门禁。

## 命令与目标

在项目根目录运行，或显式指定项目路径。生成项目使用仓库配置/锁定版本的 `fino`；验证 Fino 源码时使用 `go run ./cmd/fino`，记录命令与版本。

```bash
fino doctor --output=. --targets=<targets>
fino build --output=. --targets=<targets>
```

两者必须显式指定 `--targets`；一次构建合并已有且受影响的目标，不为验证新增目标。

| 目标 | 何时选择 | 输出/依赖 |
|---|---|---|
| `api` | 只需刷新 API、文档、OpenAPI 或 IR | `go/` protobuf/gRPC、`docs/`、`openapi/`、`.fino/ir/` |
| `service` | 受影响的 `backend/` | 自动准备 API/IR，再生成后端 |
| `client` | 受影响的 `go/*/client/` | 自动准备 API/IR，可独立运行，不依赖 `service` |
| `frontend` | 受影响的前端生成契约/模板 | 自动准备 API/IR，再生成前端 |
| `sidecar` | 受影响的 `backend/cmd/*-sidecar-server/` | 必须同时选择 `service` |

`service`、`client`、`frontend` 隐式生成 API/IR。例如已有后端、客户端和前端用 `--targets=service,client,frontend`，有 sidecar 再加 `sidecar`。

## 生成与产物

1. 对 proto/生成器改动，先为所选目标运行 `doctor`，区分环境缺失与生成失败。
2. 构建所有受影响目标，检查生成签名、JSON 名称、FieldMask、分页链接、响应映射和用户代码保留情况，再适配项目代码。
3. 运行相关契约/传输或代表性项目测试；编译通过不能证明线协议、授权或生命周期正确。

API 阶段产出 Go protobuf/gRPC、按源文件相对路径组织的 `docs/**/*.md`、`openapi/**/*.yaml` 和 `openapi/**/*.schema.json`，以及 `.fino/ir/manifest.json`、镜像 `proto/` 的 `.fino/ir/**/*.fino.ir.json`。后端生成前会校验 IR 清单与载荷哈希。缺失或不一致时刷新 API，不能手改 IR。

模型及消息生命周期检查见[契约](contracts.md)。

## 最小验证

| 改动 | 验证 |
|---|---|
| Proto/API | 构建所有已有且受影响目标，检查 diff，验证变化的契约/传输行为 |
| 后端实现 | `gofmt` 修改的 Go 文件，运行受影响包测试；跨包时在 `backend/` 执行 `go test ./...` |
| 启动装配、生成签名、依赖组合 | 后端测试，并在 `backend/` 执行 `go vet ./...`、`go build ./cmd/<service>-server` |
| Client 生成 | 构建 `client`，测试生成的 Go 模块或受影响消费者 |
| 前端实现 | 现有检查/构建脚本，标准脚手架为 `cd frontend && npm run build` |
| 前端生成契约/模板 | 构建 `frontend` 及其他受影响目标，再做前端检查/构建 |
| 设置/工具链 | 对相关目标运行 `doctor` |
| 共享生成器/模板 | 聚焦包测试，在隔离输出目录生成小型代表项目并检查产物及扩展点保留 |
| 共享运行时 | 在所属库运行边界测试并验证受影响调用链；仅当输出或依赖变化时重新生成 |
| Skill 文本 | 本文末尾的 skill 校验与安装比较，无需生成或构建后端/前端 |

组合改动组合检查，复用现有测试/Make 目标，最后运行 `git diff --check`；报告实际结果及未验证边界。

## 首个失败点

| 失败层 | 首要动作 |
|---|---|
| 前置条件 | 用 `doctor` 定位缺少的命令或无效项目文件 |
| Proto 编译 | 修复契约/import，再构建受影响目标 |
| IR/生成 | 刷新 API，检查清单、哈希、依赖或模板不一致 |
| 编译 | 先确认产物与契约一致，再调整必要的项目自有签名/调用方 |
| 运行时 | 沿 `transport -> middleware -> app -> domain -> repository/dependency` 追踪 |

后续阶段失败时可能留下部分产物，以首个失败命令为依据，不能用“文件存在”代替成功。运行时只启动受影响服务与必要依赖，验证健康检查或相关 RPC；清理遵循项目工作项协议。

Go 服务生成会刷新 `deploy/compose` 和 `deploy/helm`，Fino 源码对应 `deploy/generator` 和 `build/deploy`。部署资产与版本发布产物 `.fino/release/` 分开管理。

发布变化要核对 `.fino/frontend`、`frontend/Dockerfile`、`.goreleaser.yaml` 与 `deploy/compose/compose.yaml` 的一致性；同时存在应用和基础设施 compose 时分别验证。

## 生成漂移（按需）

复用现有门禁和 Fino 命令；仅用户要求建设自动化时新增脚本或 CI，基础设施验证放集成目标。需要检查漂移时：

1. 将当前契约、配置、必需输入（含相关未提交/新增文件）复制到临时目录，排除 `.git`、缓存、凭据和运行数据。工具及依赖版本保持一致。
2. 执行 `fino build --output=<temporary-project> --targets=<owned-targets>`。
3. 比较明确的受管清单：IR、protobuf/文档/OpenAPI/schema、生成客户端、`frontend/generated/` 和其他 `DO NOT EDIT` 文件。比较预期与现有路径的并集，暴露新增、变更、过期文件。
4. 混合代码和项目自有文件只检查需协调的变化，不整体比较 `backend/` 或 `frontend/` 作为失败判据。报告首个不同路径及责任源，不明归属的过期文件先确认再删。

无清单时结合目标路径与文件头界定范围，并报告限制。建立默认生成基线也使用这种隔离方式，不能覆盖已演进项目。

## 已发布契约的兼容性

已发布契约变化必须检查兼容性，范围见[契约](contracts.md)。使用同一工具链，将 descriptor/IR 与指定 commit/tag 基线比较；缺少基线或比较器时检查源 diff，并明确未做自动兼容性验证。

破坏性变更（含事件 subject/durable consumer）需要版本或迁移决策；descriptor 检查不能替代授权、错误映射、数据库或消息代理的运行测试。

## 更新内置 skill

事实源是 Fino 仓库的 `skills/generator/templates/skills/fino/`。先改模板；下游同步属于本次范围时，比较并保留项目定制，不因拿项目作参考就覆盖它的 skill。

使用可用的 skill 校验器检查 frontmatter，并确认参考链接有效。然后从 Fino 仓库安装到临时项目并比较五份文件：

```bash
skill_output=$(mktemp -d)
go run ./cmd/fino create skills --output="$skill_output"
diff -r skills/generator/templates/skills/fino "$skill_output/skills/fino"
```

安装器默认保留已有文件；`--force` 覆盖所有内置 skill 的对应文件，不仅是 `fino`。更新单个 skill 时合并其差异，不能用全量强制覆盖替代同步。仅改 skill 文本不需要运行 `fino build`。
