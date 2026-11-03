# 契约

用于 proto/API、模型、校验、HTTP 绑定和消息订阅。目标与生成命令见[构建与验证](build-and-verify.md)，业务实现见[后端实现](implementation-patterns.md)。

## 来源、服务与布局

- `proto/` 是契约事实源；包与依赖见 `package.yaml`，服务和基础设施见 `fino.yaml`。生成目标及命令见构建参考。
- 保持 `package.repository` 稳定，除非有意迁移生成代码 import 路径。
- 一个 service 对应一个业务边界和后端模块；例如 `UserService` 对应 `user-server`。模型的 services、配置键、路径和 receiver 按当前生成命名规则对应，修改规则时同步引用。
- 持久化模型放在 `proto/<package>/`，可按模型分文件；共用领域枚举也放在该目录。
- `proto/<package>/v1/` 中一个文件对应一个 service，文件内先定义 service，再按 RPC 声明顺序放置对应的 Request、Response 及专用 Input/View。共享 DTO 在首次使用处定义一次，后续复用；同一服务的参数不另拆到多个文件，也不为缩短文件而拆分业务服务。
- 模型用于持久化，API input/view 用于写入和读取，不能直接把持久化记录暴露为 RPC 载荷。派生/聚合值放只读 view，不做持久化模型或更新输入。
- `fino create` 的 `resource.proto` 和配套 Resource RPC 仅供参考。业务契约替换示例后，清理未使用的模型、RPC、请求响应、import 和已失效的生成产物；不要无依据将示例当作兼容接口保留。若已被真实调用或发布，则先处理兼容性与迁移，不直接删除。

## API 形态与兼容性

新 RPC 使用 `<Action>Request` / `<Action>Response`，输入和视图使用 `<Entity>Input` / `<Entity>View`；无载荷的删除返回 `fino.core.Null`。已有调用方直接接收 View 时保持原形态，不为统一风格破坏兼容性。

- 客户端可写字段与服务端 ID、生命周期时间戳、审计/派生字段分开。后者只读或由领域层赋值；状态转换只有在契约明确暴露时才允许客户端触发。
- 已发布字段编号不复用；删除时用 `reserved` 保留编号和名称。请求形态、必填性、`json_name`、类型/cardinality、枚举数值、响应存在性、HTTP 绑定和事件 subject 的变化都需检查兼容性。
- 旧客户端不能继续解码/调用时，优先增量演进、明确的兼容别名或新 RPC/版本。必要的破坏性变化要同步生产者、消费者、客户端、OpenAPI、前端及迁移/双读双写安排。

## 模型与 Fino 选项

| 选项 | 用途 |
|---|---|
| `(fino.model)` | 标记数据库模型，选择生成及目标服务 |
| `(fino.db)` | 持久化元数据，仅用于模型 |
| `(fino.validate)` | 外部请求参数和写入 input 的校验，不用于数据库模型字段 |
| `(fino.messaging)` | 标记消费消息的 RPC |

示例片段，按标识、基础信息和生命周期分组；已发布契约遵守兼容性规则：

```proto
import "fino/core/time.proto";
import "fino/options.proto";

message Resource {
  option (fino.model) = {generate: true services: "ResourceService"};

  string id = 1 [(fino.db) = {len: 36 comment: "主键"}];
  string key = 2 [(fino.db) = {len: 128 unique_index: "idx_resource_key" comment: "资源唯一标识"}];
  string name = 3 [(fino.db) = {len: 128 default: "" comment: "资源名称"}];

  fino.core.Timestamp create_time = 98 [(fino.db) = {comment: "创建时间"}];
  fino.core.Timestamp update_time = 99 [(fino.db) = {comment: "最后更新时间"}];
  fino.core.Timestamp delete_time = 100 [(fino.db) = {index: "" comment: "软删除时间，未设置表示未删除"}];
}
```

持久化模型的每个字段用 `(fino.db).comment` 说明业务含义，包括主键、关联标识、状态和生命周期时间；必要时补充单位、空值或枚举含义。Proto 的 `//` 注释用于补充 API/实现语义，不能替代数据库字段注释。

