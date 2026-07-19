# Micro-One-API v0.9.0 发布公告

> 2026-07-19 · 上一版：[v0.8.2](./release-v0.8.2.md)（2026-07-17）

v0.9.0 是 v0.8.2 之后的 **MINOR** 版本，落地架构重构路线图（[ARCHITECTURE_REFACTOR.md](../design/ARCHITECTURE_REFACTOR.md)）的 Phase 2.1 / 2.2 / 2.3 / 2.4 / 2.5 与 Phase 3.3 共 6 项。本版把 relay-gateway 提交链路上的两个 DB 热点（billing commit、log insert）改成异步批量、补齐 WebSocket 优雅下线、并通过 per-service schema 隔离 + 配置热更新为水平扩展铺路。

所有新增能力**默认关闭或保持旧行为**，升级即生效、无需改环境变量、无需迁移；本版**没有 API 破坏性变更**，**没有新增编号业务表迁移**。schema 隔离与异步 billing / log 批量都是 opt-in，按需开启。

## 亮点

- **异步 billing 落地（Phase 2.1）**：`billing-service` 的 `CommitQuota` 不再阻塞 relay 主路径。当 `billing.async.enabled=true` 时，结算入队后台 worker 执行权威的 `CommitQuotaWithUsageAndSplit` 流水线（预留态机 + 钱包扣减 + 账本 + 订阅用量），relay 立即返回带 `async_enqueued=true` 的临时结果。关掉时退回同步路径，零行为变更。
- **批量日志写入（Phase 2.3）**：`log-service` 的 `IngestLog` 走 `BatchLogWriter` 批量 flush（默认关闭，`log.batch_enabled=true` 开启），替代每条 `INSERT` 一次的同步写入。队列满 / writer 关闭时自动降级为 `repo.Create`，保证不丢日志。
- **加权选择器端到端验证（Phase 2.2）**：新增 `GET /api/v1/admin/channels/selector/stats`（ADMIN_TOKEN 常量时间守卫）暴露 `WeightedSelector` 运行态快照；集成测试 `TestRelaySelectChannel_WeightedDistribution` 端到端验证高权重严格多选中、`CurrentWeight` 被修改、p95 延迟和熔断器状态闭环。
- **WebSocket 优雅排空（Phase 3.3）**：relay-gateway 在 SIGTERM 时先返回 503 + Retry-After 拒绝新升级、等待既有连接 drain（默认 30s）再强制关闭。`/healthz` 在 draining 期间返回 503 让负载均衡器摘流，配合 kratos `BeforeStop` / `StopTimeout` 完成 graceful shutdown。
- **per-service DB schema 隔离（Phase 2.4，opt-in）**：`xdb.DatabaseConfig` 新增 `Schema` 字段。MySQL 通过 DSN DBName 重写、Postgres 通过 `options=-c search_path=<schema>` 注入，SQLite 忽略（=换文件）。每个服务通过 `<SVC>_SCHEMA`（或全局 `DATABASE_SCHEMA`）启用，默认空保持单库行为。`cmd/migrate` 新增 `-ownership` 按 `migrations/ownership.yaml` 在 per-service schema 上只跑该服务拥有的迁移。
- **配置热更新（Phase 2.5）**：`platform/config/source.go` 用 fsnotify 替换原来的 noop watcher，debounced 100ms 重载、原子保存编辑器（Rename/Remove）自动 re-add、fan-out 到所有订阅者、fsnotify 不可用时优雅降级 noop。`ModelMapper` 持有原子快照，`configs/models.yaml` 变化时热替换，解析失败保留旧快照并告警。

## 变更内容

### Added

#### 异步 billing —— `feat(billing): wire async billing into commit hot path (Phase 2.1)`（`9911671`）

- `biz.AsyncBillingUsecase.SettleTask` 扩展 `ReservationID/Success/Usage`，worker 携带完整 commit 输入。
- `processSettlement` / `settleSync` 跑权威的 `CommitQuotaWithUsageAndSplit` 流水线，不再绕过预留态机直接写账本。新增共享 `runCommitPipeline`。
- `BillingService` 新增 `asyncUc` 字段 + `SetAsyncBillingUsecase` setter：`CommitQuota` 在 `asyncUc != nil && req.Success` 时入队并返回临时成功，`Success=false`（release）保持同步。
- wire（重新生成）：`svc.SetAsyncBillingUsecase(asyncBilling)`，仅在 `cfg.Billing.Async.Enabled` 时构造。
- 测试：`SettleEnqueuesTask` / `SettleFallsBackWhenQueueFull` / `SettleNilTaskIsSafe`。

