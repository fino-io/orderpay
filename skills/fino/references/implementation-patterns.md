# 后端实现

用于项目自有 RPC、领域、仓储、映射、中间件、下游客户端及 Go 排版。契约规则见[契约](contracts.md)，命令与启动排查见[构建与验证](build-and-verify.md)。按受影响请求路径阅读，不先通读目录树。

## 入口与扩展点

启动路径：`backend/cmd/<service>-server/main.go`（进程、信号、失败处理）→ `bootstrap.go`（配置、观测、依赖、schema、应用组合）→ `backend/modules/<service>-server/api/run.go`（endpoint、HTTP/gRPC/debug、发现、订阅、关闭）。

请求路径：`transport -> protocol middleware -> endpoint middleware -> endpoint -> app -> domain service/model -> repository port -> infra/DAO 或外部适配器`。沿路径检查类型、context、错误转换、副作用和归属。

下表路径相对 `backend/modules/<service>-server/`，其他路径单独注明；以当前版本文件头和生成器写入策略为准。

| 位置 | 编辑方式 |
|---|---|
| `app/app.go` | 生成的 RPC 入口与项目方法体混合；保留签名和 `implements Interface`、`fino:req-fields` 注释标记 |
| `app/mapper/*_mapper_gen.go` | 受管映射；自定义转换放同包独立文件 |
| `domain/model/*_gen.go` | 受管模型；行为与 `Filter{IDs}` 等扩展放非生成的 `<entity>.go` |
| `domain/service/`、`domain/repo/`、`infra/repo/` | 检查是否受管或已成为用户扩展点，保持生成签名一致 |
| `infra/repo/mapper/*_mapper_gen.go` | 受管存储映射；自定义映射放独立文件 |
| `api/run.go` | 模板创建后由项目维护，负责 HTTP/gRPC 路由、协议策略和启动装配 |
| `api/endpoint/endpoints.go`、生成传输绑定 | 受管契约代码；endpoint 构造和协议绑定随 proto 更新 |
| `api/messaging.go` | 仅有消息订阅的服务生成，订阅映射随 proto 更新 |
| `backend/cmd/<service>-server/bootstrap.go` | 默认创建后保留，用于项目依赖组合 |
| `backend/pkg/errcode/configs/*.yaml` | 错误定义源，生成流程见下文 |

`api/run.go` 与 `bootstrap.go` 的项目装配会在重建时保留；新增 RPC 会刷新绑定，首次启用消息订阅等启动能力时需协调项目装配代码。

生成代码是骨架，真实用例维护在扩展点。`app.go` 合并方法体，service 文件可能跳过写入；被跳过文件不会自动补齐方法，签名变化时在隔离副本比较并适配。

新项目的自定义 RPC 默认返回 `Unimplemented`，实现后再替换方法体。标准 CRUD 只覆盖模板已映射的字段；逐字段检查 app、domain 和 repo，尤其是列表过滤、删除选项，以及 create input 中需额外处理的密码等字段。不要把传入 mapper 等同于业务字段已经生效。

## 分层与复用

```text
bootstrap -> app -> domain/service -> domain/model
                         \-> domain/repo <- infra/repo -> DAO
```

| 层 | 职责 |
|---|---|
| `bootstrap` | 构造具体依赖并注入 app/service，不在 RPC 中创建依赖 |
| `app` | 请求形态和作用域检查、DTO/mask 映射、调用一个领域用例、响应组合、错误与请求日志 |
| `app/mapper` | 有明确方向的 API↔领域转换、表示层规范化/清洗 |
| `domain/service` | 工作流、跨模型决策、默认值、生命周期、幂等、状态转换，组合 CRUD 和专用端口 |
| `domain/model` | 局部不变量/状态行为，不依赖 transport、DAO、ORM |
| `domain/repo` | 用领域术语定义狭窄端口 |
| `infra/repo`、`infra/repo/dao` | 适配领域端口，DAO/ORM 查询、存储错误转换、数据库约束和事务 |
| endpoint middleware | 共享认证、tenant/locale、校验、追踪、指标、请求级缓存及跨协议中断观测 |
| HTTP/gRPC middleware | 协议 header/metadata、CORS、编解码、恢复、status 转换 |

