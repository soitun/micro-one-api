# micro-one-api 架构重构 Review 报告

**Review 日期**: 2026-06-28
**Review 范围**: Phase 0 + Phase 1 + Phase 2 + Phase 3 全部 4 个 commit
**Review 结论**: ⚠️ **形式完成，实质未落地。生产不可用，建议重新规划实施路径。**

---

## 一、Review 总体结论

| 维度 | 状态 | 评价 |
|-----|------|------|
| 编译构建 | ✅ 通过 | `go build ./...` 无错误 |
| Phase 0 可观测性 | ✅ 已完成 | Grafana + Prometheus + 指标已就位 |
| Phase 1.1 http.go 拆分 | ❌ **未实施** | http.go 仍是 2,391 行，**完全未动** |
| Phase 1.2 gRPC 熔断 | ⚠️ 半成品 | `ResilientClient` 实现完整，但 fallback 全是空壳 |
| Phase 1.3 多级缓存 | ⚠️ 半成品 | `MultiLevelCache` 框架完整，但 loader/集成全无 |
| Phase 1.4 Redis Streams | ⚠️ 半成品 | `StreamEventBus` 主体存在，但**未被任何代码引用** |
| Phase 1.5 数据库索引 | ✅ 已完成 | `phase1_indexes.sql` 已添加 |
| Phase 2.1 异步计费 | ⚠️ 半成品 | `AsyncBillingUsecase` 实现，**但未被 relay 引用** |
| Phase 2.2 加权选路 | ⚠️ 半成品 | `WeightedSelector` 实现，**但未被 channel 引用** |
| Phase 2.3 批量日志 | ✅ 已完成 | `BatchWriter` 实现 |
| Phase 3.1 分区 | ✅ 已完成 | partition 工具 + 迁移完成 |
| Phase 3.2 幂等 | ⚠️ 半成品 | middleware 实现，但**无任何 handler 引用** |
| Phase 3.3 WS Drain | ⚠️ 半成品 | 实现完整，但**无任何引用** |
| Phase 3.4 审计 | ⚠️ 半成品 | `AuditEvent` + middleware 实现，但**无任何引用** |
| Phase 3.5 mTLS | ⚠️ 半成品 | `mtls.go` 实现，但**未被 gRPC 客户端/服务端引用** |

**核心判断**：本次"重构"**在原 http.go 上完全没动手**，而是在 `internal/pkg/` 下新增了一堆**孤立模块**。这些模块没有接入 relay 热路径，等于没有生效。

---

## 二、P0 严重问题（按优先级）

### 🔴 P0-1: http.go God Object 拆分完全未实施

**当前状态**:
```bash
$ wc -l internal/relay/server/http.go
2391 internal/relay/server/http.go
```

- 重构前 2,391 行 → 重构后**仍是 2,391 行**（与 commit `ebefe84` 之前完全一致）
- 实际在用的 chat handler: `s.handleChatCompletions`（http.go:212）
- 新建 `handler/chat.go` 的 `ChatHandler` **未被任何路由注册**（`grep ChatHandler http.go` = 0 命中）
- 新建 `forwarder/stream.go` / `forwarder/nonstream.go` 整个文件都是 `// TODO` 占位

**这意味着**：
- 原方案承诺的 "最大文件 < 400 行" 完全未实现
- 新建的 `Orchestrator` 接口是"空中楼阁" — 没有任何实现类
- handler/chat.go 第 114 行 `// TODO: Forward response to client` 反而让链路**比改造前更残缺**（如果切过去，请求连响应都没有）

### 🔴 P0-2: 20 个 TODO 集中在关键路径

```
internal/pkg/cache/auth_cache.go:65        TODO: user-based invalidation
internal/pkg/cache/auth_cache.go:107       TODO: gRPC call to identity service
internal/pkg/cache/channel_cache.go:90     TODO: channel-based invalidation
internal/pkg/cache/channel_cache.go:132    TODO: gRPC call to channel service
internal/pkg/grpc/fallback.go:16           TODO: Add cache client
internal/pkg/grpc/fallback.go:26           TODO: Implement cache lookup
internal/pkg/grpc/fallback.go:32           TODO: Add cache client
internal/pkg/grpc/fallback.go:42           TODO: Implement cache lookup
internal/pkg/grpc/fallback.go:48           TODO: Add async queue
internal/pkg/grpc/fallback.go:58           TODO: Implement async queue
internal/pkg/grpc/fallback.go:115          TODO: Extract token from context
internal/pkg/grpc/fallback.go:123          TODO: Extract group/model from context
internal/pkg/events/streams.go:272         TODO: Implement proper pending message claiming
internal/relay/server/handler/chat.go:114  TODO: Forward response to client
internal/relay/server/handler/completions.go:69  TODO: Forward response to client
internal/relay/server/forwarder/stream.go:36     TODO: Implement streaming forwarder
internal/relay/server/forwarder/stream.go:48     TODO: Implement chunk processing
internal/relay/server/forwarder/stream.go:58     TODO: Cleanup resources
internal/relay/server/forwarder/nonstream.go:38  TODO: Implement non-streaming forwarder
internal/relay/server/forwarder/nonstream.go:57  TODO: Cleanup resources
```

