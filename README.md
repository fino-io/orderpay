# Orderpay

基于 Fino、Go 和 SQLite 的支付 / 订单服务。接口和前端 API Console 由 proto 生成，业务规则在框架扩展点实现。

当前完成本地订单、幂等、取消、商户隔离、限流及审计。真实授权服务和支付渠道未接入，正式启动的支付入口返回 **503：payment channel is disabled**，不会产生扣款或模拟成功。支付受理与状态规则通过独立测试授权验证。

前端提供订单列表、创建、详情/取消及管理员审计页面；支付只读展示，凭证仅保存在当前页面内存。

- [架构设计](docs/architecture.md)
- [前端页面设计](docs/frontend-design.md)
- [验收记录与边界](docs/acceptance.md)
- [接口契约](proto/orderpay/v1/orderpay.proto)
- [生成接口文档](docs/orderpay/v1/orderpay.md)

## 生成与验证

使用 Fino v1.7.4；Go 版本以 `backend/go.mod` 为准，允许 Go 自动下载所需工具链。Node.js 要求见 `frontend/package.json`。

```sh
fino doctor --output=. --targets=api,service,frontend
fino build --output=. --targets=api,service,frontend
```

```sh
cd backend
go test ./...
go test -race ./modules/orderpay-server/app ./modules/orderpay-server/infra/auth
go vet ./...
go build ./cmd/orderpay-server
```

```sh
cd frontend
npm ci
npm test
npm run build
```

不手改 `DO NOT EDIT` 文件，修改生成契约时先更新 proto。业务代码位置、排版与验证流程遵循 `skills/fino/SKILL.md`。

## 本地启动

```sh
cd backend
cp .env.example .env
```

在 `.env` 中设置 `ORDERPAY_API_KEYS`，以 JSON 映射私有服务密钥与主体。商户密钥至少32位，merchantId必须为UUID；管理员使用独立密钥。以下仅展示格式，需替换密钥：

```dotenv
ORDERPAY_API_KEYS='{"replace-with-private-merchant-secret":{"merchantId":"019a4a70-0000-7000-8000-000000000001"},"replace-with-private-administrator-secret":{"admin":true}}'
```

```sh
go run ./cmd/orderpay-server
```

端口见 `.env`，默认 HTTP 31101、gRPC 31102、Debug 31103；SQLite 数据保存在 `backend/orderpay.db`。认证使用生成框架现有协议：HTTP `Authorization: Bearer <key>`，gRPC metadata `x-session-key: <key>`。管理员只能访问管理员接口，商户不能指定其他商户身份。

```sh
cd frontend
npm ci
npm run dev -- --host 127.0.0.1 --port 5173
```

打开 `http://127.0.0.1:5173/orders`，在连接设置中选择商户模式并输入服务凭证；管理员选择审计模式后手动查询。凭证刷新页面即清除，工作模式不授予权限。

开发配置自动读取 `backend/.env` 的 HTTP 端口；后端仅放行 localhost/127.0.0.1:5173 两个开发来源。页眉直接提供 API Console、Issues、Monitor、Logs 和 Trace，Console 的请求 Header 独立配置。正式部署需接入浏览器身份体系，并由网关转发 `/orderpay/v1` 或配置实际 API 地址与 CORS；当前 Caddy 仅代理监控路径。

创建订单的 HTTP 请求体是 OrderInput，例如：

```json
{"merchantOrderNo":"demo-001","amountMinor":1990,"description":"本地测试订单"}
```

同商户同订单号、同内容重复提交返回同一订单，内容不同返回409。支付渠道未启用时不会创建支付记录，可继续取消订单。只保存必要服务凭据，不将 `.env`、数据库或调试日志提交到仓库。

## 本地演示数据

后端初始化本地 SQLite 后，可导入 30 笔订单、24 笔模拟支付与 54 条导入审计，覆盖分页、正常状态及异常核查：

```sh
cd backend
go run ./cmd/seed-demo
```

默认使用本地 `orderpay.db` 和示例商户，可通过 `-db`、`-merchant` 指定。订单号以 `MOCK-DEMO-` 开头，描述标明模拟；重复运行跳过已有演示订单，保留其他数据。支付使用禁用的模拟账户，不调用渠道，也不参与调度。此工具只用于本地界面演示。
