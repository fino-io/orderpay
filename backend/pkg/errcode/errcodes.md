# 业务错误码

> 此文件由 `errorxgen` 自动生成，请勿手动修改。

本文档是业务错误码的统一索引。模块顺序为：`platform` 优先，其它模块按定义文件名升序排列；同一模块内按错误子码升序排列。

## 编码与字段约定

- `错误码`：稳定的机器可读业务标识，格式为 `appCode(1位) + bizCode(4位) + code(4位)`；客户端和服务间应按错误码判断，不要按消息文本判断。
- `Reason / 名称`：稳定的业务原因名称，用于代码调用、日志检索和错误判断；名称一旦发布不应复用。
- `消息模板`：面向调用方或用户的可读消息，可包含 `{key}` 占位符；不承载底层异常和敏感信息。
- `场景说明`：说明错误何时发生，帮助调用方和维护者正确处理。
- `HTTP 状态`：协议层映射，不参与业务错误码唯一性；未配置时使用 `500`。
- `SLA 统计`：仅用于稳定性指标和告警策略；未配置时使用 `是`。运行时的 `extra/metadata` 属于单次错误上下文，不在错误码目录中固定。

## 总览

| 业务模块 | 定义文件 | appCode | bizCode | 错误数 | 错误码范围 |
| --- | --- | ---: | ---: | ---: | --- |
| platform | platform.yaml | 1 | 6 | 10 | 100060400 - 100060504 |
| orderpay | orderpay.yaml | 1 | 6 | 17 | 100060601 - 100061072 |

## platform

| 错误码 | Reason / 名称 | 消息模板 | 场景说明 | HTTP 状态 | SLA 统计 |
| ---: | --- | --- | --- | ---: | --- |
| 100060400 | PlatformBadRequest | bad request | request cannot be processed | 400 | 否 |
| 100060401 | PlatformUnauthenticated | authentication required | request has no valid authentication | 401 | 否 |
| 100060403 | PlatformForbidden | permission denied | authenticated caller lacks permission | 403 | 否 |
| 100060404 | PlatformNotFound | resource not found | requested platform resource does not exist | 404 | 否 |
| 100060409 | PlatformConflict | resource conflict | request conflicts with current resource state | 409 | 否 |
| 100060429 | PlatformTooManyRequests | too many requests | request was rate limited | 429 | 否 |
| 100060500 | PlatformInternalServerError | internal server error | unexpected platform failure | 500 | 否 |
| 100060502 | PlatformBadGateway | bad gateway | upstream service returned an invalid response | 502 | 否 |
| 100060503 | PlatformServiceUnavailable | service unavailable | platform dependency is unavailable | 503 | 否 |
| 100060504 | PlatformGatewayTimeout | gateway timeout | platform dependency timed out | 504 | 否 |

## orderpay

| 错误码 | Reason / 名称 | 消息模板 | 场景说明 | HTTP 状态 | SLA 统计 |
| ---: | --- | --- | --- | ---: | --- |
| 100060601 | OrderpayRequestValidationFailed | invalid {field}: validation failed on {rule} | request field validation failed before app processing | 400 | 否 |
| 100061001 | OrderpayRequestMissing | request is required | app handler received a nil request | 400 | 否 |
| 100061011 | GetOrderpayFailed | get resource failed | failed to get resource in service | 500（默认） | 是 |
| 100061012 | OrderpayNotFound | resource not found | requested resource does not exist | 404 | 否 |
| 100061021 | ListOrderpaysFailed | list resources failed | failed to list resources in service | 500（默认） | 是 |
| 100061022 | ListOrderpaysBadPageToken | list resources with bad page token | failed to list resources with bad page token | 400 | 是 |
| 100061031 | CreateOrderpayInputInvalid | create resource input invalid | create resource input cannot be converted to domain input | 400 | 否 |
| 100061032 | CreateOrderpayFailed | create resource failed | failed to create resource in service | 500（默认） | 是 |
| 100061041 | UpdateOrderpayInputInvalid | update resource input invalid | update resource input cannot be converted to domain input | 400 | 否 |
| 100061042 | UpdateOrderpayFailed | update resource failed | failed to update resource in service | 500（默认） | 是 |
| 100061051 | DeleteOrderpaysFailed | delete resources failed | failed to delete resource in service | 500（默认） | 是 |
| 100061061 | CancelOrderFailed | cancel order failed | cancel order failed | 500 | 是 |
| 100061062 | OrderConflict | order content or state conflicts | order content or state conflicts | 409 | 否 |
| 100061063 | PaymentConflict | payment content or state conflicts | payment content or state conflicts | 409 | 否 |
| 100061064 | PaymentUnavailable | payment channel is disabled | payment channel is disabled | 503 | 否 |
| 100061071 | CreatePaymentFailed | create payment failed | create payment failed | 500 | 是 |
| 100061072 | ListAuditEventsFailed | list audit events failed | list audit events failed | 500 | 是 |
