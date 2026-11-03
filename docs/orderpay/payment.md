| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  | 主键及稳定渠道幂等键，服务端生成UUIDv7 |
| `merchantId` | `string` |  | 否 |  | 订单所属商户UUID |
| `orderId` | `string` |  | 否 |  | 订单UUID，每单一个支付意图 |
| `amountMinor` | `integer` | `Int64` | 否 |  | 受理金额快照，人民币分，受理后不可修改 |
| `mandateReference` | `string` |  | 否 |  | 上游已签约授权的不可变引用，不保存授权凭证原文 |
| `channelAccountId` | `string` |  | 否 |  | 已与授权绑定的不可变渠道账户配置引用 |
| `status` | [`orderpay.PaymentStatus`](status.md#orderpay-paymentstatus) |  | 否 |  | 支付状态，技术异常保持待确认 |
| `providerReference` | `string` |  | 否 |  | 稳定渠道支付对象标识，无独立对象时使用流水号，首次取得后不可替换 |
| `reasonCode` | `string` |  | 否 |  | 服务端脱敏原因码，含待核查、终态冲突或待补偿标记，不能由普通确认清除 |
| `submittedAt` | `string` | `Timestamp` | 否 |  | 首次准备提交时间，重启不重置 |
| `retryDeadlineAt` | `string` | `Timestamp` | 否 |  | 首次提交前确定的保守幂等重试截止时间，空值禁止再次提交 |
| `submitAttempts` | `integer` | `Int32` | 否 |  | 已预留的提交调用轮次，调用渠道前递增，重启不重置 |
| `nextCheckAt` | `string` | `Timestamp` | 否 |  | 下一次提交或查询时间，正常终态为空，异常终态安排低频核查 |
| `checkAttempts` | `integer` | `Int32` | 否 |  | 累计查询次数，用于退避和核查告警，重启不重置 |
| `createTime` | `string` | `Timestamp` | 否 |  | 创建时间，UTC |
| `updateTime` | `string` | `Timestamp` | 否 |  | 最后更新时间，UTC |
