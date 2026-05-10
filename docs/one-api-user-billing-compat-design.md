# One-API 用户 Dashboard 与 TopUp 兼容设计

## 背景

`one-api` 用户端包含两个常用自助账务接口：

- `GET /api/user/dashboard`：查看当前用户额度、已用额度、请求次数等信息。
- `POST /api/user/topup`：使用兑换码为自己充值。

当前 `micro-one-api` 已实现 identity-service 的用户注册、登录、self、token、aff，以及 billing-service 的账户快照、兑换码和充值能力，但还没有把用户自助账务入口暴露为 One API 风格 HTTP API。

## 目标

补齐用户自助账务兼容入口：

1. `GET /api/user/dashboard`
2. `POST /api/user/topup`

响应保持 One API 风格：

```json
{
  "success": true,
  "message": "",
  "data": {}
}
```

## 非目标

本期不做：

1. 完整 billing usage 图表。
2. 支付平台充值链接。
3. 前端页面改造。
4. 兑换码导出或批量管理，这些属于 admin API。
5. session/cookie 鉴权，本项目继续使用 Bearer token。

## 架构

当前 `/api/user/*` 路由在 identity-service HTTP server 内。Dashboard 和 topup 需要 billing-service 能力，因此在 identity-service 中新增可选的 billing client 依赖：

```go
func NewHTTPServer(addr string, uc *biz.IdentityUsecase, oauthRegistry *oauth.ProviderRegistry, billingClient ...billingv1.BillingServiceClient) *khttp.Server
```

这样可以保持现有调用点兼容：

- 未传 billing client 时，账务接口返回 503。
- identity-service wiring 可在后续接入真实 billing client。
- 单元测试可注入 fake billing client。

## API 设计

### GET /api/user/dashboard

鉴权：

- `Authorization: Bearer <token>`

流程：

1. 通过现有 `authSnapshotFromRequest` 获取用户 ID。
2. 调用 `billing.GetAccountSnapshot(user_id)`。
3. 返回 One API 兼容 dashboard 数据。

响应 data 建议字段：

```json
{
  "quota": 1000,
  "used_quota": 100,
  "request_count": 10,
  "group": "default",
  "group_ratio": 1,
  "frozen_quota": 0
}
```

### POST /api/user/topup

请求：

```json
{
  "key": "REDEEM-CODE"
}
```

流程：

1. 通过 Bearer token 获取用户 ID。
2. 校验 `key` 非空。
3. 调用 `billing.RedeemCode(user_id, key)`。
4. 成功时返回兑换额度。

成功响应：

```json
{
  "success": true,
  "message": "",
  "data": 1000
}
```

失败响应：

```json
{
  "success": false,
  "message": "..."
}
```

## 错误处理

- 未授权：HTTP 401，`success=false`。
- billing client 未配置：HTTP 503，`success=false`。
- key 为空：HTTP 200，`success=false`。
- billing 返回 `success=false`：HTTP 200，透出 `error_message`。
- billing gRPC error：HTTP 200，`success=false`，message 为错误文本。

## 测试策略

1. HTTP server 测试：
   - dashboard 未授权返回 401。
   - dashboard 未配置 billing 返回 503。
   - dashboard 成功返回 quota、used_quota、request_count。
   - topup 未授权返回 401。
   - topup 空 key 返回 `success=false`。
   - topup 成功返回兑换额度。
   - topup billing 失败返回 `success=false`。

2. 集成测试：
   - 现有 integration fake identity repo 继续编译。
   - 不要求启动完整 billing-service。

## 文档更新

实现完成后更新：

- `docs/one-api-full-gap-analysis-20260509.md`
- 必要时更新 README 的 API 测试示例。

## 验收标准

1. `go test ./internal/identity/server -count=1` 通过。
2. `go test ./...` 通过。
3. `go build ./...` 通过。
4. `/api/user/dashboard` 和 `/api/user/topup` 均有鉴权边界测试。