优先扩展已有项目 service 和普通辅助函数，复用生成 CRUD。只有生成归属或独立用例边界需要时才加封装，真实消费者/替换需求需要时才加接口；不为每个 RPC 创建 service/repository/mapper，也不引入通用策略对象或注册表。

| 用例 | 项目实现重点 |
|---|---|
| 标准 CRUD | 映射、生成 service、服务端默认值、不变量 |
| 组合 create/update | app 映射父子 input，domain 编排写入与事务/补偿 |
| 状态转换 | app 授权，domain 校验转换并设置生命周期字段 |
| 批量写入 | app 校验 envelope/映射条目结果，domain 负责幂等与一致性 |
| 详情/列表 | app 适配查询并组合 view，domain 定义读取策略，repo/DAO 负责查询语义 |

项目自有接口与辅助函数按职责放在同包独立文件，如 `app/auth.go`；不追加到 `app.go`，不为每个接口建包。认证适配留 `infra/`，测试替身留 `_test.go`。RPC 签名及合并标记保留原位，必要时委托职责文件；入口日志只记一次，错误日志留实际处理处。

跨模型用例放 `domain/service/`，重复映射放 `app/mapper/`，读取补全可放 `app/query.go` 或 `app/view.go`；按实际需求选择，不创建空文件。领域决策不移入 app helper。生成模型补齐缺失的创建/更新时间，其他服务端业务默认值由领域 service 设置。

## Go 代码排版

默认采用分段型排版，优先参考项目规范、用户样例和手工调整。参考代码只用于对齐排版，不改变 API、业务语义和生成归属。

- **Import**：标准库单独一组，其他依赖（含项目包）通常合为一组，组内按路径排序并保留别名。已有明确含义的分组和注释（例如 app 的服务 API 分组）可以保留；不为每个库或别名额外分组，也不添加无实际用途的说明。
- **声明与构造**：函数、方法和构造函数使用常规多行函数体，顶层声明之间留一个空行。结构体中的命名字段逐行声明，嵌入类型与自有依赖字段可以分段；空标记结构体保留 `struct{}`。接口方法按实体或用例成组，不逐方法插空行。
- **复合字面量**：多字段配置、输入、实体、View 和构造结果逐字段换行，嵌套配置按层级展开并保留尾逗号。短的单字段参数和空值可以紧凑书写；RPC 响应遵循下文的逐字段多行直接返回约定。
- **职责分段**：按校验、准备、执行或状态更新、返回分段，循环和回调内同样适用。不同约束的校验分别留空行，即使都是相邻的 `if ... return`。
- **块内紧凑**：同一约束的连续检查、相关赋值、调用与错误检查保持相邻；不按语句类型机械插空行。
- **短函数**：短且单一职责的函数保持紧凑，例如 Count 的查询、错误检查和结果返回；函数体内部不额外插空行，也不为最终 return 单独分段。是否分段看职责和可读性，不设固定行数阈值。
- **换行尺度**：长调用优先在参数、复合字面量或回调边界换行；长条件按可读的逻辑组合断行。易读的函数签名和短表达式保持自然长度，不强制每个运算符单独占行。不要为了排版新增变量、拆分函数、增加接口或改变响应构造方式。
- **事务与回调**：循环和回调内沿用同样的职责分段。局部闭包定义与后续调用之间留一个空行；多条语句逐行写，不用分号压缩。闭包、defer 和构造函数的单行函数体按常规块展开，短委托仍保持原有职责，不增加无用层次。

只整理格式时，先检查 diff 和文件头，保留已有修改、签名、注释及 `fino:req-fields` 等合并标记；受管文件通过责任源更新。对项目扩展点不重新生成，先调整有意义的布局，再运行 `gofmt` 和相关检查。审查 diff，确认变化限于 import 排序/分组、空白、换行及等价字段声明展开；范围较大时可比较规范化 AST 和注释，避免把格式整理变成业务重构。同时对照样例检查分段和展开尺度。修改生成模板时同时检查首次输出与已有项目的保留文件，验证流程见[构建与验证](build-and-verify.md)。

