# 安全审计报告

**项目名称**：micro-one-api
**审计时间**：2026-05-06 14:00
**审计范围**：当前完整目录下的所有源代码与配置文件
**总发现数量**：28 个（Critical: 3, High: 8, Medium: 10, Low: 5, Info: 2）

## 1. 执行摘要（Executive Summary）

micro-one-api 是一个基于 Go/Kratos 的 API 网关微服务系统，整体安全成熟度评分为 **4/10**。最严重的风险集中在三个方面：(1) Admin API 端点完全缺乏应用层认证，仅依赖基础设施层防护；(2) 多处硬编码默认凭证（JWT 密钥、数据库密码、demo token）在未正确配置环境变量时会被激活；(3) 生产代码路径未接入已实现的安全中间件（速率限制、CORS、安全头、输入验证），安全代码存在但未生效。**最紧急的修复建议**：立即为 Admin API 添加认证中间件，移除所有硬编码凭证，将 EnhancedHTTPServer 的安全中间件链接入生产代码路径。

## 2. 发现详情（Findings）

### 2.1 [Critical] Admin API 无应用层认证

- **位置**：`internal/admin/server/http.go` 全文
- **描述**：Admin HTTP 服务器注册了 `/v1/users`、`/v1/channels`、`/v1/system/options`、`/v1/logs`、`/v1/account`、`/v1/redeem-codes` 等端点，但无任何认证中间件。gRPC 服务器同样使用裸 `grpc.NewServer()` 无拦截器。仅依赖 K8s Ingress nginx basic auth。
- **证据**：
```go
// internal/admin/server/http.go:16-61
func NewHTTPServer(addr string, svc *service.AdminService) *khttp.Server {
    srv := khttp.NewServer(khttp.Address(addr))
    // 直接注册路由，无任何中间件
    srv.HandleFunc("/v1/users", func(w http.ResponseWriter, r *http.Request) {
        handleListUsers(w, r, svc)
    })
    srv.HandleFunc("/v1/system/options", func(w http.ResponseWriter, r *http.Request) {
        handleGetSystemOptions(w, r, svc) // PUT 可修改系统配置
    })
    // ...
}
```
```go
// internal/admin/server/grpc.go:12
srv := grpc.NewServer() // 无认证拦截器
```
- **影响**：任何能访问 admin-api 端口的客户端可枚举用户、修改系统配置、查看日志和账单。K8s Ingress 之外的部署（Docker Compose、开发环境）完全暴露。
- **CVSS 3.1 分数**：9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)
- **修复建议**：为 Admin HTTP 和 gRPC 添加认证中间件：
```go
func NewHTTPServer(addr string, svc *service.AdminService, authMiddleware func(http.Handler) http.Handler) *khttp.Server {
    srv := khttp.NewServer(khttp.Address(addr))
    // 所有管理端点使用认证中间件
    adminChain := func(next http.HandlerFunc) http.HandlerFunc {
        return authMiddleware(next).ServeHTTP
    }
    srv.HandleFunc("/v1/users", adminChain(func(w http.ResponseWriter, r *http.Request) {
        handleListUsers(w, r, svc)
    }))
    // ... 其他路由同理
}
```
- **参考**：OWASP A01:2021 - Broken Access Control, CWE-306

---

### 2.2 [Critical] 硬编码默认 JWT 密钥

- **位置**：`internal/pkg/auth/jwt.go:32-36`
- **描述**：当 `JWT_SECRET_KEY` 环境变量未设置时，使用公开可见的默认密钥 `"dev-secret-key-change-in-production"` 签发 JWT。攻击者可直接伪造服务间认证 token。
- **证据**：
```go
secretKey := os.Getenv("JWT_SECRET_KEY")
if secretKey == "" {
    secretKey = "dev-secret-key-change-in-production"
    applogger.Log.Warn("Using default JWT secret key - change for production!")
}
```
- **影响**：攻击者可伪造任意服务身份的 JWT token，绕过服务间认证。
- **CVSS 3.1 分数**：9.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N)
- **修复建议**：启动时强制要求 JWT_SECRET_KEY，缺失则拒绝启动：
```go
func NewJWTManager() (*JWTManager, error) {
    secretKey := os.Getenv("JWT_SECRET_KEY")
    if secretKey == "" {
        return nil, fmt.Errorf("JWT_SECRET_KEY environment variable is required")
    }
    // ...
}
```
- **参考**：CWE-798, CWE-321

