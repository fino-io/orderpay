# Orderpay Service

服务：`orderpay.v1.OrderpayService`

商户身份只取鉴权上下文。订单号与订单ID分别保证创建和支付幂等。

## HTTP 约定

- 请求 Body、Query 参数和响应数据使用 JSON。
- 响应 `Content-Type` 为 `application/json; charset=utf-8`。
- Body、Query 参数和响应字段名使用 `lowerCamelCase`；Path 参数以 URI 模板中的名称为准。
- `FieldMask` 使用逗号分隔的 `lowerCamelCase` 字段路径，例如 `displayName,description`。
- `Timestamp` 字段使用带时区的 RFC3339 字符串，例如 `2026-08-12T10:30:00.000+08:00`。

## 错误响应

默认 HTTP 模式下，错误响应格式为：

```json
{
  "code": "100060400",
  "message": "bad request"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `code` | string | fino 业务错误码 |
| `message` | string | 错误说明 |
| `details` | array | 可选的错误详情 |

HTTP 状态码与业务错误码分别表达传输状态和业务错误。

## Create Order

### 请求路径

```http
POST /orderpay/v1/orders
```

### 请求参数

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `merchantOrderNo` | `string` |  | 是 |  | 最大长度为 128 |
| `amountMinor` | `integer` | `Int64` | 是 |  | 金额单位固定为人民币分。；最小值为 1；最大值为 100000000 |
| `description` | `string` |  | 是 |  | 最大长度为 200 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `order` | `orderpay.v1.OrderView` |  | 否 |  |  |


## Get Order

### 请求路径

```http
GET /orderpay/v1/orders/{order_id}
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `order_id` | `string` |  | 最大长度为 36；UUID |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `order` | `orderpay.v1.OrderView` |  | 否 |  |  |
| `payment` | `orderpay.v1.PaymentView` |  | 否 |  | 无支付记录时不设置；查询订单详情即可跟踪支付。 |


## List Orders

### 请求路径

```http
GET /orderpay/v1/orders
```

### 请求参数

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `status` | `orderpay.OrderStatus` |  | 否 |  | 可选值：1 2 3 |
| `pageSize` | `integer` | `Int32` | 否 |  | 最小值为 1；最大值为 100 |
| `pageToken` | `string` |  | 否 |  | 最大长度为 512 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `orders` | `Array<orderpay.v1.OrderView>` |  | 否 |  |  |
| `nextPageToken` | `string` |  | 否 |  |  |


## Cancel Order

### 请求路径

```http
POST /orderpay/v1/orders/{order_id}:cancel
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `order_id` | `string` |  | 最大长度为 36；UUID |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `order` | `orderpay.v1.OrderView` |  | 否 |  |  |


## Create Payment

### 请求路径

```http
POST /orderpay/v1/orders/{order_id}/payments
```

### 请求参数

#### Path 参数

| 参数名 | 参数类型 | 格式类型 | 说明 |
|---|---|---|---|
| `order_id` | `string` |  | 最大长度为 36；UUID |

#### Body 请求对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `mandateReference` | `string` |  | 是 |  | 最大长度为 128 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `payment` | `orderpay.v1.PaymentView` |  | 否 |  |  |


## List Audit Events

### 请求路径

```http
GET /orderpay/v1/admin/audit-events
```

### 请求参数

#### Query 参数

| 参数名 | 参数类型 | 格式类型 | 是否必须 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `merchantId` | `string` |  | 否 |  | 最大长度为 36；UUID |
| `resourceId` | `string` |  | 否 |  | 最大长度为 36；UUID |
| `pageSize` | `integer` | `Int32` | 否 |  | 最小值为 1；最大值为 100 |
| `pageToken` | `string` |  | 否 |  | 最大长度为 512 |

### 返回值

#### 返回对象

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `events` | `Array<orderpay.v1.AuditEventView>` |  | 否 |  |  |
| `nextPageToken` | `string` |  | 否 |  |  |


## 数据模型

### `orderpay.v1.OrderView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `merchantOrderNo` | `string` |  | 否 |  |  |
| `amountMinor` | `integer` | `Int64` | 否 |  | 金额单位固定为人民币分。 |
| `description` | `string` |  | 否 |  |  |
| `status` | `orderpay.OrderStatus` |  | 否 |  |  |
| `createTime` | `string` | `Timestamp` | 否 |  |  |
| `updateTime` | `string` | `Timestamp` | 否 |  |  |

### `orderpay.v1.PaymentView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `orderId` | `string` |  | 否 |  |  |
| `status` | `orderpay.PaymentStatus` |  | 否 |  |  |
| `providerReference` | `string` |  | 否 |  |  |
| `amountMinor` | `integer` | `Int64` | 否 |  | 本次支付受理时的人民币分金额快照。 |
| `reasonCode` | `string` |  | 否 |  | 服务端定义的脱敏原因码；空值表示无异常，不透传渠道错误。 |
| `createTime` | `string` | `Timestamp` | 否 |  |  |
| `updateTime` | `string` | `Timestamp` | 否 |  |  |

### `orderpay.v1.AuditEventView`

| 字段 | 类型 | 格式 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `id` | `string` |  | 否 |  |  |
| `merchantId` | `string` |  | 否 |  |  |
| `actorId` | `string` |  | 否 |  |  |
| `action` | `string` |  | 否 |  |  |
| `resourceId` | `string` |  | 否 |  |  |
| `requestId` | `string` |  | 否 |  |  |
| `beforeStatus` | `string` |  | 否 |  |  |
| `afterStatus` | `string` |  | 否 |  |  |
| `reasonCode` | `string` |  | 否 |  |  |
| `evidenceReference` | `string` |  | 否 |  | 仅管理员可见的受控证据引用，不包含凭证或原始支付报文。 |
| `createTime` | `string` | `Timestamp` | 否 |  |  |


## 枚举

<a id="orderpay-orderstatus"></a>

### `orderpay.OrderStatus`

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `ORDER_STATUS_UNSPECIFIED` | 0 |  |
| `ORDER_STATUS_PENDING_PAYMENT` | 1 |  |
| `ORDER_STATUS_PAID` | 2 |  |
| `ORDER_STATUS_CANCELLED` | 3 |  |

<a id="orderpay-paymentstatus"></a>

### `orderpay.PaymentStatus`

| 枚举值 | 数值 | 说明 |
|---|---:|---|
| `PAYMENT_STATUS_UNSPECIFIED` | 0 |  |
| `PAYMENT_STATUS_CREATED` | 1 |  |
| `PAYMENT_STATUS_PENDING` | 2 |  |
| `PAYMENT_STATUS_SUCCEEDED` | 3 |  |
| `PAYMENT_STATUS_DECLINED` | 4 |  |
