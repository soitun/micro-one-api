# 功能缺失分析 — Kratos 落地方案 vs 当前实现

> 基于 `docs/One-API基于Kratos的微服务落地方案.md` 与当前代码库的全面对比，分析日期：2026-05-02

## 一、文档标记"✅ 已完成"但实际未达标的项

### 1. 健康检查（§16.3 / §16.5）

- **文档声称**：「所有服务均有 `/healthz` 端点」
- **实际状态**：4 个服务缺失健康端点
  - `billing-service` — HTTP server 注册了零路由
  - `identity-service` — HTTP server 是空壳 `func NewHTTPServer() {}`
  - `admin-api` — HTTP 和 gRPC server 都是空壳
  - `channel-service` — HTTP server 是空壳
- 另外命名不统一：relay-gateway 用 `/health`，其他用 `/healthz`

### 2. Redis 支持（§16.6 / §11）

- **文档声称**：「所有二期服务均支持 MySQL + Redis」、「Redis 独立封装到 `data/redis.go`」
- **实际状态**：`go.mod` 中**无 Redis 客户端库**，7 个服务的 `RedisConfig` 全是死代码，billing 的 `Data` 结构体中 `redis` 字段显式设为 `nil`

### 3. `internal/pkg/` 共享包完整性（§16.1）

- **文档声称**：「xtrace / xgrpc / xhttp 已创建」
- **实际状态**：这三个目录**只有 README.md 占位文件**，零 Go 代码

### 4. relay-gateway RelayService proto（§16.2.5）

- **文档标记**：⚠️ 空壳
- **确认**：`api/relay/v1/relay.proto` 中 `RelayService` 确实未被 relay-gateway 使用（HTTP 模式），但这个标记本身是准确的

---

## 二、文档§17-18 遗漏汇总中标记"✅"但实际有问题的项

### 5. 健康检查完善（§18 第 5 项）

- **文档声称**：✅ 已完成 — 所有 9 个服务均有健康检查端点
- **实际**：如上所述，4 个服务缺失

### 6. 真正接入 wire（§17 第 4 项）

- **文档声称**：✅ 已完成
- **实际**：wire.go + wire_gen.go 确实存在且可编译，此项准确

---

## 三、文档中建议但完全未实现的功能

### 7. 链路追踪 / xtrace（§17 架构改进建议第 2 条）

- **文档建议**：「补充 Jaeger/Zipkin 集成」
- **实际状态**：`xtrace/` 目录为空，无 OpenTelemetry 依赖，无分布式追踪能力

### 8. 二期服务 Proto 定义（§18 第 11 项 / §17 第 5 条）

- **文档建议**：「为 config / log / monitor / notify 服务补充 proto 定义」
- **实际状态**：这 4 个二期服务**仅有 HTTP API**，无 gRPC proto 定义，服务间无法通过 gRPC 调用

### 9. 统一错误处理增强（§17 第 1 条）

- **文档建议**：「进一步完善错误码体系」
- **实际状态**：`internal/pkg/errors/errors.go` 有 14 个 reason code，但二期服务（config/log/monitor/notify）的错误未纳入统一体系

### 10. 服务治理 / 注册中心（§17 第 3 条）

- **文档建议**：「接入 consul/nacos 等注册中心」
- **实际状态**：完全静态端点配置，无服务发现机制

### 11. 性能优化（§17 第 9 条）

- **文档建议**：「缓存策略、连接池配置、限流细化」
- **实际状态**：限流器是进程内 sync.Map，无分布式能力；无连接池配置；无 Redis 缓存层

### 12. 监控告警（§17 第 10 条）

- **文档建议**：「Prometheus metrics、Grafana dashboard」
- **实际状态**：无 Prometheus 指标暴露，无 metrics 端点

---

## 四、文档完全未提及但设计方案要求的缺失项