---

### 2.3 [Critical] 硬编码 demo 用户和 token

- **位置**：`internal/identity/data/data.go:82-103`
- **描述**：内存回退仓库包含硬编码的 root 用户（密码 "password"）和 `demo-token`（无限配额，可访问 gpt-4o-mini、gpt-4.1、claude-3-5-sonnet）。当数据库未配置时自动激活。
- **证据**：
```go
func newMemoryRepository() *Repository {
    return &Repository{
        usersByID: map[int64]*biz.User{
            1: {
                Username:     "root",
                PasswordHash: "$2a$10$PizUqaAa4Zkpmbt0zcR3ouRiWZunVYRrA7I3UD64K0Qqcdh2Cq132", // "password"
            },
        },
        tokensByKey: map[string]*biz.Token{
            "demo-token": {
                Key:            "demo-token",
                UnlimitedQuota: true,
                Models:         []string{"gpt-4o-mini", "gpt-4.1", "claude-3-5-sonnet"},
            },
        },
    }
}
```
- **影响**：未配置数据库时，任何人可用 root/password 登录，demo-token 可无限制调用 AI API。
- **CVSS 3.1 分数**：9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)
- **修复建议**：移除硬编码凭证，数据库未配置时拒绝启动或仅允许通过环境变量注入初始管理员：
```go
func newMemoryRepository() *Repository {
    // 不再提供默认用户和 token
    return &Repository{
        usersByID:   make(map[int64]*biz.User),
        tokensByKey: make(map[string]*biz.Token),
    }
}
```
- **参考**：CWE-798, OWASP A07:2021

---

### 2.4 [High] 生产代码未接入安全中间件

- **位置**：`cmd/relay-gateway/wire_gen.go:182` vs `internal/relay/server/http_enhanced.go`
- **描述**：`EnhancedHTTPServer` 实现了完整的安全中间件链（SecurityHeaders、CORS、RateLimit、MaxBodySize、InputValidation），但生产入口 `wire_gen.go` 调用的是 `httpServer.RegisterRoutes(srv)`（无中间件版本），而非 `RegisterRoutesWithSecurity`。
- **证据**：
```go
// cmd/relay-gateway/wire_gen.go:182 — 生产代码
httpServer.RegisterRoutes(srv) // 无中间件

// internal/relay/server/http_enhanced.go:49 — 已实现但未使用
func (s *EnhancedHTTPServer) RegisterRoutesWithSecurity(srv *khttp.Server) {
    // SecurityHeaders, CORS, RateLimit, MaxBodySize, Logging — 全部就绪但未接入
}
```
- **影响**：速率限制、安全头、CORS、输入验证、请求体大小限制在生产环境全部失效。
- **CVSS 3.1 分数**：8.2 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N)
- **修复建议**：将 wire_gen.go 中的 `RegisterRoutes` 替换为 `RegisterRoutesWithSecurity`，或在 `RegisterRoutes` 中添加中间件链。
- **参考**：OWASP A05:2021 - Security Misconfiguration

---

### 2.5 [High] gRPC 服务间通信默认使用明文

- **位置**：`cmd/relay-gateway/wire_gen.go:147-158`，`cmd/admin-api/wire_gen.go:74-87`
- **描述**：所有 gRPC 客户端连接使用 `insecure.NewCredentials()`，JWT token 和用户凭证以明文传输。TLS 基础设施已实现但默认关闭。
- **证据**：
```go
identityConn, err := grpc.NewClient(identityEndpoint,
    grpc.WithTransportCredentials(insecure.NewCredentials()))
```
- **影响**：网络嗅探可获取所有服务间通信内容，包括 JWT token 和用户凭证。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **修复建议**：默认启用 TLS，使用环境变量控制是否降级到明文（仅限开发）。
- **参考**：CWE-319