#### 批量日志写入 —— `feat(log): wire batch log writer into LogUsecase (Phase 2.3)`（`feb7d79`）

- `biz/log.go` 新增可选 `*BatchLogWriter` 字段 + `SetBatchWriter` setter；`IngestLog` 路由到队列，nil 时降级 `repo.Create`。新增 `LogRepoBatch` 能力接口（`CreateBatch(ctx, []*LogEntry)`）。
- `biz/batch_writer.go`：`createBatch` 接收 ctx 调用 ctx-aware `CreateBatch`。
- `data/data.go`：`Repository.CreateBatch` 走 gorm `CreateInBatches`，回填生成 ID；内存模式加锁 append。
- `conf/config.go`：新增 `BatchEnabled`（默认 false）+ `BatchFlushInterval`（默认 1s）。
- wire（重新生成）：在 `newApp` 内按 `cfg.Log.BatchEnabled` 构造 / 启动 / 停止 `BatchLogWriter`，通过 `uc.SetBatchWriter` 注入。
- 测试：`IngestLogSyncFallback` / `IngestLogBatchRouting` / `IngestLogNilBatchWriterIsSafe` / `BatchLogWriter_QueueFullReturnsError`。

#### 加权选择器端到端验证 —— `feat(channel): verify weighted selector end-to-end (Phase 2.2)`（`cdcd165`）

- `ChannelUsecase.SelectorStats()` 暴露选择器运行态：每渠道权重、`CurrentWeight`、inflight、p95 延迟、错误率、熔断器开关。
- `testutil` 再导出 `ChannelStats` + `SelectorStats`，供跨服务集成测试观察选择器状态。
- channel-service HTTP 新增 `GET /api/v1/admin/channels/selector/stats`（ADMIN_TOKEN 常量时间比较，未设置时 fail-closed）。
- `internal/integration/weighted_distribution_test.go::TestRelaySelectChannel_WeightedDistribution` 端到端验证高权重严格多选中、`CurrentWeight` 被修改、inflight 归零、p95 非零（证明 `RecordChannelHealth -> WeightedSelector.RecordHealth` 反馈环已闭环）。

#### WebSocket 优雅排空 —— `feat(relay): wire graceful WebSocket drain into openai_ws_* (Phase 3.3)`（`3acff1f`）

- `HTTPServer` 新增 `wsConnTracker` / `wsDrainCfg` 字段 + nil-safe `SetOpenAIWSConnPool`（幂等）、`SetOpenAIWSDrainConfig`、`drainConfig`、`IsWSDraining`、`DrainWSConnections(ctx)`。tracker 未注入时行为同前。
- `handleResponsesWebSocket` 在 draining 时拒绝新升级（503 + Retry-After），并将每个接受的客户端连接注册到 tracker（close 函数委托给 `wsConn.CloseNow`）；defer `Unregister` 保证取消的 relay 离开活跃集。
- `/healthz` 在 draining 时返回 503 `{"status":"draining","drain":"true"}` + Retry-After 让 LB 摘流；稳态仍 200 ok。
- `OpenAIWSConfig` 新增 `DrainTimeout`（默认 30s）+ getter；`configs/config.yaml` 暴露 `openai_ws.drain_timeout`（`OPENAI_WS_DRAIN_TIMEOUT`）。
- wire / wire_gen（重新生成）调用 `SetOpenAIWSDrainConfig`，注入 `kratos.StopTimeout(drainTimeout+10s)` + `kratos.BeforeStop(httpServer.DrainWSConnections)`，SIGTERM 触发 drain → 503 → 等待 → 强制关闭。
- `relay_helpers.go` 新增 `appwsDrainConfig`，按 drain 窗口尺寸化 `CloseTimeout` / `NotifyBeforeClose` / `MaxConcurrentClose`。
- `platform/websocket` 新增 `SetDrainingForTest` 在隔离环境翻转 CAS 标志（仅测试用，文档说明禁止在生产代码调用）。

