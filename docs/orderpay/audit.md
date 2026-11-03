| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键，UUIDv7，游标及归档去重标识 |
| `merchantId` | `string` |  | 否 |  | 资源所属商户 UUID，系统事件可为空 |
| `actorId` | `string` |  | 否 |  | 已认证操作者或系统主体标识 |
| `action` | `string` |  | 否 |  | 业务操作名称 |
| `resourceId` | `string` |  | 否 |  | 业务资源 UUID |
| `requestId` | `string` |  | 否 |  | 服务端请求关联标识 |
| `beforeStatus` | `string` |  | 否 |  | 变更前状态名称 |
| `afterStatus` | `string` |  | 否 |  | 变更后状态名称 |
| `reasonCode` | `string` |  | 否 |  | 脱敏原因码，不保存凭证或原始报文 |
| `evidenceReference` | `string` |  | 否 |  | 受控核查证据或工单引用，普通业务事件可为空，人工纠正及核查关闭时必填 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间，UTC |
