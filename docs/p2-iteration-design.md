# P2 迭代设计方案

> 基于 `gap-analysis-and-fix-plan.md` 中标记的 P2 待办事项，制定详细实施方案。

## 1. 迭代范围

| # | 任务 | 优先级 | 预计工作量 | 依赖 |
|---|------|--------|-----------|------|
| 1 | 链路追踪 Jaeger 集成 | P2-High | 2h | 无 |
| 2 | 对账任务定时调度 | P2-Medium | 1h | 无 |
| 3 | 二期服务集成测试 | P2-Medium | 3h | 无 |

## 2. 任务 1：链路追踪 Jaeger 集成

### 2.1 现状分析

`internal/pkg/xtrace/trace.go` 已存在但仅包含基础结构，未实际集成 OpenTelemetry + Jaeger。

### 2.2 设计方案

**技术选型**：OpenTelemetry SDK + Jaeger Exporter

**架构**：
```
Service → OTel SDK → Jaeger Collector → Jaeger UI
```

**实现步骤**：

1. **扩展 xtrace 包**：
   ```go
   // internal/pkg/xtrace/trace.go
   type Config struct {
       Endpoint string // Jaeger endpoint
       Service  string // Service name
       Enabled  bool   // Enable/disable tracing
   }

   func InitTracer(cfg Config) (func(), error) {
       // 1. Create OTel exporter (OTLP/HTTP to Jaeger)
       // 2. Create resource with service name
       // 3. Create TracerProvider
       // 4. Set global TracerProvider
       // 5. Return shutdown function
   }
   ```

2. **各服务 main.go 集成**：
   ```go
   shutdown, err := xtrace.InitTracer(xtrace.Config{
       Endpoint: cfg.Trace.Endpoint,
       Service:  "identity-service",
       Enabled:  cfg.Trace.Enabled,
   })
   defer shutdown()
   ```

3. **gRPC 拦截器自动注入**：
   ```go
   // 在 server/grpc.go 中添加
   grpcSrv := grpc.NewServer(
       grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
       grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
   )
   ```

4. **HTTP 中间件自动注入**：
   ```go
   // 在 server/http.go 中添加
   srv.Use(otelhttp.NewMiddleware("service-name"))
   ```

**配置文件扩展**：
```yaml
trace:
  enabled: true
  endpoint: "http://jaeger:4318/v1/traces"
  sample_rate: 1.0
```

**Docker Compose 扩展**：
```yaml
jaeger:
  image: jaegertracing/all-in-one:latest
  ports:
    - "16686:16686"  # UI
    - "4318:4318"    # OTLP HTTP
  environment:
    COLLECTOR_OTLP_ENABLED: true
```

### 2.3 验证标准

- [ ] Jaeger UI 可访问 (http://localhost:16686)
- [ ] 请求链路可在 Jaeger 中查看
- [ ] 跨服务调用链路可追踪（relay → identity → channel → billing）

## 3. 任务 2：对账任务定时调度

### 3.1 现状分析

`internal/billing/biz/reconciliation.go` 已实现 `ReconciliationUsecase`，包含：
- 清理过期 reservation
- 检查账户 quota 一致性

但无定时触发机制。

### 3.2 设计方案

**技术选型**：使用 `robfig/cron/v3` 实现定时调度

**架构**：
```
billing-service main.go
  └── Cron Scheduler
        └── ReconciliationJob (每小时)
              └── ReconciliationUsecase.RunReconciliation()
```

**实现步骤**：

1. **添加 cron 依赖**：
   ```bash
   go get github.com/robfig/cron/v3
   ```

2. **在 billing-service main.go 中添加调度**：
   ```go
   // cmd/billing-service/main.go
   c := cron.New()
   c.AddFunc("0 * * * *", func() { // 每小时执行
       result, err := reconUsecase.RunReconciliation(context.Background())
       if err != nil {
           logger.Errorf("reconciliation failed: %v", err)
       } else {
           logger.Infof("reconciliation completed: expired=%d, inconsistencies=%d",
               result.ExpiredCleaned, len(result.AccountInconsistencies))
       }
   })
   c.Start()
   defer c.Stop()
   ```

3. **配置化调度频率**：
   ```yaml
   billing:
     reconciliation:
       enabled: true
       schedule: "0 * * * *"  # cron expression
   ```

### 3.3 验证标准

- [ ] 对账任务按配置频率执行
- [ ] 过期 reservation 被正确清理
- [ ] 账户不一致被检测并记录

## 4. 任务 3：二期服务集成测试

### 4.1 现状分析

当前集成测试仅覆盖 relay 流程（`test/integration/relay_test.go`）。

二期服务（config, log, monitor, notify）缺少端到端测试。

### 4.2 设计方案

**测试范围**：

| 服务 | 测试场景 |
|------|---------|
| config-service | CRUD 配置项、事件通知 |
| log-service | 日志写入、查询、过滤 |
| monitor-service | 健康检查保存、告警规则 CRUD |
| notify-service | 通知创建、状态更新、列表查询 |

**实现步骤**：

1. **创建测试辅助函数**：
   ```go
   // test/integration/helpers.go
   func setupConfigService(t *testing.T, addr string) (func(), configv1.ConfigServiceClient)
   func setupLogService(t *testing.T, addr string) (func(), logv1.LogServiceClient)
   func setupMonitorService(t *testing.T, addr string) (func(), monitorv1.MonitorServiceClient)
   func setupNotifyService(t *testing.T, addr string) (func(), notifyv1.NotifyServiceClient)
   ```

2. **编写各服务测试**：
   ```go
   // test/integration/config_test.go
   func TestConfigIntegration(t *testing.T) {
       cleanup, client := setupConfigService(t, "127.0.0.1:19010")
       defer cleanup()

       t.Run("SetAndGet", func(t *testing.T) { ... })
       t.Run("ListAndDelete", func(t *testing.T) { ... })
   }
   ```

3. **测试数据隔离**：每个测试使用独立 namespace 避免冲突

### 4.3 验证标准

- [ ] 4 个服务各有至少 3 个测试场景
- [ ] 测试可独立运行，无数据污染
- [ ] `go test ./test/integration/...` 全部通过

## 5. 实施顺序

```
Phase 1: 链路追踪集成 (2h)
  ├── xtrace 包扩展
  ├── 各服务 main.go 集成
  └── Docker Compose Jaeger 配置

Phase 2: 对账定时调度 (1h)
  ├── 添加 cron 依赖
  └── billing-service 调度集成

Phase 3: 二期服务集成测试 (3h)
  ├── 测试辅助函数
  ├── config-service 测试
  ├── log-service 测试
  ├── monitor-service 测试
  └── notify-service 测试
```

## 6. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Jaeger 增加部署复杂度 | 运维成本 | 使用 all-in-one 镜像，开发环境可选关闭 |
| Cron 任务与主进程竞争资源 | 性能影响 | 限制并发、设置超时 |
| 集成测试依赖外部服务 | CI 不稳定 | 使用 mock 或 testcontainers |
