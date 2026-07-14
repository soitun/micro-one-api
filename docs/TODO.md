# 项目 TODO

> 最后更新：2026-07-14
>
> 当前阶段重点：先完成发布与部署收口，再完善项目展示；在这些任务完成前，暂不启动新的大型业务功能。

## P0 — 发布与部署可用性

### [ ] 合并 OAuth 回调路由修复

- [x] 等待 `develop` 提交 `2cb0a23` 的完整 CI 通过。
- [ ] 合并到 `main`。
- [x] 评估是否发布 `v0.7.2`：结论为合并部署修复后发布。

验收标准：

- GitHub CI 和 Security Pipeline 全部通过。
- OAuth 回调路由的相关单元测试通过。
- `main` 包含 `2cb0a23` 的修复。

### [ ] 同步部署方式与部署文档

关联 Issue：[部署方式是否同步更新 #5](https://github.com/mengbin92/micro-one-api/issues/5)

- [x] 统一部署文档与 K8s 清单中的数据库 Secret 名称：使用 `db-credentials`。
- [x] 在文档中补充 `admin-tls-secret` 的创建步骤。
- [x] 为 K8s `billing-service` 和 `log-service` 注入 `SERVICE_TOKEN`。
- [x] 移除生产必需 Secret 上不合理的 `optional: true`。
- [x] 核对 `config-service` 是否确实需要 `SERVICE_TOKEN`：代码不读取该变量，已从 Compose/K8s 移除。
- [x] 文档说明如何替换 `your-registry/<service>:v0.7.2`，生产示例使用固定版本而非浮动 `latest`。
- [x] 核对全部 ConfigMap、Secret、Service、Ingress 名称和端口引用。
- [ ] 验证全新 Docker Compose 部署。
- [x] 使用 kind、k3d 或测试集群执行一次 K8s smoke test。

2026-07-14 进度记录：

- `2cb0a23` 的 GitHub CI 和 Security Pipeline 均已通过，`develop` 后续头提交的两条流水线也通过。
- kind v1.33.1 smoke 中，九个应用及 MySQL/Redis 均达到 `1/1 Running`；Admin Pod 可访问 billing/log 内部接口，共享令牌鉴权成功，Relay `/healthz` 成功。
- 全新 MySQL 已一次完成 55 项自动迁移；修复了 SQL 字符串内分号解析、`phase1_indexes.sql` 错误列名，并将可选 `phase3_partitioning.sql` 排除出自动迁移。
- Compose 曾在旧初始化方式下达到 22 项 smoke 全通过，但该过程暴露出 MySQL 初始化失败后重启会掩盖不完整 schema。现已改为强制一次性 `migrate` 服务；最终全新卷复验在镜像构建完成前按当天工作安排中止，因此本项保持未完成。
- 待续：完成最终 Compose smoke，执行项目基线测试，推送并等待 `develop` CI，合并 `main` 后发布 `v0.7.2`。

验收标准：

- 新环境可以仅按照 `docs/deployment.md` 完成部署。
- 所有 Pod 正常 Ready，Admin、Relay、billing/log 内部接口可访问。
- 文档中的 Secret 名称、环境变量和清单完全一致。

## P1 — 文档与项目展示

### [ ] 增加软件界面图和用户向文档

关联 Issue：[希望能有软件界面图和文档 #6](https://github.com/mengbin92/micro-one-api/issues/6)

- [ ] 增加用户 Dashboard 截图。
- [ ] 增加 Token 管理和用量统计截图。
- [ ] 增加渠道管理或渠道健康截图。
- [ ] 增加成本分析截图。
- [ ] 增加订阅套餐或订阅账号截图。
- [ ] 增加日志详情或对账页面截图。
- [ ] 在 README 中新增“界面预览”章节。
- [ ] 增加简化架构图，以及“适合谁 / 不适合谁”说明。
- [ ] 补充从空环境部署到创建首个渠道和 Token 的最短流程。
- [ ] 修复当前文档重组后遗留的失效相对链接。
- [ ] 将 `docs/README.md` 的最新版本从 `v0.7.0` 更新为 `v0.7.1` 或当前最新版本。

建议截图目录：

```text
docs/assets/screenshots/
```

验收标准：

- 新用户在 README 中能快速了解项目界面、服务组成和主要能力。
- README 和 `docs/**/*.md` 的本地文件链接检查无错误。
- 截图不包含真实密钥、邮箱、用户数据或上游账号凭据。

### [ ] 增加部署与文档漂移检查

- [ ] CI 执行 `docker compose config`。
- [ ] 使用 `kubeconform` 或同类工具校验 `deployments/k8s/*.yaml`。
- [ ] 增加 Markdown 本地链接检查。
- [ ] 对部署清单中的必需 Secret/ConfigMap 引用增加静态检查或 smoke test。

验收标准：

- Secret 名称、失效文档链接或非法 K8s 清单能够在 PR 阶段阻断 CI。

## P2 — 路线图清理

### [ ] 重新评估 grpc-gateway 迁移计划

关联文档：[grpc-gateway 迁移 TODO](./migration/grpc-gateway-migration-todo.md)

- [ ] 明确标准 CRUD 是否继续使用 Kratos `protoc-gen-go-http` 生成的 HTTP handler。
- [ ] 评估引入 grpc-gateway runtime mux 是否有明确收益。
- [ ] 保留流式响应、WebSocket、Webhook 和 One-API 兼容路由的自定义 HTTP 实现。
- [ ] 根据评审结果更新为正式迁移计划、ADR，或标记为不再推进。

验收标准：

- 路线图不再保留已经过期但没有明确决策的版本承诺。
- 项目只维护必要的 HTTP 转换机制，避免同时维护两套重复方案。

## 基线检查

每个任务完成前至少执行：

```bash
./scripts/check-architecture.sh
make test-unit
cd web && npm test && npm run lint
```

涉及部署时追加：

```bash
cd deployments/docker-compose
docker compose config
```