## RPC 格式与日志

按职责阶段分段：请求检查 → 授权/归属检查 → DTO 和 mask 映射 → 一个领域用例 → 必要的关联读取 → view/响应。app 不直接调用 DAO/repository；错误在所属阶段就地映射并保留 cause。阶段间留空行，包括请求缺失检查与授权调用之间；紧密 guard/error 分支不机械插空行。

项目自有 Go 代码遵循以下默认约定，项目 formatter、linter 和日志规范优先：

- RPC 第一行记录方法名，如 `logs.Ctx(ctx).Debugw("CreateBook")`。
- 请求缺失、输入无效和预期业务结果用 `Warnw`；意外失败用 `Errorw`；成功写入用 `Infow`，成功读取/列表用 `Debugw`。事件名稳定，信息使用结构化 key/value。
- app 记录请求日志、预期业务结果并映射错误；意外失败优先复用已接入的公共 transport 错误日志，未被公共层覆盖时才在 app 记录。domain/service、domain/model、domain/repo 不加日志；infra/repo 和 DAO 仅按运维规范记录必要技术事件。
- 不记录原始请求、凭据、token、敏感载荷或不透明动态值。
- 每次失败由明确负责的层记录一次，保留操作、必要阶段、稳定 `reason` 和安全标识；不要在 app、endpoint 和 transport 重复记录同一异常。cause 安全时记 `error`，可能含敏感数据时只记类别。公共日志和共享映射不替代用例对预期结果的分类。
- 普通 unary RPC 直接返回响应复合字面量；复合字面量多行、每字段一行，即使只有一个字段。streaming/messaging 按其生成契约。

以下示例复用公共 transport 的意外异常日志；实体、方法和错误符号需替换为项目真实定义，不能直接复制不存在的符号：

```go
// CreateBook implements Interface. fino:req-fields 1=Book
func (s bookServer) CreateBook(ctx context.Context, in *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	logs.Ctx(ctx).Debugw("CreateBook")
	if in == nil {
		logs.Ctx(ctx).Warnw("invalid create_book request", "reason", "request_missing")
		return nil, errcode.NewBookRequestMissing()
	}

	input, err := mapper.ToCreateBookInput(in.GetBook())
	if err != nil {
		logs.Ctx(ctx).Warnw("invalid create_book_input", "error", err)
		return nil, errcode.WrapCreateBookInputInvalid(err)
	}

	entity, err := s.bookSvc.CreateBook(ctx, input)
	if err != nil {
		if core.IsAlreadyExistsError(err) {
			logs.Ctx(ctx).Warnw("failed to create book", "reason", "already_exists", "error", err)
			return nil, errcode.WrapPlatformConflict(err)
		}
		logs.Ctx(ctx).Errorw("failed to create book", "error", err)
		return nil, errcode.WrapCreateBookFailed(err)
	}

	logs.Ctx(ctx).Infow("book created", "book_id", entity.ID())

	return &pb.CreateBookResponse{
		Book: mapper.ToBookView(entity),
	}, nil
}
```

领域 service/repository 同样按上述职责分段，不添加请求日志或传输响应变量。

## 映射、更新与一致性