#### per-service DB schema 隔离 —— `feat(phase2): add per-service DB schema isolation and config hot reload (Phase 2.4 / 2.5)`（`3571d5d`，Phase 2.4 部分）

- `xdb.DatabaseConfig` 新增 `Schema` 字段：MySQL 通过 go-sql-driver `ParseDSN/FormatDSN` 重写 DSN DBName（保留 auth/TLS/parseTime/charset），Postgres 在 URL-form 和 key=value DSN 里注入 `options=-c search_path=<schema>`，SQLite 忽略。
- Rewriter 导出为 `xdb.RewriteMySQLDBName` / `RewritePostgresSearchPath`，供持有 `*sql.DB` 的调用方（admin-api 的 system_options repo）应用同一 schema。
- 每个服务的 `internal/conf.DatabaseConfig` 新增 `Schema` 字段，每个 `configs/config.yaml` 暴露 `${<SVC>_SCHEMA:-}`，默认空保留旧行为。
- `data.NewRepositoryFromEnv` / `NewData` 按 wire 参数 > `<SVC>_SCHEMA` > `DATABASE_SCHEMA` 优先级解析 schema。
- `wire.go` / `wire_gen.go` 把 `cfg.Data.Database.Schema` 串到第三参数；`admin_helpers` 新增 `resolveAdminSchemaDSN`。
- `platform/database/migrate.Runner` 新增 `WithOwnershipFilter(service)`，由 `migrations/ownership.yaml`（shared + per-service 列表）支持；`cmd/migrate` 暴露 `-ownership`。manifest 缺失或未知服务时退化为「应用全部」，未启用 schema 隔离的部署完全不受影响。
- 新增 `migrations/schema_split.sql`：参考 DDL，把共享 `oneapi` 库的表 **复制**（非移动）到 8 个 per-service schema。
- `docs/deployment.md §10` 记录 opt-in、切流步骤、billing 对账的跨 schema 读过渡方案。

#### 配置热更新 —— `feat(phase2): ... (Phase 2.5 部分)`（`3571d5d`）

- `platform/config/source.go` 用 fsnotify 替换 noop watcher：debounced（100ms）重载、Rename/Remove re-add 以抗原子保存编辑器、fan-out 到所有订阅者、fsnotify 不可用时降级 noop watcher（服务仍可启动）。
- `platform/config/subscribe.go` 新增 `SubscribeFile(path, cb)` 返回 stop func。
- `internal/biz.ModelMapper` 改持原子快照 + `Reload()`；`Resolve/HasCapability/GetEntry` 无锁读。
- `cmd/relay-gateway` 通过 `xconfig.SubscribeFile` 订阅 `models.yaml`，变化时调 `ModelMapper.Reload()`；解析失败保留旧快照并告警；stop 接入 app cleanup。
- fsnotify 从 indirect 提升为 direct 依赖。
- `docs/deployment.md §11` 记录机制、已接入热更新点、已知限制。

### Changed

- `api/billing/v1/billing.proto` 的 `CommitQuotaResponse` 新增 `async_enqueued` 字段（field 7），向后兼容。relay-gateway 在 async 入队时跳过临时金额会计，但仍记录 channel token usage。
- WebSocket drain 503 响应的 `Retry-After` 改为根据配置的 drain timeout 动态计算。
- `xdb.ResolveSchema` 集中 schema 解析逻辑，应用到全部 9 个服务的 data 包和 subscription data。
- channel-service 把 `ADMIN_TOKEN` 通过 `sync.Once` 缓存（原来是每次请求 `os.Getenv` 系统调用）；测试暴露 `resetAdminTokenCache`。端点保持常量时间比较 + fail-closed。
- `internal/server/http.go` 用 `sync.OnceValue` 缓存默认 drain timeout，避免 `/healthz` 热路径每次探针分配 `DrainConfig`。
- `internal/server/openai_ws_forwarder.go` 在 `wsConnTracker.NewConnection` 返回 nil 时告警（之前静默忽略，导致连接对 drain 不可见）。
- `async_billing.go` / `batch_writer.go` 把 `fmt.Printf` 替换为 `applogger.Log.Warn`，让结算 / 丢日志错误走结构化日志。