| 缺失项 | 来源 | 说明 |
|--------|------|------|
| **幂等键机制** | Kratos 方案§6（billing 设计） | reservation 的 request_id 无去重，重复请求会产生重复计费 |
| **密码哈希** | Kratos 方案§3.2（identity 职责） | Login() 接受任意非空密码，无 bcrypt |
| **OAuth/SSO** | Kratos 方案§3.2（identity 职责） | 无 OAuth2/OIDC 实现 |
| **事件总线** | Kratos 方案§10（配置变更通知） | events.go 只有 6 个未使用的 topic 常量，无消息队列 |
| **Config/Channel 变更通知** | Kratos 方案§10 | 直接写 DB 无副作用，无 pub/sub |
| **账务对账任务** | Kratos 方案§6（billing 设计） | 无 reconciliation、无审计报告 |
| **分组倍率外部化** | Kratos 方案§10（动态业务配置） | GroupRatio() 硬编码，config 字段未接入 |
| **分布式限流** | Kratos 方案§9（中间件设计） | 进程内限流器，多实例无法协同 |

---

## 五、文档§16 进度清单准确度评估

| 章节 | 准确度 | 问题 |
|------|--------|------|
| §16.1 骨架层 | **准确** | 目录结构、proto、configs 确实完整 |
| §16.2 服务实现层 | **基本准确** | RPC 方法确实已实现，但实现深度有限（如无密码哈希） |
| §16.3 基础设施层 | **部分失实** | 健康检查、Redis 支持标记为完成但实际未达标 |
| §16.4 配置一致性 | **准确** | 端口配置确实一致 |
| §16.5 部署配置 | **基本准确** | docker-compose 和 K8s 确实存在，但服务实际不用 Redis |
| §16.6 第二阶段服务 | **部分失实** | HTTP API 存在，但「支持 Redis」不实，且无 gRPC proto |

---

## 六、建议修正进度清单

以下 §16 中标记为 ✅ 的项目应降级为 ⚠️ 或 ❌：

| 原清单项 | 原标记 | 建议修正 | 原因 |
|----------|--------|----------|------|
| §16.3 健康检查 | ✅ | ⚠️ 部分完成 | 4 个服务缺失，命名不统一 |
| §16.3 Redis 支持 | ✅ | ❌ 未实现 | go.mod 无 Redis 依赖，config 是死代码 |
| §16.1 xtrace/xgrpc/xhttp | ✅ | ⚠️ 仅占位 | 只有 README，零代码 |
| §16.6 二期服务 Redis | ✅ | ❌ 未实现 | 同上 |
| §18 第 5 项 健康检查 | ✅ | ⚠️ 部分完成 | 4 个服务缺失 |
| §17 第 10 项 监控告警 | P2 待做 | 确认待做 | 无 Prometheus/metrics |

---

## 七、总结

当前实现与 Kratos 落地方案的**骨架层吻合度很高**（目录结构、DDD 分层、proto 定义、服务划分都按方案执行），但在**深度实现层面**有明显差距：

1. **Redis 完全未接入**（文档多次声称已完成）
2. **4 个服务健康检查缺失**（文档声称全部完成）
3. **xtrace/xgrpc/xhttp 是空壳**（文档列入共享包清单）
4. **二期服务无 gRPC proto**（文档已列为建议但未跟进）
5. **事件驱动、幂等、对账、OAuth 等关键能力完全缺失**（文档未充分覆盖）

### 建议下一步

| 优先级 | 任务 | 修正文档对应项 |
|--------|------|---------------|
| P0 | 接入 Redis 客户端库 + 各服务 data 层集成 | §16.3, §16.6 |
| P0 | 补齐 4 个服务健康检查 + 统一命名 | §16.3, §18.5 |
| P0 | 密码哈希（bcrypt） | — |
| P0 | 幂等键去重 | — |
| P1 | 实现 xtrace 包（OpenTelemetry） | §16.1, §17.2 |
| P1 | 二期服务补充 gRPC proto | §17.5, §18.11 |
| P1 | 事件总线引入 | — |
| P2 | 监控 metrics 端点 | §17.10 |
| P2 | 分布式限流 | — |
| P3 | OAuth/SSO | — |