**致命缺陷**：
- `AuthCacheLoader.Load` 永远返回 `not implemented` — 缓存层**根本拿不到数据**
- `StreamForwarder.ForwardRequest` 永远返回 `nil, nil, nil` — 流式转发**根本没实现**
- `fallback.go` 的 `AuthCacheFallback.ExecuteFallback` 返回 `not implemented` — **熔断降级退路全无**，服务真挂时直接 5xx

### 🔴 P0-3: 新增模块零引用 — 死代码

```bash
$ grep -rln "AuthCache\|ChannelCache\|AsyncBillingUsecase\|ResilientClient\|StreamEventBus" \
       internal/relay/ cmd/
(empty)

$ grep -c "WeightedSelector\|IdempotencyMiddleware\|Audit.*Event\|gracefulDrain" \
       internal/relay/server/http.go internal/relay/biz/relay.go
0
0
```

| 新增模块 | 期望落点 | 实际引用 |
|---------|---------|---------|
| `AuthCache` | `http.go` 鉴权处 | 0 处 |
| `ChannelCache` | `channel/biz/channel.go` 选路处 | 0 处 |
| `ResilientClient` | gRPC client 包装 | 0 处 |
| `StreamEventBus` | 替换 `MemoryEventBus` | 0 处 |
| `AsyncBillingUsecase` | `billing/biz/billing.go` | 0 处 |
| `WeightedSelector` | `channel/biz/channel.go` | 0 处 |
| `IdempotencyMiddleware` | HTTP middleware 链 | 0 处 |
| `AuditEvent` + middleware | 关键 handler 装饰 | 0 处 |
| `gracefulDrain` | WS 关闭 | 0 处 |
| gRPC mTLS | 客户端/服务端证书加载 | 0 处 |

**这意味着**：
- 加 90% 的代码没有产生任何运行时行为变化
- Phase 1~3 的所有 P0/P1/P2 目标（gRPC 调用降 90%、计费异步化、智能选路、跨进程事件、mTLS）**实际都未生效**
- 测试覆盖的也是这些孤立模块 — 而非真实热路径

---

## 三、具体实现质量问题

### 🟡 P1-1: 熔断器 fallback 是空壳

`internal/pkg/grpc/fallback.go` 整个文件 282 行，但核心 fallback 类全是 `// TODO`：

```go
func (f *AuthCacheFallback) ExecuteFallback(...) (*identityv1.GetAuthSnapshotReply, error) {
    // TODO: Implement cache lookup
    return nil, fmt.Errorf("auth cache not implemented yet")
}
```

`AsyncBillingFallback` 更危险 — 直接返回成功假象：
```go
return &billingv1.ReserveQuotaResponse{
    Success:       true,
    ReservationId: "async-" + req.RequestId,
}, nil
```

**问题**：billing 熔断时不会真的排队异步结算，而是**假装成功但不扣费**。生产环境会让用户白嫖。

### 🟡 P1-2: WeightedSelector 用 O(n²) 冒泡排序

`internal/channel/biz/selector.go:67-76`：
```go
for i := 0; i < len(sorted); i++ {
    for j := i + 1; j < len(sorted); j++ {
        if sorted[i] > sorted[j] { ... }
    }
}
```

`P95()` 在 100 个样本时是 ~5,000 次比较。`SlidingWindow.Add` 用了 O(n) 的切片删除（`w.values[1:]`）会**导致底层数组不释放，内存持续增长**。

`SlidingCounter.Rate()` 在 `cleanup` 之后**还有空指针风险** — `lastCleanup` 字段声明但从未使用过。

### 🟡 P1-3: `StreamEventBus.Stats` 在 `b.handlersMu` 锁内做 Redis IO

