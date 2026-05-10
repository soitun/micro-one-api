# One-API Option 路由兼容设计

## 背景

`one-api` 管理端通过 `/api/option/` 读取和更新系统选项。当前项目已有 `/v1/system/options` 和 `AdminService.GetSystemOptions/UpdateSystemOptions`，但缺 One API 路径兼容。

## 目标

补齐：

- `GET /api/option/`
- `PUT /api/option/`

继续使用 admin Bearer token 鉴权。

## 非目标

本期不做：

1. One API 所有 option key 的完整迁移。
2. Root/Auth 角色体系细分。
3. 前端设置页面。

## 架构

在 `admin-api` HTTP server 中新增 `/api/option/` 路由，复用现有 system options service。

兼容请求：

```json
{
  "site_title": "My API",
  "registration_enabled": false
}
```

也兼容现有 protobuf JSON：

```json
{
  "options": {
    "site_title": "My API",
    "registration_enabled": false
  }
}
```

响应使用 One API 风格：

```json
{
  "success": true,
  "message": "",
  "data": {
    "site_title": "My API",
    "registration_enabled": false
  }
}
```

## 测试策略

1. 未授权请求返回 401。
2. `GET /api/option/` 返回 One API 风格 data。
3. `PUT /api/option/` 接受扁平 JSON 并更新配置。

## 验收标准

1. `go test ./internal/admin/server -run 'TestAdminHTTPOption' -count=1` 通过。
2. `go test ./...` 通过。
3. `go build ./...` 通过。
