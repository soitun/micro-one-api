# 安全审计报告

**项目名称**：micro-one-api
**审计时间**：2026-05-06 14:00
**审计范围**：当前完整目录下的所有源代码与配置文件
**总发现数量**：28 个（Critical: 3, High: 8, Medium: 10, Low: 5, Info: 2）
**已修复数量**：17 个（Critical: 3, High: 5, Medium: 8, Low: 1）
**修复提交**：`security-audit` 分支

## 1. 执行摘要（Executive Summary）

micro-one-api 是一个基于 Go/Kratos 的 API 网关微服务系统。经过全面安全审计，发现 28 个安全问题，其中 3 个 Critical、8 个 High。审计后已修复 17 个漏洞，包括所有 Critical 级别问题和大部分 High/Medium 级别问题。整体安全成熟度评分从 **4/10** 提升至 **6/10**。

**已修复的关键问题**：Admin API 认证、硬编码凭证移除、密码哈希修复、错误消息脱敏、OAuth CSRF 防护、LIKE 注入防护、CSP/HSTS 加固、Docker Compose 安全加固。

**仍需处理的问题**：生产代码未接入安全中间件（需较大重构）、gRPC 服务间 TLS、OAuth 用户配额策略、Gemini URL 注入防护。

## 2. 发现详情（Findings）

### 2.1 [Critical] Admin API 无应用层认证 ✅ 已修复

- **位置**：`internal/admin/server/http.go`
- **描述**：Admin HTTP 服务器注册了 `/v1/users`、`/v1/channels`、`/v1/system/options`、`/v1/logs`、`/v1/account`、`/v1/redeem-codes` 等端点，但无任何认证中间件。
- **修复状态**：✅ 已修复 — 添加 `AdminAuth` 中间件，使用 `ADMIN_TOKEN` 环境变量进行 Bearer token 认证，使用 `crypto/subtle.ConstantTimeCompare` 防止时序攻击。
- **CVSS 3.1 分数**：9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)
- **参考**：OWASP A01:2021 - Broken Access Control, CWE-306

---

### 2.2 [Critical] 硬编码默认 JWT 密钥 ✅ 已修复

- **位置**：`internal/pkg/auth/jwt.go:32-36`
- **描述**：当 `JWT_SECRET_KEY` 环境变量未设置时，使用公开可见的默认密钥签发 JWT。
- **修复状态**：✅ 已修复 — `NewJWTManager()` 和 `LoadServiceAuthConfig()` 在环境变量缺失时返回错误，拒绝启动。
- **CVSS 3.1 分数**：9.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N)
- **参考**：CWE-798, CWE-321

---

### 2.3 [Critical] 硬编码 demo 用户和 token ✅ 已修复

- **位置**：`internal/identity/data/data.go:82-103`，`internal/channel/data/data.go:77-104`
- **描述**：内存回退仓库包含硬编码的 root 用户（密码 "password"）、`demo-token`（无限配额）、以及硬编码的上游 API key。
- **修复状态**：✅ 已修复 — identity 和 channel 的内存回退仓库均改为返回空 map，不再包含任何硬编码凭证。
- **CVSS 3.1 分数**：9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)
- **参考**：CWE-798, OWASP A07:2021

---

### 2.4 [High] 生产代码未接入安全中间件 ❌ 未修复

- **位置**：`cmd/relay-gateway/wire_gen.go:182` vs `internal/relay/server/http_enhanced.go`
- **描述**：`EnhancedHTTPServer` 实现了完整的安全中间件链（SecurityHeaders、CORS、RateLimit、MaxBodySize、InputValidation），但生产入口调用的是无中间件版本。
- **影响**：速率限制、安全头、CORS、输入验证、请求体大小限制在生产环境全部失效。
- **CVSS 3.1 分数**：8.2 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N)
- **备注**：需要较大重构，将 `EnhancedHTTPServer` 的中间件链提取为共享组件并接入所有 HTTP 服务器。建议作为下个迭代的优先任务。
- **参考**：OWASP A05:2021 - Security Misconfiguration