---

### 2.6 [High] 硬编码数据库凭证

- **位置**：`configs/*.yaml`（8个文件），`deployments/docker-compose/docker-compose.yml`
- **描述**：所有服务配置文件使用 `root:password` 作为数据库 DSN 默认值，docker-compose 使用 `rootpassword` 作为 MySQL root 密码默认值。
- **证据**：
```yaml
# configs/identity-service.yaml:9
source: ${DATABASE_DSN:-root:password@tcp(127.0.0.1:3306)/oneapi}

# docker-compose.yml:10
MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-rootpassword}
```
- **影响**：环境变量未设置时使用默认弱密码连接数据库。
- **CVSS 3.1 分数**：7.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N)
- **修复建议**：移除默认值，环境变量缺失时拒绝启动。配置文件改为仅包含变量引用。
- **参考**：CWE-798

---

### 2.7 [High] OAuth 用户自动获得无限配额

- **位置**：`internal/identity/biz/auth.go:310-316`
- **描述**：OAuth 登录创建的新用户自动获得 `UnlimitedQuota: true`，无需管理员审批。
- **证据**：
```go
tokenRecord := &Token{
    UserID:         user.ID,
    Key:            token,
    Status:         TokenStatusEnabled,
    UnlimitedQuota: true, // OAuth 用户自动无限配额
    Models:         []string{},
}
```
- **影响**：任何人通过 OAuth 注册即可无限制消耗 AI API 配额。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H)
- **修复建议**：OAuth 新用户默认配额为 0，需管理员审批后分配配额。
- **参考**：OWASP A04:2021 - Insecure Design

---

### 2.8 [High] Admin CreateUser 不设置密码

- **位置**：`internal/identity/biz/auth.go:224-240`
- **描述**：`CreateUser` 方法接受 `password` 参数但从未使用，`PasswordHash` 字段留空，导致管理员创建的用户无法通过密码登录。
- **证据**：
```go
func (uc *IdentityUsecase) CreateUser(ctx context.Context, username, displayName, email, password, group string, quota int64) (*User, error) {
    user := &User{
        Username:    username,
        DisplayName: displayName,
        // PasswordHash 未设置，password 参数被忽略
    }
    if err := uc.repo.CreateUser(ctx, user); err != nil {
        return nil, err
    }
    return user, nil
}
```
- **影响**：管理员创建的用户无法登录，且如果 login 逻辑检查空 hash 有缺陷可能导致绕过。
- **CVSS 3.1 分数**：7.2 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:N)
- **修复建议**：使用 bcrypt 哈希 password 参数并设置 PasswordHash。
- **参考**：CWE-256

---

### 2.9 [High] 错误信息泄露内部细节

- **位置**：`internal/admin/server/http.go:114`，`internal/relay/server/http.go:88,147,156`，`internal/identity/server/http.go:111,117`
- **描述**：多个 HTTP handler 将 `err.Error()` 原始返回给客户端，可能暴露数据库表结构、内部服务名、gRPC 错误详情。
- **证据**：
```go
// admin/server/http.go:114
writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})

// relay/server/http.go:147
fmt.Sprintf("upstream error after %d attempts: %v", result.Attempt+1, result.Err)
```
- **影响**：攻击者可获取内部架构信息，辅助进一步攻击。
- **CVSS 3.1 分数**：7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **修复建议**：所有 HTTP 错误返回通用消息，内部错误仅记录到日志。参考 `http_enhanced.go` 中的 `applogger.Sanitize()`。
- **参考**：OWASP A04:2021, CWE-209

---

### 2.10 [High] SSRF 风险 — Gemini provider URL 注入

