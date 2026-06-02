# 覆盖/缺口清单：前端页面 ↔ 后端路由 ↔ RPC 契约

> 日期：2026-06-02
> 分支：`develop` (cf065aa)
> 方法：前端 `web/src` API 调用清点 + 后端各 `server/http.go` 路由清点 + billing/log/identity/channel proto 与实现清点，三方交叉核对。
> 结论性判断已对可疑项做源码二次验证（见 §4）。

## 0. 一句话结论

**契约层是健康的**：前端实际调用的每个端点都有对应后端实现，且关键响应字段（`trade_no`/`pay_url`、`quota`、ledger `amount`）形状一致，未发现"前端调用了不存在的路由"或字段对不上的破裂。

**真正的缺口是单边的**：一批**后端已完整实现、但前端尚无入口**的能力（OAuth 登录、对账复核、邀请/分销码、找回密码、内容/分组管理、渠道测试）。这些就是 P0 web 前端的剩余工作面，而不是后端 bug。

---

## 1. 主流程覆盖矩阵（前端有页面 → 后端有路由 → 有 RPC/逻辑支撑）

图例：✅ 三方齐全且已验证一致 ｜ ⚠️ 齐全但未做端到端验证

| 业务流 | 前端页面 | 后端路由（base `/api`） | 后端支撑 | 状态 |
|---|---|---|---|---|
| 注册 / 登录 / 登出 | LoginPage | `POST /api/user/{register,login,logout}` | identity 本地逻辑 | ✅ |
| 个人资料查看/修改 | ProfilePage | `GET/PUT /api/user/self` | identity `GetUser`/`UpdateSelf`（PUT 已确认支持，见 §4.1） | ✅ |
| 仪表盘（额度/趋势/模型分布） | DashboardPage | `GET /api/user/dashboard` | billing `GetAccountSnapshot` + `AggregateLedgerByDate` | ✅ |
| Token 管理 | TokensPage / DashboardPage | `GET/POST /api/token`、`DELETE /api/token/{id}` | identity token 逻辑 | ✅ |
| 用量日志 | UsagePage | `GET /api/user/logs?type=consume` | billing `ListLedger` | ✅ |
| 订单 + 账目记录 | OrdersPage | `GET /api/user/payment/orders[/{tradeNo}]`、`GET /api/user/logs` | billing `ListPaymentOrders`/`GetPaymentOrderByTradeNo`/`ListLedger` | ✅ |
| 在线充值（支付宝） | RechargePage | `POST /api/user/pay` | billing `CreatePaymentOrder` → alipay；返回 `trade_no`+`pay_url`（见 §4.2） | ⚠️ 字段已核对，回调闭环未端到端验证 |
| 兑换码充值 | RedeemPage | `POST /api/user/topup` | billing `RedeemCode` | ✅ |
| 价格表 | PricingPage | `GET /api/pricing` | admin/relay 本地（读 options） | ✅ |
| 管理员总览 | admin/OverviewPage | `GET /api/admin/summary` | 聚合 identity+channel+log+billing+config RPC | ✅ |
| 用户管理 | admin/UsersPage | `GET /api/user`、`POST /api/user/{disable,enable}/{id}`、`POST /api/user/manage` | admin `ListUsers`/`UpdateUser` + identity `SetUserRole`（manage 在前缀处理器内，见 §4.3） | ✅ |
| 渠道管理 | admin/ChannelsPage | `GET/POST/PUT /api/channel`、`.../{disable,enable}/{id}`、`/update_balance/{id}` | channel CRUD + 余额刷新 | ✅ |
| 计费/价格配置 | admin/PricingPage | `GET/PUT /api/option/` | admin options（`HandlePrefix` 匹配尾斜杠，见 §4.4） | ✅ |
| 兑换码管理 | admin/RedemptionsPage | `GET/POST /api/redemption`、`DELETE /api/redemption/{code}` | billing redeem CRUD | ✅ |
| 支付订单管理 | admin/PaymentOrdersPage | `GET /api/payment/orders[/{tradeNo}]` | billing `ListPaymentOrders` | ✅ |
| 系统选项 | admin/OptionsPage | `GET/PUT /api/option/` | admin options | ✅ |
| 账目日志（管理） | admin/LogsPage | `GET /api/log` | admin `ListLedgerEntries`/`ListLogs` | ✅ |
| CSV 导出 | 多页面 | `/api/{user,log,channel,redemption}/export` | 对应 List RPC | ⚠️ 路由在，未逐一验证下载 |

---

## 2. 缺口清单 A：后端已就绪，前端缺入口（= P0 剩余工作面）

这些后端路由 + RPC 均已实现，但 `web/src` 中**没有任何页面调用**。这是把"OpenAI 兼容后端"变成"完整 One-API 产品"前端的剩余清单。