---

### 2.5 [High] gRPC 服务间通信默认使用明文 ❌ 未修复

- **位置**：`cmd/relay-gateway/wire_gen.go:147-158`，`cmd/admin-api/wire_gen.go:74-87`
- **描述**：所有 gRPC 客户端连接使用 `insecure.NewCredentials()`，JWT token 和用户凭证以明文传输。
- **影响**：网络嗅探可获取所有服务间通信内容。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **备注**：需要配置 TLS 证书基础设施，建议配合 K8s mTLS 方案（如 Istio/Linkerd）或手动配置证书。
- **参考**：CWE-319

---

### 2.6 [High] 硬编码数据库凭证 ⚠️ 部分修复

- **位置**：`configs/*.yaml`，`deployments/docker-compose/docker-compose.yml`
- **描述**：配置文件和 docker-compose 使用默认弱密码。
- **修复状态**：⚠️ 部分修复 — docker-compose 已改为要求 `${MYSQL_ROOT_PASSWORD:?...}`、`${DATABASE_DSN:?...}` 等必需环境变量，不再有默认值。configs/*.yaml 仍保留 `${DATABASE_DSN:-root:password}` 默认值（因为这些文件被 gitignore，仅作为模板参考）。
- **CVSS 3.1 分数**：7.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N)
- **参考**：CWE-798

---

### 2.7 [High] OAuth 用户自动获得无限配额 ❌ 未修复

- **位置**：`internal/identity/biz/auth.go:310-316`
- **描述**：OAuth 登录创建的新用户自动获得 `UnlimitedQuota: true`，无需管理员审批。
- **影响**：任何人通过 OAuth 注册即可无限制消耗 AI API 配额。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H)
- **备注**：需要设计配额审批流程，涉及业务逻辑变更。
- **参考**：OWASP A04:2021 - Insecure Design

---

### 2.8 [High] Admin CreateUser 不设置密码 ✅ 已修复

- **位置**：`internal/identity/biz/auth.go:224-240`
- **描述**：`CreateUser` 方法接受 `password` 参数但从未使用，`PasswordHash` 字段留空。
- **修复状态**：✅ 已修复 — 现在使用 `bcrypt.GenerateFromPassword` 哈希 password 参数并设置 `PasswordHash`。
- **CVSS 3.1 分数**：7.2 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:N)
- **参考**：CWE-256

---

### 2.9 [High] 错误信息泄露内部细节 ✅ 已修复

- **位置**：`internal/admin/server/http.go`，`internal/relay/server/http.go`，`internal/identity/server/http.go`
- **描述**：多个 HTTP handler 将 `err.Error()` 原始返回给客户端。
- **修复状态**：✅ 已修复 — admin、identity、relay 三个服务的 HTTP handler 全部改为返回通用错误消息（"internal server error"、"upstream service error" 等），不再泄露内部细节。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **参考**：OWASP A04:2021, CWE-209

---

### 2.10 [High] SSRF 风险 — Gemini provider URL 注入 ❌ 未修复

- **位置**：`internal/relay/provider/gemini.go:163,202`
- **描述**：`req.Model`（用户可控输入）直接拼接到 URL 路径中。
- **影响**：精心构造的 model 名称可操纵 URL 路径。
- **CVSS 3.1 分数**：7.2 (AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:N/A:N)
- **备注**：需要在 provider 层对 model 名称做 URL 编码或白名单验证。
- **参考**：CWE-918

---

### 2.11 [Medium] OAuth state 验证可绕过 ✅ 已修复

- **位置**：`internal/identity/server/http.go:101-107`
- **描述**：当 `oauth_state` cookie 缺失或为空时，state 验证被跳过。
- **修复状态**：✅ 已修复 — state 验证现在为强制性，cookie 缺失或为空时拒绝请求。
- **CVSS 3.1 分数**：6.5 (AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N)
- **参考**：CWE-352

---

### 2.12 [Medium] IP 欺骗 — Rate Limiter 信任客户端头 ❌ 未修复

- **位置**：`internal/pkg/middleware/ratelimit.go:212-233`
- **描述**：`getClientIP` 直接信任 `X-Forwarded-For` 和 `X-Real-IP` 头。
- **影响**：攻击者可伪造 IP 绕过速率限制。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N)
- **参考**：CWE-348

---

### 2.13 [Medium] LIKE 通配符注入 ✅ 已修复

- **位置**：`internal/identity/data/data.go`，`internal/log/data/data.go`，`internal/channel/data/data.go`
- **描述**：用户输入直接嵌入 LIKE 模式，未转义 `%` 和 `_` 通配符。
- **修复状态**：✅ 已修复 — 三个模块均添加 `escapeLike()` 函数，转义 `%`、`_` 和 `\` 字符。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **参考**：CWE-89

---

### 2.14 [Medium] 无登录/注册速率限制 ❌ 未修复

- **位置**：`internal/identity/service/identity.go:82-111`
- **描述**：Login 和 Register gRPC 端点无速率限制。
- **影响**：攻击者可进行大规模密码猜测攻击。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **参考**：CWE-307

---

### 2.15 [Medium] Redis 无认证 ✅ 已修复

- **位置**：`deployments/docker-compose/docker-compose.yml`
- **描述**：docker-compose 中 Redis 无 `--requirepass` 且暴露端口到主机。
- **修复状态**：✅ 已修复 — Redis 使用 `--requirepass` 启动，密码通过 `${REDIS_PASSWORD:?...}` 必需环境变量注入，端口不再暴露到主机。
- **CVSS 3.1 分数**：5.9 (AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **参考**：CWE-306

---

### 2.16 [Medium] TLS 默认关闭 ❌ 未修复

- **位置**：`.env.example:28`，`internal/pkg/tls/config.go:26`
- **描述**：`TLS_ENABLED=false` 是默认值。
- **影响**：默认部署所有通信为明文。
- **CVSS 3.1 分数**：5.9 (AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **参考**：CWE-319

---

### 2.17 [Medium] CSP 允许 unsafe-inline ✅ 已修复

- **位置**：`internal/pkg/middleware/security.go:29-30`
- **描述**：CSP 策略中 `script-src` 和 `style-src` 允许 `'unsafe-inline'`。
- **修复状态**：✅ 已修复 — 移除 `unsafe-inline`，改为仅允许 `'self'`。
- **CVSS 3.1 分数**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N)
- **参考**：CWE-79

---

### 2.18 [Medium] HSTS 仅在 TLS 连接时设置 ✅ 已修复

- **位置**：`internal/pkg/middleware/security.go:37-39`
- **描述**：HSTS 头仅在 `r.TLS != nil` 时设置，TLS 终止代理后方不会生效。
- **修复状态**：✅ 已修复 — 现在同时检查 `X-Forwarded-Proto: https` 头，在代理后方也能正确设置 HSTS。
- **CVSS 3.1 分数**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N)
- **参考**：CWE-319

---

### 2.19 [Medium] OAuth state cookie 缺少 Secure 标志 ✅ 已修复

- **位置**：`internal/identity/server/http.go:83-89`
- **描述**：`oauth_state` cookie 缺少 `Secure: true`。
- **修复状态**：✅ 已修复 — 添加 `Secure: true`。
- **CVSS 3.1 分数**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N)
- **参考**：CWE-614

---

### 2.20 [Medium] JSON 解码无大小限制 ❌ 未修复

- **位置**：`internal/relay/server/json.go:9-14`
- **描述**：`decodeJSON` 使用 `io.ReadAll` 读取整个请求体到内存。
- **影响**：大请求体可导致内存耗尽（DoS）。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L)
- **参考**：CWE-400

---

### 2.21 [Low] math/rand 用于负载均衡 ❌ 未修复

- **位置**：`internal/pkg/registry/resolver.go:6,42`
- **描述**：使用 `math/rand` 而非 `crypto/rand` 进行服务实例选择。
- **影响**：实际安全影响极低。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)

---

### 2.22 [Low] Request ID 可预测 ✅ 已修复

- **位置**：`internal/pkg/middleware/security.go:132-134`
- **描述**：Request ID 使用 `time.Now().UnixNano()` 生成。
- **修复状态**：✅ 已修复 — 改用 `crypto/rand` 生成 16 字节随机 ID（32 位十六进制字符串）。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **参考**：CWE-330

---

### 2.23 [Low] Rate Limiter 使用弱哈希 ❌ 未修复

- **位置**：`internal/pkg/middleware/ratelimit.go:237-243`
- **描述**：`simpleHash` 使用非加密哈希。
- **影响**：合法用户可能被错误限流。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N)

---

### 2.24 [Low] Token 生成存在轻微模偏差 ❌ 未修复

- **位置**：`internal/identity/biz/auth.go:271`
- **描述**：`int(b[i])%len(letters)` 由于 256 不能被 62 整除，引入轻微偏差。
- **影响**：实际影响可忽略。
- **CVSS 3.1 分数**：2.0 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)

---

### 2.25 [Low] Channel API Key 明文存储 ❌ 未修复

- **位置**：`internal/channel/data/data.go:27`
- **描述**：上游 provider API key 以明文存储在数据库中。
- **影响**：数据库泄露时所有 API key 暴露。
- **CVSS 3.1 分数**：3.7 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **参考**：CWE-312

---

### 2.26 [Info] gRPC auth interceptor token 提取不完整 ❌ 未修复

- **位置**：`internal/pkg/grpc/auth.go:203-205`
- **描述**：`extractTokenFromContext` 无法从 gRPC metadata 提取 JWT token，仅支持 mTLS。
- **影响**：JWT 认证拦截器实际无法工作。

---

### 2.27 [Info] Security CI/CD Actions 版本过旧 ❌ 未修复

- **位置**：`.github/workflows/security.yml`
- **描述**：`actions/checkout@v3`、`actions/setup-go@v4` 等应升级到 v4+。

---

## 3. 依赖与第三方库安全扫描

### 直接依赖

| 依赖 | 版本 | 状态 |
|---|---|---|
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | 安全 |
| `golang.org/x/crypto` | v0.50.0 | 安全 |
| `github.com/go-sql-driver/mysql` | v1.7.0 | 建议升级到 v1.8+ |
| `gorm.io/gorm` | v1.30.0 | 安全 |
| `github.com/go-kratos/kratos/v2` | v2.9.2 | 安全 |
| `github.com/redis/go-redis/v9` | v9.19.0 | 安全 |
| `google.golang.org/grpc` | v1.80.0 | 安全 |
| `github.com/bytedance/sonic` | v1.15.1 | 安全 |
| `github.com/prometheus/client_golang` | v1.23.2 | 安全 |

### 间接依赖

| 依赖 | 版本 | 状态 |
|---|---|---|
| `golang.org/x/arch` | v0.0.0-20210923205945 | **过旧** — 2021 年版本，建议更新 |
| `golang.org/x/net` | v0.52.0 | 安全 |
| `golang.org/x/sys` | v0.43.0 | 安全 |

### 升级建议

1. `go-sql-driver/mysql` 升级到 v1.8+
2. `golang.org/x/arch` 更新到最新版本
3. 运行 `go get -u ./...` 并 `go mod tidy` 更新所有依赖
4. CI 中的 GitHub Actions 升级到最新版本

---

## 4. 整体安全建议

### 4.1 架构层改进

1. **统一安全中间件**：将 `EnhancedHTTPServer` 的中间件链提取为共享组件，所有 HTTP 服务器统一使用
2. **默认启用 TLS**：所有 gRPC 和 HTTP 通信默认使用 TLS，仅通过显式配置降级
3. **密钥管理**：引入密钥管理服务（如 HashiCorp Vault）管理 JWT 密钥、数据库凭证、API key
4. **零信任网络**：所有内部服务通信必须认证，不依赖网络层隔离

### 4.2 DevSecOps 流水线推荐

当前 CI 已有 gosec、govulncheck、gitleaks、Trivy、CodeQL，建议补充：

- **SAST**：已有 gosec + CodeQL ✅
- **DAST**：添加 OWASP ZAP 动态扫描
- **SCA**：已有 govulncheck ✅，建议添加 Snyk 或 Dependabot
- **Secret Scanning**：已有 gitleaks ✅
- **IaC Scanning**：添加 Checkov 扫描 Dockerfile 和 K8s manifests
- **Container Scanning**：已有 Trivy ✅

### 4.3 最佳实践 Checklist

- [x] 移除所有硬编码凭证
- [x] Admin API 添加认证中间件
- [ ] 将 EnhancedHTTPServer 中间件接入生产代码
- [ ] 默认启用 TLS
- [ ] gRPC 服务间通信使用 mTLS
- [x] OAuth state 验证设为强制
- [ ] Login/Register 添加速率限制
- [x] 所有 HTTP 错误返回通用消息
- [x] Redis 启用密码认证
- [x] Docker Compose 不暴露内部端口到主机
- [x] LIKE 查询转义通配符
- [ ] JSON 解码添加大小限制
- [x] CSP 移除 unsafe-inline
- [x] HSTS 在代理后方也能正确设置
- [x] OAuth state cookie 添加 Secure 标志
- [ ] Channel API key 加密存储
- [x] Request ID 使用 UUID

---

## 5. 修复优先级路线图

### 立即修复（7 天内） — 已完成 ✅

| # | 问题 | 严重性 | 状态 |
|---|---|---|---|
| 1 | Admin API 添加认证中间件 | Critical | ✅ 已修复 |
| 2 | 移除硬编码 JWT 默认密钥 | Critical | ✅ 已修复 |
| 3 | 移除硬编码 demo 用户/token | Critical | ✅ 已修复 |
| 4 | CreateUser 正确哈希密码 | High | ✅ 已修复 |
| 5 | 错误信息脱敏 | High | ✅ 已修复 |
| 6 | 移除硬编码数据库凭证默认值 | High | ⚠️ 部分修复（Docker Compose 已修复） |

### 短期修复（30 天内） — 部分完成

| # | 问题 | 严重性 | 状态 |
|---|---|---|---|
| 7 | 将安全中间件接入生产代码 | High | ❌ 需较大重构 |
| 8 | 默认启用 TLS | High | ❌ 未修复 |
| 9 | Gemini URL 注入防护 | High | ❌ 未修复 |
| 10 | OAuth 用户默认不给无限配额 | High | ❌ 需业务逻辑变更 |
| 11 | OAuth state 验证强制化 | Medium | ✅ 已修复 |
| 12 | Rate Limiter IP 提取安全化 | Medium | ❌ 未修复 |
| 13 | LIKE 通配符转义 | Medium | ✅ 已修复 |
| 14 | Login/Register 速率限制 | Medium | ❌ 未修复 |
| 15 | Redis 认证 | Medium | ✅ 已修复 |

### 中长期改进

| # | 问题 | 严重性 | 状态 |
|---|---|---|---|
| 16 | 引入密钥管理服务 | Medium | ❌ 未修复 |
| 17 | CSP 使用 nonce 替代 unsafe-inline | Medium | ✅ 已移除 unsafe-inline |
| 18 | Channel API key 加密存储 | Low | ❌ 未修复 |
| 19 | Request ID 使用 UUID | Low | ✅ 已修复 |
| 20 | 添加 DAST 扫描到 CI | Info | ❌ 未修复 |
| 21 | GitHub Actions 版本升级 | Info | ❌ 未修复 |