显式声明 ID 和生命周期字段；`(fino.model)` 不会补字段。软删除字段使用 `fino.core.Timestamp`，未声明则物理删除。字符串默认长度为 255，显式 `len` 应覆盖输入允许的长度；稳定业务标识使用 `unique_index`，常用过滤字段使用 `index`。

UUID ID 约定：

- 生成模型用 `google/uuid.NewV7()` 分配 `string` 主键，禁止自增。新契约的主键及关联 ID 设 `len: 36`；凭证引用、版本标识等不套用此长度。
- 必填请求 ID 使用 `{required: true max: 36 uuid: true}`；可选或局部更新 ID 使用 `{omitempty: true max: 36 uuid: true}`。ID 集合逐项校验格式、去重和归属。
- `uuid: true` 只校验格式，v7 由服务端生成保证；已有契约收紧长度或校验时，按兼容性要求处理。

稳定结构用类型明确的字段。只有确需存储不透明序列化 JSON 时才用 `*_json` 字符串并记录 schema；不能代替类型字段/索引，也不能将原始 JSON 暴露为任意过滤或排序键。

## HTTP 与更新

- 集合路径用复数；归属关系用嵌套路径，如 `/resources/{parent_id}/children`；状态动作用 `:activate`、`:archive` 等后缀。仅 gRPC 方法不加 HTTP 注解。
- 整个请求为 body 时用 `body: "*"`；path/query 字段位于嵌套 body 外时用 `body: "<field>"`。
- 生成模型更新要求非空 `update_mask`：指定字段或 `*`，拒绝空路径和未知路径；授权及字段策略由应用实现。仅不使用生成映射器的自定义全量替换 RPC 可省略 mask。

```proto
import "google/api/annotations.proto";
import "google/protobuf/field_mask.proto";

service ResourceService {
  rpc UpdateResource(UpdateResourceRequest) returns (UpdateResourceResponse) {
    option (google.api.http) = {
      patch: "/api/v1/resources/{resource_id}"
      body: "resource"
    };
  }
}

message UpdateResourceRequest {
  string resource_id = 1 [(fino.validate) = {required: true max: 36 uuid: true}];
  ResourceInput resource = 2 [(fino.validate) = {required: true}];
  google.protobuf.FieldMask update_mask = 80;
}

message UpdateResourceResponse {
  ResourceView resource = 1;
}
```

HTTP 解码/编码、OpenAPI、客户端/前端路由元数据和分页链接必须描述同一线协议：

| 内容 | 约定 |
|---|---|
| Query 名 | 使用 protobuf `jsonName`，默认 lowerCamelCase；显式 `json_name` 优先 |
| Path 变量 | 遵循 `google.api.http` 绑定字段名，不从 query 规则另推名称 |
| FieldMask query 值 | 逗号分隔的 lowerCamelCase 路径或 `*`，解码到 `FieldMask.Paths`；不要要求 JSON 对象 |
| 分页 | 请求 `pageToken`，响应 `nextPageToken`；`Link` header 使用请求参数名并保留过滤/排序作用域 |
| JSON/schema | 与实际序列化器的枚举名称、时间戳、optional 存在性、响应 envelope 内外行为一致 |

修改共享 HTTP 模板时，既断言生成绑定，也测试最终传输行为。详情可组合较小 view、关联数据和只读摘要；同一 view 的写入响应与读取响应遵循相同投影，具体映射见后端参考。

## 校验与分页

必填资源标识放 request，可选过滤保持可选。省略/零值应跳过校验时用 `omitempty`，尤其数字字段的 `min`、`max`、`oneof`；不与 `required` 组合。必须区分省略与显式 false/0 时用 proto3 `optional`。

```proto
message ListResourcesRequest {
  string key = 1 [(fino.validate) = {omitempty: true max: 128}];
  int32 page_size = 90 [(fino.validate) = {omitempty: true min: 1 max: 100}];
  string page_token = 91 [(fino.validate) = {omitempty: true max: 512}];
}

message ListResourcesResponse {
  repeated ResourceView resources = 1;
  int32 total_count = 92;
  string next_page_token = 93;
}
```