| 能力 | 后端现状 | 前端缺口 | 建议优先级 |
|---|---|---|---|
| **对账复核** | `GET /api/reconciliation`、`/api/reconciliation/{id}`（admin RPC `ListReconciliationRuns`/`GetReconciliationRun` 全实现，历史持久化） | 无 admin 对账页面 | **高**（有支付就需要对账可视化） |
| **OAuth 登录/绑定** | identity 全套：GitHub/Google/OIDC/Lark/WeChat/Telegram + `/api/oauth/state` + Turnstile | LoginPage 只有账号密码，无第三方登录按钮；ProfilePage 无绑定入口 | **高** |
| **找回密码 / 邮箱验证** | `GET /api/verification`、`/api/reset_password`、`POST /api/user/reset` | 无"忘记密码"页面 | 中 |
| **邀请/分销码** | `GET /api/user/aff`、`/api/user/invitation`（注册时发放邀请奖励已通） | 无邀请码展示页 | 中 |
| **内容管理** | `GET/PUT /api/{notice,about,home_page_content}` | 无 admin 内容编辑页 | 中 |
| **分组管理** | `GET/POST/PUT/DELETE /api/group`（含 `with_ratio`） | 无 admin 分组页（渠道页只是引用分组） | 中 |
| **渠道连通测试** | `GET /api/channel/test`、`/api/channel/test/{id}` | ChannelsPage 未接测试按钮 | 低 |
| **全量余额刷新** | `GET /api/channel/update_balance`（无 id，刷新全部） | 前端只调用了单渠道 `/{id}` 版本 | 低 |

> 注：还存在一组 `/v1/*` 形态的 admin CRUD（`/v1/users`、`/v1/channels`、`/v1/redeem-codes`、`/v1/system/options` 等），与 `/api/*` 版本功能重叠，前端统一走 `/api/*`。`/v1/*` 可视为内部/兼容别名，无需前端覆盖，但**重复维护两套需留意契约漂移**。

---

## 3. 缺口清单 B：故意留置的占位 / 未集成

| 项 | 现状 | 性质 |
|---|---|---|
| relay 30+ OpenAI 兼容端点 | `/v1/{files,batches,fine_tuning,assistants,threads,vector_stores,evals,...}` 稳定返回 501 NotImplemented | 故意占位，保持 OpenAI 形状的拒绝；非缺口 |
| `/v1/images/{edits,variations}`、`/v1/edits` | 501 占位 | 同上 |
| `/api/user/aff_transfer` | 200 disabled | 上游也未实现，故意停用 |
| 8 个原生 Provider 适配器 | `factory.go` 返回 "requires a native provider adapter" | **暂停**，卡在沙箱凭证/staging（见 gap-priority 文档） |
| **monitor-worker / notify-worker** | 各自 `/v1/health-checks`、`/v1/alert-rules`、`/v1/notifications` 端点存在 | **未接入 admin BFF，也无前端**——目前是孤立服务，无人调用 |

---

## 4. 可疑项的源码二次验证（避免误报）

三个看似"前端调了后端没有"的点，经验证均为**误报**——路由藏在前缀/分发处理器里，不是显式 `HandleFunc`：

1. **`PUT /api/user/self`**：`internal/identity/server/http.go:985` `handleSelf` 内 `switch r.Method` 显式处理 `MethodPut → uc.UpdateSelf(...)`；admin-api 侧 `:230` 用 `identityProxy.ServeHTTP` 透传所有方法。✅
2. **`POST /api/user/pay` 响应字段**：`http.go:860-861` 返回 `"trade_no"` + `"pay_url"`，与前端 `CreatePaymentResponse { trade_no?, pay_url? }` 完全一致。✅
3. **`POST /api/user/manage`**：`internal/admin/server/http.go:1104` 在前缀处理器内 `trimmed == "api/user/manage"` 分支，含 `promote`/`demote` 逻辑 → identity `SetUserRole`。✅
4. **`/api/option/`（尾斜杠）**：`http.go:286` 用 `HandlePrefix("/api/option", ...)`，前缀匹配涵盖尾斜杠。✅

---

## 5. 验收建议（把"能跑通"正式锁定）

当前仅有 `web/e2e/admin-smoke.spec.ts` 一条 Playwright 冒烟。建议把 §1 中标 ✅ 的主流程补为可重复验收：

1. **用户侧 e2e**：注册 → 登录 → 建 Token → 看仪表盘 → 兑换码充值 → 看订单/用量。
2. **管理侧 e2e**：登录 → 用户增删改/提权 → 渠道增删改+刷余额 → 价格/选项保存 → 兑换码生成。
3. **支付闭环（§1 标 ⚠️）**：用 `mock` provider 跑 下单 → 回调 `/api/.../alipay/notify` → 订单转 paid → ledger 入账 → 对账无差异。这是目前唯一"字段对得上但闭环未验证"的主流程，应优先补。

## 6. 下一步动作（按价值排序）

1. 补 **对账复核 admin 页面** + **OAuth 登录按钮**（后端最完整、产品价值最高的两块前端空白）。
2. 补 **支付闭环 e2e**（mock provider），锁定唯一未验证的资金主流程。
3. 决策 **monitor/notify** 是接入 admin BFF + 前端，还是明确标注为独立运维服务。
4. 决策 `/v1/*` 与 `/api/*` 双轨：合并或明确其中一套为内部别名，消除漂移面。