- 自定义 mapper 名称体现方向，如 `BookInputToCreateInput`、`UpdateBookFromBookInput`、`BookToBookView`、`BooksToBookViews`；沿用项目已有的等价命名。
- 同一 view 复用一个规范的 entity-to-view 映射器。关联集合/嵌套数据通过 app/query 或领域读取用例明确补全，list 保持轻量，detail 保持完整；不要在 create/update/transition/login 中手拼残缺 view。
- 写入与随后读取使用相同投影，除非契约明确允许异步或部分响应；检查时间戳、枚举、optional false/0、作用域和关联数据。
- 在 app 将非空 update mask 映射为领域字段名，拒绝未知路径；保留未选字段，只有实际变更才更新时间。全量替换的例外见契约参考。
- ID、生命周期和服务端状态由领域层负责，客户端只有在契约明确授权时才可写。
- 多次写入需要基础设施真正支持的事务/unit-of-work；否则明确补偿与部分失败。清理动作不等于原子事务。
- 幂等依赖稳定领域键与数据库唯一约束。app 可规范化 input/计算哈希；查询、竞争、重复/冲突决策归 domain/service 或 repo，必要时比较请求哈希并在竞争后重读。
- 过滤、确定性排序、计数、分页数据和 token 语义贯穿 app/domain/repo/DAO/内存适配器。作用域变化重置或拒绝 token，无界过滤交给 repo；有界小数据可暂用聚焦查询 helper 聚合，方便日后迁移。
- 软删除过滤在 repo/DAO 统一实施，明确唯一键和删除后重建策略，不在 handler 补过滤。

## 认证、作用域与上下文

- 每个外部可访问的非公开 RPC 都要有明确认证/授权；生成传输不意味着 endpoint 公开。可能泄露资源存在性时先认证，再查询资源并校验归属/能力。
- 内部 RPC 从请求或可信主体显式传递全部 tenant/app/project 作用域，校验下游返回作用域并拒绝越界；不依赖零值或未经验证的 token。公开与可信内部工作流授权不同时分开，不能为内部调用削弱公开 create/update。
- 区分外部 ID、session/token ID 和存储主键，在所属适配器转换；不能假设 session ID 是行 ID，或在协议要求外部 ID 时暴露存储键。
- context 贯穿数据库/外部调用，携带请求、session、locale、trace、取消等元数据，不作为业务状态容器。
- app 校验外部形态，model/service 执行不变量，数据库约束作最后一致性保障。动态 JSON/Struct 的大小、深度、schema/version、类型和清洗遵循契约参考；持久化前、响应及日志边界都应用策略。
- 多服务配置按服务加载，不合并兄弟服务命名空间。注入下游地址以适配本地/容器环境，不将容器 DNS 固定为本地默认。错误地址或配置串用可能造成“健康但写错依赖”时，补启动/配置测试。

## 错误

先在 `backend/pkg/errcode/configs/*.yaml` 定义需要的错误，再运行：

```bash
cd backend
./pkg/errcode/generate.sh
```

不手改生成的 `backend/pkg/errcode/*.go`。沿用 YAML 的 `appCode`、`bizCode`，使用未占用的本地 code，明确 name、message、description、countInSLA 和需要的 httpStatus。

| 边界/情况 | 规则 |
|---|---|
| 无 cause / 保留 cause | 分别使用生成的 `New` / `Wrap` |
| nil request | `errcode.New<Service>RequestMissing()` |
| 领域业务状态 | 类型错误或命名 sentinel，以 `errors.Is` 判断，映射 service 专用错误 |
| DAO 未找到 | 将 `gorm.ErrRecordNotFound` 转为 `core.NewNotFoundError`；上层用 `core.IsNotFoundError`，不泄漏 GORM |
| 共享传输/认证语义 | 才使用 `Platform*`；客户端需区分的业务结果定义服务错误 |
| 意外失败 | 操作级兜底错误、安全公开消息，保留 cause 用于诊断 |

`pending`、`disabled`、`unverified`、凭据错误、冲突、`forbidden` 等预期状态不能统一变成意外失败。需要下游区分时不用普通 `fmt.Errorf` 丢失类别：

```text
domain sentinel -> app/service errcode -> gRPC status
                -> downstream domain/platform error -> HTTP status + client code
```

app 返回 `errcode.New/Wrap`；跨 RPC 的取消/超时观测放 `api/endpoint/middleware/error.go`，协议转换分别放 gRPC 中间件和 HTTP 公共错误编码器。不要在每个 RPC 重复判断 context 错误，错误 YAML 不配置 gRPC 状态。具体文件归属以当前模板和文件头为准。

