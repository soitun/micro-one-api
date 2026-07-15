# Micro-One-API v0.7.2 发布公告

> 2026-07-15 · 上一版：[v0.7.1](./release-v0.7.1.md)（2026-07-14）

v0.7.2 是 v0.7.1 之后的 **PATCH** 版本，聚焦 OAuth 绑定回调修复，以及 Docker Compose、Kubernetes 和数据库迁移流程的发布可用性收口。

本版**没有新增业务表迁移**，**没有破坏性 API 变更**。部署流程有一项重要变化：应用不再依赖 MySQL `/docker-entrypoint-initdb.d` 隐式初始化，而是由一次性 `migrate` 任务显式执行迁移；迁移失败时应用不会继续启动。

## 亮点

- **OAuth 绑定回调恢复可用**：admin-api 代理 `/api/oauth/*` 和 `/v1/oauth/*` 到 identity-service，浏览器回调统一走 `/v1/oauth/{provider}/callback`；绑定成功后返回个人资料页，不再误进入普通 OAuth 登录流程。
- **全新 Compose 部署可重复**：MySQL 健康后先运行一次性 `migrate`，成功后九个应用服务才启动；全新环境 smoke 共 23 项全部通过。
- **数据库迁移失败可见**：修复 SQL 字符串内分号的拆分逻辑，修正 `phase1_indexes.sql` 的错误列名，并将有额外前置条件的 `phase3_partitioning.sql` 排除出自动迁移。
- **Kubernetes 清单与文档一致**：统一 `db-credentials`、补齐 `admin-tls-secret` 和服务间 `SERVICE_TOKEN`，移除生产必需 Secret 的 `optional: true`，修正 Admin 端口、服务引用和 NetworkPolicy。
- **镜像构建可复现**：九个服务与迁移镜像支持当前 amd64/arm64 构建链路，生产清单使用固定 `v0.7.2` 占位标签而非浮动 `latest`。

## 变更内容

### Fixed

- admin-api 新增 OAuth 绑定和回调代理路由，并保留 query、Authorization、Cookie、重定向和 `Set-Cookie` 响应。
- identity-service 在 OAuth callback 中恢复绑定态快照，绑定已有用户身份后重定向到 `/profile?oauth_bind=success&provider=...`。
- Admin Kubernetes Deployment 的容器端口、探针和 Service targetPort 统一为实际监听端口 `8000`。
- K8s NetworkPolicy 补齐 admin-api 到 identity/channel/billing/log/notify 的必要 HTTP egress。
- 迁移 SQL 分割器不再把字符串字面量中的分号误判为语句边界。
- `phase1_indexes.sql` 使用实际存在的账务列，并避免全新数据库迁移失败。

### Changed

- Docker Compose 删除 MySQL `/docker-entrypoint-initdb.d` 挂载，新增一次性 `migrate` 服务；所有应用通过 `service_completed_successfully` 等待迁移完成。
- 自动迁移执行编号 SQL 和 `phase1_indexes.sql`，不自动执行需要额外主键、维护窗口和 MySQL 分区前置条件的 `phase3_partitioning.sql`。
- K8s 使用 `kubectl apply -k deployments/k8s` 作为统一部署入口，并为九个服务提供固定版本镜像覆盖。
- `billing-service`、`log-service`、relay 和 admin 使用相同的 `SERVICE_TOKEN`；`config-service` 不再注入代码未读取的该变量。
- 部署文档补齐 Secret 创建、镜像替换、数据库迁移、rollout、内部接口验证，以及旧 Compose 数据卷升级步骤。

## 数据库迁移

本版没有新增编号迁移文件，但**迁移执行方式发生变化**。

全新 MySQL 环境会由一次性 `migrate` 服务按顺序执行 55 项自动迁移（编号迁移和 `phase1_indexes.sql`），成功后才启动应用。`phase3_partitioning.sql` 仍是可选运维操作，不会自动执行。

从 v0.7.1 或更早的旧 Compose 数据卷升级时，先备份数据库。若已确认旧 schema 完整执行到 `phase1_indexes.sql`，可先登记 brownfield baseline：

```bash
cd deployments/docker-compose
docker compose --env-file .env up -d mysql redis
docker compose --env-file .env run --rm migrate -baseline phase1_indexes
docker compose --env-file .env up -d --build
```

不要对结构不完整或曾初始化失败的数据库直接执行 baseline；baseline 只登记版本，不会补建缺失表、列或索引。详细说明见 [部署运维文档](../deployment.md#81-从-v071-或更早的-compose-数据卷升级)。

## 破坏性变更

API 和数据库 schema 均无破坏性变更。运维侧需要注意：

- Compose 现在强制要求迁移成功，旧数据卷可能需要一次 brownfield baseline。
- 生产必需的数据库、Redis、JWT、Admin 和服务间令牌不再允许缺失。
- Kubernetes 示例镜像为 `your-registry/<service>:v0.7.2` 占位符，部署前必须替换成实际 registry 或 digest。

## 升级步骤

```bash
git fetch --tags
git checkout v0.7.2

# 检查并替换 deployments/docker-compose/.env 中的生产密钥
cd deployments/docker-compose
docker compose --env-file .env config --quiet

# 旧数据卷先按上文完成备份和 baseline；全新环境直接启动
docker compose --env-file .env up -d --build
```

Kubernetes 部署应先运行迁移，再执行 `kubectl apply -k deployments/k8s`，并等待九个 Deployment rollout 完成。完整步骤和 Secret 清单见 [docs/deployment.md](../deployment.md)。

## 验证

发布前已执行：

```bash
./scripts/check-architecture.sh                                      # 通过
make test-unit                                                      # 全部通过
cd web && npm test && npm run lint                                  # 24 files / 72 tests，ESLint 通过
docker compose --env-file .env.example config --quiet              # 通过
./scripts/test-docker-compose.sh                                    # 23 passed，0 failed
go test ./app/admin/internal/server ./app/identity/internal/server  # OAuth 路由定向测试通过
```

全新 Docker Compose smoke 中，MySQL/Redis 健康、迁移任务成功退出、九个应用容器运行，七个内部 HTTP 健康端点、log-service 共享令牌鉴权、Relay `/healthz` 和 `/v1/models` 未授权响应均符合预期。kind v1.33.1 smoke 中九个应用及 MySQL/Redis 均达到 `1/1 Running`，Admin 可访问 billing/log 内部接口，Relay 健康检查成功。

`develop` 头提交 `2ecee00` 的 GitHub CI 和 Security Pipeline 均已通过，包括 Backend、Frontend、amd64/arm64 Docker Build、CodeQL、license scan 和 security scan。