生成的通用列表按请求与模型同名、同类型的字符串、整数或枚举字段做非零值精确筛选；示例中的 `key` 与模型字段对应。`keyword`、范围、排序和关联过滤需要项目实现明确的查询逻辑，不能只在请求里加字段。只有响应暴露 `total_count` 和/或 `next_page_token` 才添加分页，保持完整请求/响应语义。声明 `total_count` 会让生成仓储执行精确计数；不需要总数的接口可只暴露 `next_page_token`，生成代码用 `pageSize+1` 判断下一页并跳过 `COUNT(*)`。排序必须确定，token 绑定完整过滤和排序作用域；过滤、计数、分页数据和 next token 来自同一服务端查询，不能只过滤前端当前页。有界小数据集可用 offset；大量或并发变化数据造成 offset 不稳定时使用 cursor/keyset。

## 字段编号

简单消息的业务字段从 `1` 开始连续编号，相关字段相邻；公共字段使用下表推荐编号。复杂模型及其较大的 View 可按实际职责局部分组，例如从 `10`、`20` 开始，适当留出空位；不为简单消息或整个项目划定统一业务号段。

| 用途 | 推荐编号 |
|---|---|
| `update_mask` | `80` |
| `page_size` / `page_token` | `90` / `91` |
| `total_count` / `next_page_token` | `92` / `93` |
| `create_time` / `update_time` / `delete_time` | `98` / `99` / `100` |

- `create_time = 98`、`update_time = 99`、`delete_time = 100` 保持固定；其余业务字段不占用上述公共字段编号。
- `version`、`expected_version`、请求幂等标识、`server_time` 等按所在消息分配普通业务编号，不规定全局固定编号。
- 局部分组只在职责清楚且方便扩展时采用，用空行和简短业务注释区分；不要求不同消息的同类字段同号，也不要求小型 Input 与复杂持久化模型对齐。模型与对应复杂 View 可复用分组。
- 未使用的空号留给后续扩展，不写 `reserved`；删除已发布字段时才保留其编号和名称。枚举值不套用字段分组。
- 已发布契约不为整理编号而重排；用户明确授权破坏性调整时，同步刷新受影响产物并说明兼容性边界。这是契约设计约定，不增加生成器自动分配机制。

## 动态载荷（仅在需要时）

仅扩展属性使用 `google.protobuf.Struct`；资源、版本、主体、请求和时间等可搜索/审计字段保持明确类型。

在应用边界约束大小、深度、允许类型；持久化前及响应/日志边界再次应用 schema、版本和清洗规则。字段稳定、需要查询或独立授权时改为明确消息。批量操作用结构化 repeated view（如 `FieldViolationView`）表达校验问题和已接受/拒绝条目。

## Schema 与消息生命周期

模型/字段变化后，检查 migration、seed、upsert、fixture 和 bootstrap。删除列或值的引用必须一起处理；不修补生成 schema/seed 来掩盖契约不一致。明确这些文件的项目归属，验证当前 schema 启动，并在可行时验证全新 schema 启动。

消息订阅在 `fino.yaml` 为目标服务启用提供方，并标记消费 RPC：

```yaml
backend:
  resource-server:
    infra:
      messaging: nats
```

```proto
rpc StartTask(StartTaskRequest) returns (StartTaskResponse) {
  option (fino.messaging) = {subscription: true};
}
```

proto 选项只表达是否订阅；topic、consumer group、NATS 投递设置放在 `backend/configs/<service>-server/messaging.yaml`。生成启动代码初始化 messaging 并调用 `SubscribeService`，业务处理按 endpoint/app/domain 实现，不修改生成路由。

NATS/JetStream subject 使用稳定、完全限定且带版本的名称，如 `product.resource.event.v1`。生产者与消费者配置一致，durable consumer 身份绑定 subject 版本。subject 不兼容变化时使用新版本 consumer 或显式迁移；只改 subject 会留下旧过滤条件。启动时不隐式清理/删除已有 durable consumer。其他 broker 遵循等价的 topic/consumer 版本规则。

共性失败在契约/模板层回归，业务断言留项目；生成与验证见构建参考。