- **位置**：`internal/relay/provider/gemini.go:163,202`
- **描述**：`req.Model`（用户可控输入）直接拼接到 URL 路径中，生产代码未对 model 做验证（验证仅在未启用的 EnhancedHTTPServer 中）。
- **证据**：
```go
url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.apiKey)
```
- **影响**：精心构造的 model 名称可操纵 URL 路径，配合被入侵的 admin 账户可实现 SSRF。
- **CVSS 3.1 分数**：7.2 (AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:N/A:N)
- **修复建议**：在 provider 层对 model 名称做 URL 编码或白名单验证。
- **参考**：CWE-918

---

### 2.11 [Medium] OAuth state 验证可绕过

- **位置**：`internal/identity/server/http.go:101-107`
- **描述**：当 `oauth_state` cookie 缺失或为空时，state 验证被跳过，削弱 CSRF 防护。
- **证据**：
```go
state := r.URL.Query().Get("state")
cookie, _ := r.Cookie("oauth_state")
if cookie != nil && cookie.Value != "" && cookie.Value != state {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
    return
}
```
- **影响**：攻击者可发起 OAuth CSRF 攻击，将攻击者的 OAuth 身份关联到受害者会话。
- **CVSS 3.1 分数**：6.5 (AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N)
- **修复建议**：state 验证必须为强制性，cookie 缺失时拒绝请求。
- **参考**：CWE-352

---

### 2.12 [Medium] IP 欺骗 — Rate Limiter 信任客户端头

- **位置**：`internal/pkg/middleware/ratelimit.go:212-233`
- **描述**：`getClientIP` 直接信任 `X-Forwarded-For` 和 `X-Real-IP` 头，攻击者可伪造 IP 绕过速率限制。
- **证据**：
```go
func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        if idx := indexOf(xff, ","); idx != -1 {
            return xff[:idx]
        }
        return xff
    }
```
- **影响**：攻击者可通过伪造 IP 头绕过所有 IP-based 速率限制。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N)
- **修复建议**：仅在存在受信任代理时解析这些头，使用 `TrustedProxies` 配置。
- **参考**：CWE-348

---

### 2.13 [Medium] LIKE 通配符注入

- **位置**：`internal/identity/data/data.go:355`，`internal/log/data/data.go:122`，`internal/channel/data/data.go:324`
- **描述**：用户输入直接嵌入 LIKE 模式，未转义 `%` 和 `_` 通配符。
- **证据**：
```go
query = query.Where("username LIKE ?", "%"+keyword+"%")
```
- **影响**：输入 `%` 可匹配所有记录，导致信息泄露。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：转义 LIKE 通配符：
```go
func escapeLike(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "%", "\\%")
    s = strings.ReplaceAll(s, "_", "\\_")
    return s
}
query = query.Where("username LIKE ?", "%"+escapeLike(keyword)+"%")
```
- **参考**：CWE-89

---

### 2.14 [Medium] 无登录/注册速率限制

- **位置**：`internal/identity/service/identity.go:82-111`
- **描述**：Login 和 Register gRPC 端点无速率限制，可被暴力破解。
- **影响**：攻击者可进行大规模密码猜测攻击。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：为 Login/Register 添加基于 IP 的速率限制。
- **参考**：CWE-307

---

### 2.15 [Medium] Redis 无认证