### Fixed

#### `fix(review): address Phase 2/3 review findings (security, drain, data-loss)`（`7a81294`）

- **安全**：`xdb.withPostgresSearchPath` / `withMySQLDBName` 在拼进 DSN 前按 `^[A-Za-z_][A-Za-z0-9_]{0,62}$`（Postgres NAMEDATALEN-1）校验 schema 标识符。攻击者设置 `DATABASE_SCHEMA='public -c statement_timeout=1'` 之前可经 `options=-c search_path` 注入任意 Postgres runtime 参数。`withPostgresSearchPath` 改返回 `(string, error)`，`openPostgres` / `RewritePostgresSearchPath` / `resolveAdminSchemaDSN` 传播错误。
- **drain / shutdown 正确性**：
  - `AsyncBillingUsecase.Close()` 在取消 worker 前设置 `closed`（atomic.Bool），让并发 `Settle` 调用者回退同步 commit 路径，而不是入队到无消费者的队列；`Close` 在 `wg.Wait()` 后内联 drain `settleQueue`，`settlementWorker` 退出时也 drain，shutdown 期间不丢结算。
  - `BatchLogWriter.Stop()` 幂等化（`sync.Once`）并 drain 队列——processor 看到 stopChan 时仍在队列的条目被移入最终 flush 而非丢弃；`IngestLog` 在已关闭 writer 上回退 `repo.Create`；`dropped` 计数器改为 `atomic.Int64`（原来是有竞争的 int64）。
  - `BatchLedgerWriter.Stop` 用 `stopOnce` 守护，防止 `AsyncBillingUsecase.Close` 的二次 Close 在 close-of-closed channel 上 panic。
  - `config.EnvFileSource` 新增 `Close()` 终止广播 goroutine，移除 `broadcastLoop` 中失效的 `pending` 变量，所有退出路径都停 debounce timer；`SubscribeFile` 调用 `source.Close()`。
- **可观测 / 卫生**：billing-service 在 async 分支无 `reservation_id` 时 emit `AsyncBillingMissingReservationID`，让破坏契约的调用方在监控中可见。
- **测试**：xdb schema 校验拒绝 whitespace/quote/semicolon/dollar/length/digit-leading；`SettleAfterCloseFallsBackToSync` / `CloseDrainsQueuedTasks` 覆盖新 shutdown drain 契约；`log_test` 的 mock repo 加 `sync.Mutex` 避免 `-race` 下的 background flush 竞争。
- **格式**：gofmt 16 个被最近 8 个 commit 触碰但未 gofmt-clean 的文件。

#### `fix: address review issues from recent Phase 2/3 commits`（`eb6a92e`）

- `CommitQuotaResponse` 新增 `async_enqueued` 标志，让调用方区分临时异步结算与同步零成本结果。`SettleTask.RequestID` 填 reservation id 便于追踪。
- relay-gateway 在 async 入队时跳过临时金额会计，但仍然记录 channel token usage。
- WebSocket drain 503 响应的 `Retry-After` 改为根据配置的 drain timeout 动态计算。
- 新增 `xdb.ResolveSchema` 集中 schema 解析，应用到全部 9 个服务 data 包和 subscription data；文档说明 Postgres search_path options 替换行为。

#### `fix(security): correct gitleaks ignore fingerprint path for renamed doc`（`b88ae3d`）

- `subscription-account-setup-guide.md` 在 `065668a` 移到 `docs/runbooks/`，但 gitleaks 指纹用的是引入 finding 那次 commit（`7e0af96`）的路径 `docs/subscription-account-setup-guide.md`。`.gitleaksignore` 原本是重命名后路径导致无法匹配，Security Pipeline 在 `curl-auth-header` 第 288 行失败。对齐 gitleaks 实际报告的路径让 secret-scan job 重新通过（本地 gitleaks 8.x 验证 `no leaks found`）。

## 数据库迁移

本版**没有新增编号业务表迁移**，迁移执行方式与 v0.8.2 一致（一次性 `migrate` 服务按顺序执行自动迁移，成功后才启动应用）。

