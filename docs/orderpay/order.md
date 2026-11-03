| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键，服务端生成UUIDv7 |
| `merchantId` | `string` |  | 否 |  | 已认证商户UUID |
| `merchantOrderNo` | `string` |  | 否 |  | 商户业务订单号，同时作为创建幂等标识 |
| `amountMinor` | `integer` | `Int64` | 否 |  | 人民币分，范围1至100000000，创建后不可变 |
| `description` | `string` |  | 否 |  | 订单描述，创建后不可变 |
| `status` | [`orderpay.OrderStatus`](status.md#orderpay-orderstatus) |  | 否 |  | 订单状态，仅领域用例可修改 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间，UTC |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间，UTC |