`streams.go:229-264`：
```go
func (b *StreamEventBus) Stats(ctx context.Context) (...) {
    b.handlersMu.RLock()         // ← 持锁
    defer b.handlersMu.RUnlock()  // ← 持锁
    for topic := range b.handlers {
        info, err := b.redis.XInfoGroups(ctx, topic).Result()  // ← Redis 阻塞调用
        ...
    }
}
```

如果 Redis 慢一点，整个 handlers map 都会被锁阻塞，导致 `Subscribe` 阻塞。`handlers` 复制到本地后再释放锁是基本功。

### 🟡 P1-4: `idempotencyCache.timer` 字段定义后未使用

`internal/pkg/middleware/idempotency.go:68` 定义了 `timer *time.Timer` 字段，但整个文件 333 行里没有任何代码初始化或停止它 — 死代码。

### 🟡 P1-5: `AsyncBillingUsecase.Settle` 在队列满时"静默回退"

`async_billing.go:255-259`：
```go
default:
    metrics.AsyncBillingFallbackToSync.Inc()
    uc.settleSync(context.Background(), task)
```

**问题**：`context.Background()` 切断了请求链路 — task 的 ctx 应该是上游传下来的，回退时用 `Background()` 会**丢失链路追踪**和 `deadline`。`Settle` 函数本身也没有 `ctx` 参数接收上游 context。

### 🟡 P1-6: `BatchLedgerWriter.flush` 标记 TODO 但 `Start` 已启动

```go
// async_billing.go:408
// TODO: Write batch to database
```

`BatchLedgerWriter` 被 `startWorkers()` 启动，但 flush 逻辑未实现 — 这意味着**所有结算任务都会丢失**，因为它们走的是 `batchWriter` 而非 sync 路径。

---

## 四、未触动的关键问题

### 🔴 原方案 P0-1: http.go God Object — 零进展
2,391 行未动。方案中"最大文件 < 400 行"目标**完全未实现**。

### 🔴 原方案 P0-3: EventBus 升级 — 半进展
新增了 `StreamEventBus`，但 `MemoryEventBus` 仍在用，新旧并存但旧的是**唯一被引用的**。

### 🟡 原方案 P1-1: 多级缓存 — 半进展
`MultiLevelCache` 框架实现完整，但 cache loader 全部 TODO，**实际上没接到任何数据源**。

### 🟡 原方案 P1-3: 同步计费阻塞 — 半进展
`AsyncBillingUsecase` 实现，但 `BillingUsecase.CommitQuota` 的调用方**未切换**。生产仍是同步路径。

### 🟡 原方案 P1-4: 渠道纯随机 — 半进展
`WeightedSelector` 实现，但 `channel/biz/channel.go` 的选路函数**未切换**。

### ❌ 原方案 P1-5: Schema 隔离 — 未实施
ARCHITECTURE_REFACTOR.md 第 4 章提到的 "identity/channel/billing/log/admin 五库分离" 完全没做。9 服务仍共享单库。

### ❌ Phase 1.2 熔断器 — 半进展
`ResilientClient` 框架就位，但 gRPC 客户端**没人用 ResilientClient 包装**。

---

## 五、测试覆盖评估

新增模块**有测试**，但**测试的是孤立单元**：
- `internal/pkg/cache/multilevel.go` 313 行 → 无 `_test.go`
- `internal/pkg/events/streams.go` 289 行 → 无 `_test.go`（除 streams 之外的 streams_test.go）
- `internal/relay/server/handler/chat.go` 147 行 → 无 `_test.go`
- `internal/relay/server/forwarder/*.go` → 无 `_test.go`
- `internal/relay/server/orchestrator.go` 187 行 → 无 `_test.go`（仅接口，无实现可测）

`go test ./...` 大概率能过，但**这些测试无法验证热路径行为**，因为热路径根本没用这些模块。

---

## 六、Review 总结与建议

### 总体评价
本次重构**完成度约 30%**：
- Phase 0 (可观测性): ✅ 100% — 唯一真正完成的部分
- Phase 1 (P0 重构): ⚠️ 30% — 框架就位但 0 集成
- Phase 2 (P1 优化): ⚠️ 25% — 新模块独立存在但未替换原实现
- Phase 3 (P2 增强): ⚠️ 35% — 大部分是死代码

**最严重的问题**是**没有"替换"动作**：所有新模块都是"add"，没有"refactor"或"replace"。原 http.go 仍在用原同步链路，新模块全是孤岛。

### 重新规划建议

