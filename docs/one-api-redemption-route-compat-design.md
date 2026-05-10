# One-API Redemption 路由兼容设计

## 背景

`one-api` 管理兑换码使用 `/api/redemption/*` 路由。当前项目已经实现了 billing redeem code 能力，并在 admin-api 暴露 `/v1/redeem-codes*`，但缺少 One API 的 `/api/redemption/*` 路径兼容。

## 目标

补齐基础路由兼容：

- `GET /api/redemption/`
- `GET /api/redemption/search`
- `GET /api/redemption/:id`
- `POST /api/redemption/`
- `PUT /api/redemption/`
- `DELETE /api/redemption/:id`

继续使用 admin Bearer token 鉴权。

## 非目标

本期不做：

1. One API 以数字 id 为主的 redemption 数据模型迁移。
2. 导出兑换码。
3. 前端兑换码管理页面。

## 架构

admin-api 新增 `/api/redemption/` 路由适配层，复用现有：

- `handleRedeemCodes`
- `handleRedeemCodeByCode`

由于当前 billing redeem code 主键是 `code`，One API 路径 `:id` 在本项目中解释为兑换码 code。

## 测试策略

1. 未授权请求返回 401。
2. `GET /api/redemption/` 代理到列表。
3. `GET /api/redemption/search?keyword=...` 代理到搜索。
4. `DELETE /api/redemption/:code` 调用删除兑换码。

## 验收标准

1. `go test ./internal/admin/server -run 'TestAdminHTTPRedemption' -count=1` 通过。
2. `go test ./...` 通过。
3. `go build ./...` 通过。