新增的 `migrations/schema_split.sql` 与 `migrations/ownership.yaml` 是 Phase 2.4 schema 隔离的**可选运维工件**：只有需要把服务切到独立 schema 的部署才需要执行，未启用 schema 隔离的部署**完全不需要**碰这两个文件。切流步骤和风险登记见 `docs/deployment.md §10` 与 `docs/TODO.md` 的 Phase 2.4 风险登记（R1–R6 + 验收标准）。

## 破坏性变更

API、数据库 schema 和默认部署行为**均无破坏性变更**。本版所有新能力默认关闭或保持旧行为，升级即生效，无需修改环境变量或配置。

需要注意的非破坏性变更：

- **`billing.proto` 新增字段**：`CommitQuotaResponse.async_enqueued`（field 7）。老的客户端会忽略未知字段；只有当你自己解析 `committed_amount` 并依赖其准确性时，需要检查 `async_enqueued=true` 时跳过临时金额（参考 relay-gateway 已实现的跳过逻辑）。
- **新可选环境变量**（不设置时全部保持旧行为）：
  - `BILLING_ASYNC_ENABLED`（默认 false）：开启异步 billing。
  - `LOG_BATCH_ENABLED`（默认 false）：开启批量日志写入。
  - `<SVC>_SCHEMA` / `DATABASE_SCHEMA`（默认空）：启用 per-service schema 隔离。
  - `OPENAI_WS_DRAIN_TIMEOUT`（默认 30s）：WebSocket drain 窗口。
- **Phase 2.4 schema 隔离启用前必须先完成 `docs/TODO.md` 的 R1–R6 前置修复**（billing 对账跨 schema 读、subscription_plans 漏复制、relay-gateway `SUBSCRIPTION_SCHEMA` 对齐等）。不启用 schema 隔离则完全无影响。
- **`xdb.withPostgresSearchPath` 函数签名**从 `string` 改为 `(string, error)`——这是 `platform/database/xdb` 的内部函数，仓库外不应依赖。

## 升级步骤

```bash
git fetch --tags
git checkout v0.9.0

# 检查并替换 deployments/docker-compose/.env 中的生产密钥
cd deployments/docker-compose
docker compose --env-file .env config --quiet

# 旧数据卷升级前先备份；全新环境直接启动
docker compose --env-file .env up -d --build
```

可选启用新能力（按需，不启用则保持旧行为）：

```bash
# 异步 billing（billing-service）
BILLING_ASYNC_ENABLED=true

# 批量日志写入（log-service）
LOG_BATCH_ENABLED=true

# per-service schema 隔离（按服务逐个切换，启用前务必看 docs/TODO.md 的 R1–R6）
# BILLING_SCHEMA=oneapi_billing
# LOG_SCHEMA=oneapi_log
```

Kubernetes 部署应先运行迁移，再执行 `kubectl apply -k deployments/k8s`，并等待九个 Deployment rollout 完成。完整步骤和 Secret 清单见 [docs/deployment.md](../deployment.md)。

## 验证

发布前已执行：

```bash
go build ./...                       # 通过
go vet ./...                         # 通过
./scripts/check-architecture.sh      # 通过（exit 0）
```

各 feat/fix 提交内附带的针对性测试（xdb schema 校验、async billing shutdown drain、batch log writer、weighted distribution 集成测试、WebSocket drain 503 flip、config hot reload 等）均已在各自 commit 中验证通过。

## 完整变更日志

- 9911671 feat(billing): wire async billing into commit hot path (Phase 2.1)
- feb7d79 feat(log): wire batch log writer into LogUsecase (Phase 2.3)
- 17aff20 docs(todo): mark Phase 2.1 and 2.3 complete, promote 3.3
- 3acff1f feat(relay): wire graceful WebSocket drain into openai_ws_* (Phase 3.3)
- cdcd165 feat(channel): verify weighted selector end-to-end (Phase 2.2)
- 3571d5d feat(phase2): add per-service DB schema isolation and config hot reload (Phase 2.4 / 2.5)
- b88ae3d fix(security): correct gitleaks ignore fingerprint path for renamed doc
- eb6a92e fix: address review issues from recent Phase 2/3 commits
- 7a81294 fix(review): address Phase 2/3 review findings (security, drain, data-loss)
- 8cd6b57 docs(todo): register Phase 2.4 schema isolation prod-enablement risks and plan