- 领域错误只表达业务类别；app 在实际失败处直接处理该调用可能产生的类别，并明确选择操作级兜底错误。日志 reason 只作分类，不作为业务分发依据；不按 RPC 方法名或前缀推断返回错误，不为减少几个分支重建通用错误分发函数或注册表。
- mapper、认证/权限及意外失败已有 cause 时使用 `Wrap`。endpoint 的中断观测沿错误链用 `errors.Is` 判断，仅记录日志并原样返回 response/error，不用 context 常量替换包装错误；不能仅因 `ctx.Err()` 非空就覆盖成功结果或其他业务错误。业务日志类别与返回结果保持一致。
- 协议层识别包装后的取消、超时：gRPC 映射为 `Canceled` / `DeadlineExceeded`，`PlatformServiceUnavailable` / `PlatformGatewayTimeout` 分别映射为 `Unavailable` / `DeadlineExceeded`；HTTP 超时使用 gateway timeout 错误及 504，客户端 context 已取消时不再写响应。保留项目明确约定的 envelope 状态映射（如关闭映射时返回 200），并区分“客户端已取消”与“请求仍有效但下游返回取消错误”。
- gRPC 除取消/超时的专门规则外，先用 `errorx.From(err)` 提取外层业务错误，再对该对象用 `errcode.IsXxx` 选择状态，统一返回其 `Message()`；公开消息不重复硬编码，也不读取带外层诊断前缀的 `err.Error()`。未映射业务错误使用 `Internal`，不能落入 cause 的 gRPC/core 状态；只有非业务错误再走已有 gRPC status、`CodeFromCoreError` 及安全兜底。
- 手写映射只引用实际存在的错误函数。CRUD 错误定义按模型生成，不能仅凭 RPC 名推断；修改模板时覆盖有模型与无模型服务，条件与错误 YAML 一致。

认证/权限使用规范 gRPC status，下游客户端先规范化再交 app 映射。HTTP status 与业务码一致，不能 HTTP 4xx 搭配通用 500 业务码；内部 cause 不暴露给客户端。分别覆盖预期分支和意外兜底。

生成的标准模型 handler 共用服务级操作错误，以下服务本地区间各预留首值：

| 操作 | 区间 | 默认错误名 |
|---|---|---|
| Get | `1010-1019` | `Get<Service>Failed`、`<Service>NotFound` |
| List | `1020-1029` | `List<Services>Failed`、`List<Services>BadPageToken` |
| Create | `1030-1039` | `Create<Service>InputInvalid`、`Create<Service>Failed` |
| Update | `1040-1049` | `Update<Service>InputInvalid`、`Update<Service>Failed` |
| Delete | `1050-1059` | `Delete<Services>Failed` |

这些名称按服务而非模型分配，同一服务的标准模型共享。自定义 RPC 不自动获得操作错误，需自行定义 YAML 并在 app 映射。按实际操作选错误，例如列表的 bad page token 不能套到 Get。

## 验证重点

- domain：用 fake/内存 repo 验证默认值、状态转换、幂等和部分失败，无需传输层。
- app：请求/访问检查、错误映射、响应补全；尤其写入/读取的 view 一致性。
- repo/DAO：过滤、排序、分页、未找到/唯一性错误、mask、软删除。复合谓词和可选条件需验证组合及占位参数绑定，编译不证明 SQL 正确。
- 内存与持久化适配器：同一端口的查询、唯一性、删除、错误语义一致，不能用宽松替身掩盖约束。
- 跨服务认证/错误：分别验证 domain/app、gRPC 转换、下游规范化、HTTP status/业务码。错误重构按受影响路径覆盖外层业务错误包裹 gRPC/core 错误、公开消息不含诊断前缀、错误链与 response 保留、包装后的取消/超时、存活请求的下游取消及 envelope 兼容性；客户端已取消时确认没有写 header/body。单点单测不能代替整条边界。
- 契约/模板：生成片段断言保留关键装配、顺序及模型条件；行为通过传输测试或代表项目验证，避免锁定变量名、if/switch 形式等等价写法。同类错误用例优先合并为表驱动测试。持久化模板、migration 或 bootstrap 改动要用代表项目验证真实建表与读写路径；seed 改动按契约参考验证生命周期。
