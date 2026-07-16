# Micro-One-API v0.6.1 ~ v0.8.0 发版公告:从品牌落地到大仓重构与客户端体验

![Micro-One-API community cover](../assets/micro-one-api-community-cover.svg)

## 摘要

本文整理 `micro-one-api` 从 v0.6.1 到 v0.8.0 共 5 个版本的发布内容。这段时间内项目完成了品牌视觉落地、Kratos 官方大仓结构迁移、管理后台稳定性修复、OAuth 与部署可用性收口,以及面向客户端的 API 指南与 CC Switch 一键导入。范围覆盖 `v0.6.1..v0.8.0` 共 29 次提交、593 个文件、+12.0k/-3.7k 行。

整个周期没有 API 破坏性变更,也没有新增业务表迁移。v0.7.0 的结构重构和 v0.7.2 的部署流程变化需要运维侧关注,其余版本只需重建镜像并滚动重启。

## 版本总览

| 版本 | 日期 | 类型 | 关键词 |
|------|------|------|--------|
| [v0.6.1](#v061) | 2026-07-10 | PATCH | 品牌视觉落地、logo 404 修复 |
| [v0.7.0](#v070) | 2026-07-12 | MINOR | Kratos 大仓结构迁移、架构边界守卫 |
| [v0.7.1](#v071) | 2026-07-14 | PATCH | 日志详情修复、服务间路由修复、安全扫描收敛 |
| [v0.7.2](#v072) | 2026-07-15 | PATCH | OAuth 回调修复、Compose/K8s/迁移可用性收口 |
| [v0.8.0](#v080) | 2026-07-16 | MINOR | API 指南页、CC Switch 导入、前端不再内嵌 |

---

## v0.6.1

> 2026-07-10 · 上一版:v0.6.0(2026-07-09)

v0.6.1 聚焦品牌视觉资产落地与管理后台 logo 资源 404 修复。无 proto 变更,无数据库迁移,无破坏性 API 变更,升级时只需替换镜像 + 重建容器即可。

### 亮点

- **品牌视觉落地**:新增项目 logo 图标、横排 wordmark SVG 资产,刷新 README 头图与发版链接。
- **管理后台 logo 404 修复**:admin-api HTTP server 补齐 `/logo-icon.svg`、`/logo-wordmark.svg` 路由,前端 logo 引用不再 404。

### 变更内容

**Added**

- `docs/assets/micro-one-api-logo-icon.svg`、`micro-one-api-logo-wordmark.svg`:项目 logo SVG 资产。
- `docs/logo-design.md`:logo 设计说明。
- `web/public/logo-icon.svg`、`logo-wordmark.svg`、`favicon.svg`:前端静态 logo 资产与 favicon 刷新。

**Changed**

- `README.md`:头图改为 logo wordmark,最新发版链接更新到 v0.6.0,功能概览补充订阅套餐与用量查询、订阅账号治理描述。
- `web/src/components/AppNavigation.tsx`:导航栏 logo 引用更新为新资产路径。

**Fixed**

- `internal/admin/server/http.go`:admin HTTP server 新增 `/logo-icon.svg` 与 `/logo-wordmark.svg` handler 注册。此前 Vite 构建输出的 logo 资产未在 admin server 路由中暴露,前端引用返回 404。
- 新增外部 web root 下 logo 资产服务回归测试。

### 兼容性与回滚

- HTTP 客户端协议完全向后兼容,所有 `/v1/*` 接口行为不变。
- gRPC proto 无变更,数据库无新增迁移。
- 回滚到 v0.6.0 镜像即可;logo 资源 404 不影响功能正确性,仅为视觉缺失。

---

## v0.7.0

> 2026-07-12 · 上一版:[v0.6.1](#v061)(2026-07-10)

v0.7.0 是 v0.6.1 之后的 **Kratos 大仓结构迁移**版本。范围覆盖 `v0.6.1..v0.7.0` 共 9 次提交、560 个文件、+8.6k/-3.2k 行。本版为纯结构性重构,不涉及 API 破坏性变更,不新增数据库迁移。

### 亮点

- **Kratos 官方大仓结构对齐**:relay-gateway 保留为根项目(`cmd/relay-gateway/` + `internal/`),其余 8 个服务迁移到 `app/<service>/cmd/<service>/` + `app/<service>/internal/`,全部共用根 `go.mod`,与 `kratos-layout` 模板和 `kratos new app/<service> --nomod` CLI 约定一致。
- **基础设施分层**:原 `internal/pkg/` 提取为 `platform/`(15 个基础设施包)和 `pkg/`(7 个纯工具包),明确区分基础设施层与业务代码。
- **共享域库**:新增 `domain/subscription` 和 `domain/upstream` 共享域库,admin/billing/relay 直接嵌入订阅域 biz+data 代码,走模块化单体(modular-monolith)边界而非额外网络调用。
- **架构边界守卫**:新增 `scripts/check-architecture.sh`,7 条层级依赖规则 + wireinject 编译检查,集成到 CI,防止层间违规依赖。
- **配置布局重构**:各服务独立 `configs/config.yaml`,Dockerfile 设 `ENV CONF_PATH=/configs/config.yaml`,部署清单不再依赖根目录多配置文件。
- **前端 502/404 修复**:修正 SubscriptionPlansPage API 路径前缀和路由注册顺序。

### 目录结构变化

```
v0.6.1 之前                          v0.7.0
─────────────────────────────────────────────────────────────
cmd/admin-api/                  →   app/admin/cmd/admin/
cmd/billing-service/            →   app/billing/cmd/billing/
cmd/channel-service/            →   app/channel/cmd/channel/
cmd/config-service/             →   app/config/cmd/config/
cmd/identity-service/           →   app/identity/cmd/identity/
cmd/log-service/                →   app/log/cmd/log/
cmd/monitor-worker/             →   app/monitor/cmd/monitor/
cmd/notify-worker/              →   app/notify/cmd/notify/
cmd/relay-gateway/              →   cmd/relay-gateway/ (保留在根)

internal/admin/                  →   app/admin/internal/
internal/billing/                →   app/billing/internal/
internal/channel/                →   app/channel/internal/
internal/config/                 →   app/config/internal/
internal/identity/               →   app/identity/internal/
internal/log/                    →   app/log/internal/
internal/monitor/                →   app/monitor/internal/
internal/notify/                 →   app/notify/internal/
internal/config (conf proto)     →   internal/conf/

internal/pkg/                    →   platform/ (基础设施) + pkg/ (纯工具)
internal/subscription/           →   domain/subscription/
internal/relay/provider + credential → domain/upstream/

api/relay/                       →   api/relay-gateway/
configs/<service>.yaml           →   app/<service>/configs/config.yaml
```

### 破坏性变更

API 和数据库 schema 均无破坏性变更,但运维侧有以下结构变化:

- **Docker 构建路径变化**:各子服务从根 Dockerfile 统一构建改为 per-service Dockerfile 独立构建。
- **配置文件路径变化**:原 `configs/<service>.yaml` 已删除,改为 `app/<service>/configs/config.yaml`。
- **Go import 路径变化**:234 处 import 路径更新。

### 升级步骤

1. 拉取新代码:`git pull` 或 checkout v0.7.0 tag。
2. 重建镜像:各服务 Dockerfile 路径已变更,需重建所有服务镜像。
3. 更新部署清单:docker-compose / k8s manifest 中的 `SERVICE_PATH` 和构建上下文路径已更新,使用仓库内最新版本。
4. 滚动重启:按 identity → channel → billing → admin → relay → log/monitor/notify 顺序。
5. 验证健康检查、API 请求、管理后台页面、订阅套餐管理页(特别验证 502/404 修复)。

---

## v0.7.1

> 2026-07-14 · 上一版:[v0.7.0](#v070)(2026-07-12)

v0.7.1 是 v0.7.0 之后的 **PATCH** 版本,聚焦管理后台日志详情与分页修复、服务间路由与鉴权配置修复,以及代码安全扫描告警的收敛。覆盖 `v0.7.0..v0.7.1` 共 7 次提交(6 `fix:` + 1 `docs:`),无 `feat` 提交。不涉及数据库迁移,无破坏性 API 变更。

### 亮点

- **管理后台日志详情修复**:admin-api 的 `/api/log/{id}` 由代理 log-service 改为直接走 billing-service 的 `GetLedgerEntry`,彻底解决 log-service 403/404 导致日志详情打不开的问题,并补充 `username` 关联字段。
- **日志分页对齐**:`/api/log` 列表返回补齐 `total` 字段,修正前端分页控件无法显示总数的问题。
- **服务间路由修复**:log/notify/monitor/config/identity 多个服务把 gorilla/mux 的 `HandleFunc`(仅精确匹配)改为 `HandlePrefix`(前缀匹配),修复 `/v1/logs/{id}` 等子路径返回裸 404 的路由回归。
- **docker-compose 鉴权修复**:log-service 和 billing-service 补齐 `SERVICE_TOKEN` 环境变量,修复 ServiceAuth 中间件因 token 未设置而对所有 `/v1/logs` 请求返回 403 的问题。
- **安全扫描收敛**:移除幂等中间件中来自 HTTP 头的明文密钥日志(CodeQL CWE-312/315/359),升级 `golang.org/x/crypto` 至 v0.54.0,并抑制无修复版本且本仓不引用的 `GO-2026-5932`(openpgp)告警。

### 变更内容

**Added**

- `api/billing/v1`:新增 `GetLedgerEntry` RPC 及 `GetLedgerEntryRequest` / `GetLedgerEntryResponse` 消息(向后兼容,不删除任何已有 RPC/字段)。
- `api/common/v1.LedgerEntry`:新增 `username` 字段(序号 24,从 `users` 表 LEFT JOIN 关联查询,读时计算不落表)。
- `app/billing/internal/biz`:`Ledger` DO 新增 `Username` 字段;`BillingUsecase.GetLedgerByID` 用例;`LedgerRepo.GetLedgerByID` 仓储接口;`ErrLedgerNotFound` 类型化错误。
- `app/billing/internal/data`:`ledgerRepo.GetLedgerByID` 实现(含 `users` LEFT JOIN)。
- `app/admin/internal/service`:`AdminService.GetLedgerEntry` 传输适配。
- `app/log/internal/server/http_getlog_route_test.go`:`/v1/logs/{id}` 路由可达性回归测试。

**Fixed**

- admin 日志详情 403/404:从"代理 log-service HTTP"改为"调用 billing `GetLedgerEntry`",消除 log-service token 未配置或路由不匹配导致的故障。
- admin 日志分页:`/api/log` 列表返回结构补齐 `total`,并修正 `page` 从 0 基改为 1 基。
- gorilla/mux 前缀路由:log/notify/monitor/config/identity 服务把 `HandleFunc("/v1/.../")` 改为 `HandlePrefix("/v1/.../")`。
- docker-compose `SERVICE_TOKEN`:log-service 与 billing-service 补齐环境变量,修复 ServiceAuth 返回 403。
- 安全扫描告警:`platform/middleware/idempotency.go` 移除来自 `Idempotency-Key` 头的明文日志字段;升级 `golang.org/x/crypto` v0.52.0 → v0.54.0。

**Changed**

- `docs/` 目录重组:45 个平铺文档分类到 `releases/`(17)、`runbooks/`(8)、`design/`(15)、`migration/`(5),新增 `docs/README.md` 导航索引。

### 兼容性

proto 变更为纯新增 RPC 与字段(序号递增,不删除/不重编号),gRPC/HTTP 向后兼容;无数据库 schema 变更;无配置项删除。部署侧:重建各服务镜像并滚动重启即可,无需执行 SQL 迁移。如使用 docker-compose,请确认已通过 `.env` 设置 `SERVICE_TOKEN`。

---

## v0.7.2

> 2026-07-15 · 上一版:[v0.7.1](#v071)(2026-07-14)

v0.7.2 是 v0.7.1 之后的 **PATCH** 版本,聚焦 OAuth 绑定回调修复,以及 Docker Compose、Kubernetes 和数据库迁移流程的发布可用性收口。没有新增业务表迁移,没有破坏性 API 变更。

### 亮点

- **OAuth 绑定回调恢复可用**:admin-api 代理 `/api/oauth/*` 和 `/v1/oauth/*` 到 identity-service,浏览器回调统一走 `/v1/oauth/{provider}/callback`;绑定成功后返回个人资料页,不再误进入普通 OAuth 登录流程。
- **全新 Compose 部署可重复**:MySQL 健康后先运行一次性 `migrate`,成功后九个应用服务才启动;全新环境 smoke 共 23 项全部通过。
- **数据库迁移失败可见**:修复 SQL 字符串内分号的拆分逻辑,修正 `phase1_indexes.sql` 的错误列名,并将有额外前置条件的 `phase3_partitioning.sql` 排除出自动迁移。
- **Kubernetes 清单与文档一致**:统一 `db-credentials`、补齐 `admin-tls-secret` 和服务间 `SERVICE_TOKEN`,移除生产必需 Secret 的 `optional: true`,修正 Admin 端口、服务引用和 NetworkPolicy。
- **镜像构建可复现**:九个服务与迁移镜像支持当前 amd64/arm64 构建链路,生产清单使用固定 `v0.7.2` 占位标签而非浮动 `latest`。

### 变更内容

**Fixed**

- admin-api 新增 OAuth 绑定和回调代理路由,并保留 query、Authorization、Cookie、重定向和 `Set-Cookie` 响应。
- identity-service 在 OAuth callback 中恢复绑定态快照,绑定已有用户身份后重定向到 `/profile?oauth_bind=success&provider=...`。
- Admin Kubernetes Deployment 的容器端口、探针和 Service targetPort 统一为实际监听端口 `8000`。
- K8s NetworkPolicy 补齐 admin-api 到 identity/channel/billing/log/notify 的必要 HTTP egress。
- 迁移 SQL 分割器不再把字符串字面量中的分号误判为语句边界。
- `phase1_indexes.sql` 使用实际存在的账务列,并避免全新数据库迁移失败。

**Changed**

- Docker Compose 删除 MySQL `/docker-entrypoint-initdb.d` 挂载,新增一次性 `migrate` 服务;所有应用通过 `service_completed_successfully` 等待迁移完成。
- 自动迁移执行编号 SQL 和 `phase1_indexes.sql`,不自动执行需要额外主键、维护窗口和 MySQL 分区前置条件的 `phase3_partitioning.sql`。
- K8s 使用 `kubectl apply -k deployments/k8s` 作为统一部署入口。
- `billing-service`、`log-service`、relay 和 admin 使用相同的 `SERVICE_TOKEN`;`config-service` 不再注入代码未读取的该变量。

### 破坏性变更

API 和数据库 schema 均无破坏性变更。运维侧需要注意:

- Compose 现在强制要求迁移成功,旧数据卷可能需要一次 brownfield baseline。
- 生产必需的数据库、Redis、JWT、Admin 和服务间令牌不再允许缺失。
- Kubernetes 示例镜像为 `your-registry/<service>:v0.7.2` 占位符,部署前必须替换成实际 registry 或 digest。

### 升级步骤

```bash
git fetch --tags
git checkout v0.7.2

cd deployments/docker-compose
docker compose --env-file .env config --quiet

# 旧数据卷升级前先备份并完成 baseline;全新环境直接启动
docker compose --env-file .env up -d --build
```

Kubernetes 部署应先运行迁移,再执行 `kubectl apply -k deployments/k8s`,并等待九个 Deployment rollout 完成。完整步骤见 [docs/deployment.md](../deployment.md)。

---

## v0.8.0

> 2026-07-16 · 上一版:[v0.7.2](#v072)(2026-07-15)

v0.8.0 是 v0.7.2 之后的 **MINOR** 版本。本版新增面向客户端的 **API 指南页**与 **CC Switch 一键导入**,并将管理后台前端从 `go:embed` 改为 `ADMIN_WEB_ROOT` 运行时提供,使前端构建产物不再进入 Git。没有新增业务表迁移,没有 API 破坏性变更。

### 亮点

- **新增 API 指南页**:`/api-guide` 页面聚合 OpenAI / Claude / Gemini 的调用示例与端点发现,从 `/status` 读取 `server_address` 与 `system_name`,无需用户手填占位符。
- **CC Switch 一键导入**:Tokens 页面新增 `ccswitch://v1/import` 深链接生成,支持 Claude Code、Codex、Gemini CLI 客户端快速导入令牌与基地址;对话框状态在重开时正确重置。
- **管理前端不再内嵌进二进制**:移除 `//go:embedall:static/web`,admin 前端统一由 `ADMIN_WEB_ROOT` 提供;构建产物加入 `.gitignore`,CI 不再需要 Node 构建步骤,"Verify generated files" 步骤不再因前端资产漂移而失败。
- **支付宝证书挂载可配置**:新增 `ALIPAY_CERT_DIR` 环境变量,宿主机密钥目录可只读挂载到容器 `/cert/alipay`。
- **路径遍历收敛**:`scripts/check-k8s-references.go` 中的文件读取改为基于 `os.Root` 的受限读取,不再可能逃逸出固定目录根。

### 变更内容

**Added**

- 新增 `/api-guide` 页面,提供 OpenAI / Claude / Gemini 调用文档与端点发现。
- 新增 `CCSwitchDialog` 组件,生成 `ccswitch://v1/import` 深链接并接入 TokensPage。
- `/status` 端点新增 `server_address` 字段,并支持可配置的 `SystemName` 选项。
- Admin 选项页新增 `ServerAddress` 文本字段。

**Changed**

- admin-api 移除 `go:embed` 前端机制,改为仅从 `ADMIN_WEB_ROOT` 解析静态文件;未配置或目录不可用时返回 500 `frontend not available`。
- admin Dockerfile 将 web-builder 构建产物复制到 `/web` 并设置 `ENV ADMIN_WEB_ROOT=/web`;其余服务 Dockerfile 移除不再使用的 web-builder 构建阶段。
- `app/admin/internal/server/static/web/` 下 54 个已跟踪构建产物从 Git 移除,并加入 `.gitignore`。
- Makefile 的 `build` 目标不再依赖 `web-build`;CI backend 任务移除 Node 环境与"Build embedded admin frontend"步骤。

**Fixed**

- CCSwitchDialog 在 Base UI 受控 Dialog 下重新打开时状态未重置的问题。
- API 指南页步骤 2 文案改为指向 Connection 区域实际渲染的 baseUrl。
- `scripts/check-k8s-references.go` 的 gosec G304/G703 路径遍历告警。
- 支付宝证书挂载可配置:Compose 各版本均新增 `ALIPAY_CERT_DIR` 挂载点。

### 破坏性变更

API 和数据库 schema 均无破坏性变更。运维侧需要注意:

- **admin 前端提供方式变更**:admin-api 不再内嵌前端资源。部署 admin 镜像时必须确保容器内存在前端构建产物并设置 `ADMIN_WEB_ROOT`(admin Dockerfile 已默认设置为 `/web`)。未设置该变量且目录不可用时,管理后台将返回 500 而非回退到内嵌资源。
- 若此前自定义了 admin Dockerfile 或运行时未设置 `ADMIN_WEB_ROOT`,升级时需同步更新。

### 升级步骤

```bash
git fetch --tags
git checkout v0.8.0

cd deployments/docker-compose
docker compose --env-file .env config --quiet

# 旧数据卷升级前先备份;全新环境直接启动
docker compose --env-file .env up -d --build
```

admin 镜像由 Dockerfile 自动将前端构建产物放到 `/web` 并设置 `ADMIN_WEB_ROOT=/web`;自定义部署需手动构建前端(`make web-dist`)并设置 `ADMIN_WEB_ROOT` 指向产物目录。

---

## 整体升级路径(v0.6.1 → v0.8.0)

如果从 v0.6.1 或更早版本直接升级到 v0.8.0,需要注意两个关键节点:

### 1. v0.7.0 结构迁移(必读)

v0.7.0 改变了所有子服务的目录结构和 Docker 构建路径。升级时必须使用仓库内最新的 docker-compose / k8s 清单,不能沿用旧版本的构建上下文路径。relay-gateway 保留在根目录,其余 8 个服务迁移到 `app/<service>/` 下独立构建。

### 2. v0.7.2 部署流程变化(必读)

v0.7.2 开始,应用不再依赖 MySQL `/docker-entrypoint-initdb.d` 隐式初始化,而是由一次性 `migrate` 任务显式执行迁移;迁移失败时应用不会继续启动。从旧 Compose 数据卷升级时需要先备份数据库,确认旧 schema 完整后登记 brownfield baseline。

### 3. v0.8.0 前端提供方式变化(必读)

v0.8.0 移除了 admin 前端的 `go:embed` 机制。如果自定义了 admin Dockerfile,必须确保容器内存在前端构建产物并设置 `ADMIN_WEB_ROOT`。官方 admin Dockerfile 已默认处理。

### 快速升级命令

```bash
git fetch --tags
git checkout v0.8.0

cd deployments/docker-compose
# 检查 .env 中的生产密钥(SERVICE_TOKEN、JWT_SECRET、ADMIN_PASSWORD 等)
docker compose --env-file .env config --quiet

# 旧数据卷升级前先备份并完成 baseline;全新环境直接启动
docker compose --env-file .env up -d --build
```

Kubernetes 部署:先运行迁移,再执行 `kubectl apply -k deployments/k8s`,并等待九个 Deployment rollout 完成。完整步骤见 [docs/deployment.md](../deployment.md)。

## 验证

各版本发布前已执行的关键验证:

```bash
# v0.7.0
go build ./...           # 通过
go vet ./...             # 无告警
make wire                # 9 个 wire_gen.go 重生成,无 diff
make test-unit           # 全部 PASS
./scripts/check-architecture.sh  # 架构边界 0 违规

# v0.7.1
go build ./...                          # 通过
go vet ./...                            # 无告警
make wire                               # 9 个 wire_gen.go 重生成
make api && make config                 # Proto 生成,工作树 clean
make test-unit                          # 全部 PASS
gosec -exclude-generated ./...           # 0 issues
git status --porcelain                  # 生成文件新鲜度验证:clean

# v0.7.2
make test-unit                          # 全部通过
./scripts/test-docker-compose.sh       # 23 passed,0 failed
cd web && npm test && npm run lint      # 72 tests,ESLint 通过

# v0.8.0
go build ./...                          # 通过
go vet ./...                            # 通过
go test $(go list ./... | grep -v '/test/e2e/suite$')  # 全部通过
```

## 结语

从 v0.6.1 到 v0.8.0,`micro-one-api` 在不到一周的时间里完成了品牌视觉落地、Kratos 官方大仓结构迁移、管理后台稳定性修复、OAuth 与部署可用性收口,以及面向客户端的 API 指南与 CC Switch 一键导入。这段时间的迭代重点不在新功能堆叠,而在工程结构的清晰化、部署流程的可靠化和客户端使用体验的改善。

v0.7.0 的大仓结构迁移让项目与 Kratos 官方模板对齐,为后续多服务独立演进打下基础;v0.7.1 和 v0.7.2 收敛了管理后台和服务间的稳定性问题;v0.8.0 则把前端从二进制内嵌改为运行时提供,并新增了 API 指南页和 CC Switch 导入,降低客户端接入成本。

如果你对多模型网关、AI API 管理平台或 Go Kratos 微服务实践感兴趣,欢迎关注、试用和参与改进。
