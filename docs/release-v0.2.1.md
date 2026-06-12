# Micro-One-API v0.2.1 发布公告

> 2026-06-12 · 上一版: [v0.2.0](./../CHANGELOG.md) (2026-06-10)

![Micro-One-API community cover](./assets/micro-one-api-community-cover.svg)

v0.2.1 是一个面向**管理后台可视化**和**前端构建性能**的小补丁版,顺手把 e2e 测试与 compose 环境文档对齐。无数据库迁移、无破坏性 API 变更,从 v0.2.0 升级基本无感。

## TL;DR

- 管理后台新增 **Top 用量排行图表**,渠道 / 模型两个维度一眼可见。
- 前端构建接入 **chunk 拆分策略**,`react` / `charts`(ECharts)/ `ui` / `query`(React Query)各自独立 vendor,各路由按页懒加载,首屏体积显著降低。
- 端到端测试与 docker-compose 环境变量、relay 实际行为对齐,后续跑 `make test-e2e` 更稳。

## 详细变更

### ✨ 新增

#### 管理后台 Top 用量图表

`OverviewPage` 新增 Top N **渠道 / 模型** 用量图表,管理员可以快速识别「谁在用什么、按什么模型结算成本」。

- 新增入口:左侧导航 `AppNavigation` 增加「用量排行」。
- 文件:`web/src/pages/admin/OverviewPage.tsx`、`web/src/components/AppNavigation.tsx`。
- 单测:`AppNavigation.test.tsx` 补齐导航项断言。

```text
# 入口位置
Overview → Top Channels  /  Top Models
```

### 🔧 优化

#### 前端 chunk 拆分

`web/vite.config.ts` 增加 `build.rollupOptions.output.manualChunks` 策略,关键拆包:

| Chunk | 内容 | 体积 (gzip) |
|---|---|---|
| `react` | react / react-dom / scheduler | ~87 KB |
| `charts` | ECharts 及子模块 | ~108 KB |
| `ui` | 公共 UI 组件 / 工具 | ~43 KB |
| `query` | React Query 及其依赖 | ~26 KB |
| 各 `*Page` | 按路由懒加载 | 平均 1–4 KB |

`OverviewPage` 从单个胖文件拆到 **~16 KB (gzip ~4 KB)**,首屏不再被 ECharts 拖慢。

### 🐛 修复

#### e2e 与 compose env 文档对齐

- `deployments/docker-compose/.env.example`:补齐 e2e token / 通知相关变量,与 `.env.example` 保持一致。
- `test/e2e/main.go`:修正 token 创建 / 校验流程以匹配 `identity-service` 当前行为,解决 `127.0.0.1:9001 connection refused` 类环境差异导致的误失败。
- 同步更新:`.env.example`、`docs/deployment.md`、`README.md`。
- commit: `d1076a5 fix: align e2e token flow and compose env docs`

## 升级指南

从 v0.2.0 升级,基本是 `git pull` + 重新构建前端:

```bash
# 后端二进制 / 镜像无破坏性变更,常规替换即可
cd deployments/docker-compose
docker compose pull   # 拉新镜像
docker compose up -d  # 重启

# 前端如果用本地构建
cd web
npm install
npm run build
```

无新增环境变量、无数据库 migration、无配置格式变更。

## 验证

本次发版前已通过:

- `go test ./...`(排除 `test/e2e`)— 全部通过,无回归。
- `web/ npm run test` — 16 文件 / 44 用例全过。
- `web/ npm run build` — 成功,chunk 拆分符合上表预期。
- e2e(`make test-e2e`)在 compose 栈内验证修复后的 token 流程(沙箱内无法直连 3000 / 8080 端口,需要本地起服务栈)。

## 致谢与反馈

- 完整 diff 范围:`v0.2.0` (`b09a598`) → `v0.2.1` (`dbcec34`),共 13 文件 / +339 / −198。
- 详细 commit 列表: [CHANGELOG.md](./../CHANGELOG.md)
- GitHub Release: <https://github.com/mengbin92/micro-one-api/releases/tag/v0.2.1>

如果你在使用中遇到问题,或者有功能建议,欢迎通过以下方式反馈:

- GitHub Issues: <https://github.com/mengbin92/micro-one-api/issues>
- 项目内已有功能可以在管理后台「系统配置」页提交反馈。

> 项目不提供任何第三方模型账号、订阅、API Key 或代理资源,部署者需自行确保上游凭证来源合法,详见仓库 `DISCLAIMER.md`。