- **位置**：`internal/pkg/xdb/redis.go`，`deployments/docker-compose/docker-compose.yml:29-43`
- **描述**：Redis 客户端不传递密码，docker-compose 中 Redis 无 `--requirepass` 且暴露端口到主机。
- **影响**：同网络的任何客户端可连接 Redis 读写数据。
- **CVSS 3.1 分数**：5.9 (AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **修复建议**：Redis 启用密码认证，Docker Compose 中不暴露端口到主机。
- **参考**：CWE-306

---

### 2.16 [Medium] TLS 默认关闭

- **位置**：`.env.example:28`，`internal/pkg/tls/config.go:26`
- **描述**：`TLS_ENABLED=false` 是默认值，禁用 TLS 时 HTTP 客户端设置 `InsecureSkipVerify: true`。
- **影响**：默认部署所有通信为明文。
- **CVSS 3.1 分数**：5.9 (AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N)
- **修复建议**：生产部署默认启用 TLS。
- **参考**：CWE-319

---

### 2.17 [Medium] CSP 允许 unsafe-inline

- **位置**：`internal/pkg/middleware/security.go:29-30`
- **描述**：CSP 策略中 `script-src` 和 `style-src` 允许 `'unsafe-inline'`，削弱 XSS 防护。
- **证据**：
```go
"script-src 'self' 'unsafe-inline'; "+
"style-src 'self' 'unsafe-inline'; "+
```
- **影响**：降低 CSP 对 XSS 的防护效果。
- **CVSS 3.1 分数**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N)
- **修复建议**：使用 nonce 或 hash 替代 `unsafe-inline`。
- **参考**：CWE-79

---

### 2.18 [Medium] HSTS 仅在 TLS 连接时设置

- **位置**：`internal/pkg/middleware/security.go:37-39`
- **描述**：HSTS 头仅在 `r.TLS != nil` 时设置。当应用运行在 TLS 终止代理后方（K8s 常见场景），HSTS 永远不会被设置。
- **影响**：客户端可能通过 HTTP 访问，易受中间人攻击。
- **CVSS 3.1 分数**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N)
- **修复建议**：检查 `X-Forwarded-Proto` 头来判断是否为 HTTPS。
- **参考**：CWE-319

---

### 2.19 [Medium] OAuth state cookie 缺少 Secure 标志

- **位置**：`internal/identity/server/http.go:83-89`
- **描述**：`oauth_state` cookie 设置了 `HttpOnly` 和 `SameSite` 但缺少 `Secure: true`，cookie 会通过 HTTP 传输。
- **影响**：中间人可截获 state cookie。
- **CVSS 3.1 分 score**：4.3 (AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N)
- **修复建议**：添加 `Secure: true`。
- **参考**：CWE-614

---

### 2.20 [Medium] JSON 解码无大小限制

- **位置**：`internal/relay/server/json.go:9-14`
- **描述**：`decodeJSON` 使用 `io.ReadAll` 读取整个请求体到内存，生产代码未应用 MaxBodySize 中间件。
- **证据**：
```go
func decodeJSON(r io.Reader, v interface{}) error {
    data, err := io.ReadAll(r)
```
- **影响**：大请求体可导致内存耗尽（DoS）。
- **CVSS 3.1 分数**：5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L)
- **修复建议**：使用 `http.MaxBytesReader` 限制读取大小，或使用 `json.NewDecoder` 配合 `LimitReader`。
- **参考**：CWE-400

---

### 2.21 [Low] math/rand 用于负载均衡

- **位置**：`internal/pkg/registry/resolver.go:6,42`
- **描述**：使用 `math/rand` 而非 `crypto/rand` 进行服务实例选择。
- **影响**：可预测的负载均衡选择，实际安全影响极低。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：对安全性无实质影响，可保持现状或改用 `crypto/rand`。

---

### 2.22 [Low] Request ID 可预测

- **位置**：`internal/pkg/middleware/security.go:132-134`
- **描述**：Request ID 使用 `time.Now().UnixNano()` 生成，可预测。
- **影响**：可能被用于请求碰撞或时序攻击。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：使用 `crypto/rand` 生成 UUID。
- **参考**：CWE-330

---

### 2.23 [Low] Rate Limiter 使用弱哈希

- **位置**：`internal/pkg/middleware/ratelimit.go:237-243`
- **描述**：`simpleHash` 使用非加密哈希，可能导致不同 token 共享速率限制桶。
- **影响**：合法用户可能被错误限流。
- **CVSS 3.1 分数**：3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N)
- **修复建议**：使用 SHA256 替代。

---

### 2.24 [Low] Token 生成存在轻微模偏差