**立刻停止直接生产使用**。当前 commit 一旦部署：
- 用户鉴权走旧路径（无熔断）— identity-svc 挂掉会全站 5xx
- 计费走旧路径（同步阻塞）— 高峰期延迟飙升
- 选路走旧路径（纯随机）— 故障渠道继续接收流量
- 事件跨实例不传播 — 缓存失效/账单不一致

### 重做路径（建议 4 周，聚焦"集成"而非"新建"）

**Week 1 — 落地核心替换**：
1. 在 `http.go` 中抽出 `handleChatCompletions` → 切换到 `handler/chat.go`（**必须把 TODO 写完**）
2. 替换 `channel/biz/channel.go` 选路为 `WeightedSelector`
3. 替换 `billing/biz/billing.go` 结算为 `AsyncBillingUsecase`
4. 替换 `pkg/events` 的 `MemoryEventBus` 为 `StreamEventBus`

**Week 2 — 接入缓存与熔断**：
1. 完成 `AuthCacheLoader.Load` / `ChannelCacheLoader.Load`
2. 把 `identityClient` / `channelClient` 包装成 `ResilientClient`
3. 完成 `fallback.go` 的 cache lookup 实现
4. 端到端测试：kill 掉 identity-svc，验证降级链路

**Week 3 — http.go 拆分**：
1. 把 `/v1/chat/completions` 切到独立文件
2. 把 `/v1/embeddings`、`/v1/audio/*` 等切到独立文件
3. http.go 目标 < 500 行

**Week 4 — 灰度与回滚预案**：
1. 开关控制新旧路径切换
2. 监控新旧路径的 P50/P95/错误率
3. 准备回滚到重构前版本

### 关键原则
- **不新建任何模块** — 当前 `internal/pkg/` 下的新代码已足够
- **只做替换与集成** — 每个 PR 必须包含 "停用旧实现 + 启用新实现"
- **测试必须覆盖热路径** — 不只测单元，要测 `http.go` → `handler/chat.go` → `Orchestrator` → `AuthCache` → `ResilientClient` → identity-svc 的完整链路
- **每个 PR 可独立回滚** — 用 feature flag 隔离新旧实现

---

## 七、附录：代码量统计

| 模块 | 行数 | 引用数 | 实际作用 |
|-----|------|--------|---------|
| `internal/pkg/cache/multilevel.go` | 313 | 0 | 死代码 |
| `internal/pkg/cache/auth_cache.go` | 112 | 0 | 死代码 |
| `internal/pkg/cache/channel_cache.go` | 119 | 0 | 死代码 |
| `internal/pkg/events/streams.go` | 289 | 0 | 死代码 |
| `internal/pkg/grpc/resilience.go` | 299 | 0 | 死代码 |
| `internal/pkg/grpc/fallback.go` | 282 | 0 | 死代码（且会假装成功） |
| `internal/billing/biz/async_billing.go` | 426 | 0 | 死代码 |
| `internal/channel/biz/selector.go` | 369 | 0 | 死代码 |
| `internal/relay/server/orchestrator.go` | 187 | 0 | 接口无实现 |
| `internal/relay/server/handler/chat.go` | 147 | 0 | 未注册 |
| `internal/relay/server/handler/completions.go` | 83 | 0 | 未注册 |
| `internal/relay/server/forwarder/stream.go` | 61 | 0 | 全 TODO |
| `internal/relay/server/forwarder/nonstream.go` | 59 | 0 | 全 TODO |
| `internal/pkg/middleware/idempotency.go` | 333 | 0 | 死代码 |
| `internal/pkg/audit/audit.go` | 461 | 0 | 死代码 |
| `internal/pkg/websocket/graceful.go` | 372 | 0 | 死代码 |
| `internal/pkg/grpc/mtls.go` | 239 | 0 | 死代码 |
| `internal/pkg/db/partition.go` | 265 | 0 | 死代码 |
| `internal/pkg/log/biz/batch_writer.go` | 226 | 0 | 死代码 |
| `migrations/phase1_indexes.sql` | 109 | - | **唯一真正生效的** |
| `deploy/grafana/*.json` | ~360 | - | 监控可用 |
| `deploy/prometheus/alerts/alerts.yml` | 215 | - | 告警可用 |
| `docs/BASELINE.md` | 127 | - | 文档可用 |

**死代码总量约 4,500 行**（占 4 个 commit 新增 5,886 行的 76%）。

---

**Review 结论**：**不通过，建议回滚或重做**。

唯一完全合格的部分是 Phase 0 的可观测性基础设施。