- **位置**：`internal/identity/biz/auth.go:271`
- **描述**：`int(b[i])%len(letters)` 由于 256 不能被 62 整除，引入轻微偏差。
- **影响**：实际影响可忽略，32 字符 token 熵足够。
- **CVSS 3.1 分数**：2.0 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：使用拒绝采样消除偏差。

---

### 2.25 [Low] Channel API Key 明文存储

- **位置**：`internal/channel/data/data.go:27`
- **描述**：上游 provider API key 以明文存储在数据库中。
- **影响**：数据库泄露时所有 API key 暴露。
- **CVSS 3.1 分数**：3.7 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N)
- **修复建议**：使用加密存储或密钥管理服务。
- **参考**：CWE-312

---

### 2.26 [Info] gRPC auth interceptor token 提取不完整

- **位置**：`internal/pkg/grpc/auth.go:203-205`
- **描述**：`extractTokenFromContext` 无法从 gRPC metadata 提取 JWT token，仅支持 mTLS。
- **影响**：JWT 认证拦截器实际无法工作。

---

### 2.27 [Info] Security CI/CD Actions 版本过旧

- **位置**：`.github/workflows/security.yml`
- **描述**：`actions/checkout@v3`、`actions/setup-go@v4` 等应升级到 v4+。
- **影响**：可能缺少安全修复。

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

- [ ] 移除所有硬编码凭证
- [ ] Admin API 添加认证中间件
- [ ] 将 EnhancedHTTPServer 中间件接入生产代码
- [ ] 默认启用 TLS
- [ ] gRPC 服务间通信使用 mTLS
- [ ] OAuth state 验证设为强制
- [ ] Login/Register 添加速率限制
- [ ] 所有 HTTP 错误返回通用消息
- [ ] Redis 启用密码认证
- [ ] Docker Compose 不暴露内部端口到主机
- [ ] LIKE 查询转义通配符
- [ ] JSON 解码添加大小限制
- [ ] CSP 移除 unsafe-inline
- [ ] HSTS 在代理后方也能正确设置
- [ ] OAuth state cookie 添加 Secure 标志
- [ ] Channel API key 加密存储
- [ ] Request ID 使用 UUID

---

## 5. 修复优先级路线图

### 立即修复（7 天内）

| # | 问题 | 严重性 | 文件 |
|---|---|---|---|
| 1 | Admin API 添加认证中间件 | Critical | `internal/admin/server/http.go` |
| 2 | 移除硬编码 JWT 默认密钥 | Critical | `internal/pkg/auth/jwt.go:35` |
| 3 | 移除硬编码 demo 用户/token | Critical | `internal/identity/data/data.go:82-103` |
| 4 | 将安全中间件接入生产代码 | High | `cmd/relay-gateway/wire_gen.go` |
| 5 | CreateUser 正确哈希密码 | High | `internal/identity/biz/auth.go:224` |
| 6 | 移除硬编码数据库凭证默认值 | High | `configs/*.yaml` |

### 短期修复（30 天内）

| # | 问题 | 严重性 |
|---|---|---|
| 7 | 默认启用 TLS | High |
| 8 | 错误信息脱敏 | High |
| 9 | Gemini URL 注入防护 | High |
| 10 | OAuth 用户默认不给无限配额 | High |
| 11 | OAuth state 验证强制化 | Medium |
| 12 | Rate Limiter IP 提取安全化 | Medium |
| 13 | LIKE 通配符转义 | Medium |
| 14 | Login/Register 速率限制 | Medium |
| 15 | Redis 认证 | Medium |

### 中长期改进

| # | 问题 | 严重性 |
|---|---|---|
| 16 | 引入密钥管理服务 | Medium |
| 17 | CSP 使用 nonce 替代 unsafe-inline | Medium |
| 18 | Channel API key 加密存储 | Low |
| 19 | Request ID 使用 UUID | Low |
| 20 | 添加 DAST 扫描到 CI | Info |
| 21 | GitHub Actions 版本升级 | Info |
