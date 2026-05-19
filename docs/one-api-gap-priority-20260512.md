# One API Remaining Gap Priority List

> Branch: `docs/one-api-gap-refresh-20260512`
> Date: 2026-05-12 (main table refreshed 2026-05-19)
> Source: current `develop` code and sibling `../one-api`.

## Summary

The project now covers the core microservice skeleton, OpenAI-compatible relay path, token validation, channel selection, billing reservation/commit/release flow, structured usage logs, user dashboard aggregation (usage + subscription), expanded token/channel/option fields, OAuth/SSO and bind flows for GitHub/Google/OIDC/Lark/WeChat/Telegram with Turnstile and email-domain enforcement, channel balance refresh adapters for the OpenAI-compatible providers, group and content management, and a wide NotImplemented-stable OpenAI route surface.

It is still not a full One API product. The largest remaining gaps are the full web frontend, native adapters for non-OpenAI-compatible providers, real top-up / affiliate / online payment implementations (currently disabled placeholders), and explicit failure/disable semantics for channel balance refresh on uncovered providers.

## Recently Completed (since 2026-05-12)

These items from the earlier priority list are now implemented:

| Area | Current State |
| --- | --- |
| OAuth/SSO and anti-abuse | GitHub/Google/OIDC/Lark/WeChat/Telegram login + bind, `/api/oauth/state`, Turnstile verification, email-domain whitelist all wired up; one user can hold multiple OAuth identities. |
| Channel balance refresh | `/api/channel/update_balance` and `/api/channel/update_balance/{id}` refresh OpenAI, DeepSeek, OpenRouter, SiliconFlow channels via `balanceAdapterForChannel`; result persisted to `balance` and `balance_updated_time`. |
| Dashboard billing subscription | `/dashboard/billing/subscription` and `/v1/dashboard/billing/subscription` return stable subscription objects. |
| OpenAI route surface | edits, engines embeddings, files, fine_tuning (incl. graders), assistants, threads, batches, images edits/variations, audio, moderations, vector, eval, containers — all return stable NotImplemented OpenAI error payloads. |
| Group management | `/api/group` supports GET/POST/PUT/DELETE with optional `with_ratio`. |
| Content management | `/api/notice`, `/api/about`, `/api/home_page_content` accept GET and authenticated PUT. |
| Log deletion | `log-service` exposes `DeleteLogs` with mandatory `end_time`; admin-api proxies via `/api/log` DELETE when `LOG_HTTP_ENDPOINT` + `SERVICE_TOKEN` are configured. |
| Azure provider details | `azure.go` accepts `APIVersion` config; factory rejects empty `base_url`. |
| Provider catalog metadata | `/api/models` returns provider name, default base URL, required config fields, adapter state, and OpenAI-compatible/native flags; native-only providers are explicitly marked. |

## Priority 0: Product Usability

| Area | Current State | Needed Work |
| --- | --- | --- |
| Web frontend | Single embedded `admin.html` (~747 lines) only. | Build or migrate a real user/admin frontend covering login, user self-service, tokens, channels, redemptions, logs, settings, dashboard charts, content, groups, and OAuth/bind flows. |
| Top-up / affiliate / online payment | `/api/topup`, `/api/aff/transfer`, and `/api/pay/*` exist only as disabled placeholders returning stable error payloads. | Implement real quota top-up via redemption code + admin grant flow, affiliate reward transfer with audit, and at least one online-payment integration (or an explicit "self-hosted only" stance). |

## Priority 1: Compatibility Depth

| Area | Current State | Needed Work |
| --- | --- | --- |
| Channel balance refresh — uncovered providers | Refresh works for OpenAI / DeepSeek / OpenRouter / SiliconFlow; other providers return an explicit "balance refresh not supported" error without disabling the channel. | Define explicit failure semantics: stay-enabled-but-stale vs. auto-disable on persistent failure, and document which providers are intentionally unsupported. |
| Log deletion deployment dependency | Admin-api log delete depends on `LOG_HTTP_ENDPOINT` + `SERVICE_TOKEN`; missing env returns NotImplemented at runtime. | Surface this prerequisite in `deployment.md` and add a config-validation warning on admin-api startup. |
| Provider model defaults | Catalog metadata is stable; per-provider model lists are conservative defaults. | Expand provider default model lists where real-world traffic demands, driven by channel telemetry rather than upstream catalog crawls. |

## Priority 2: Provider and Relay Depth

| Area | Current State | Needed Work |
| --- | --- | --- |
| Provider-native adapters | Anthropic, Gemini, Azure, and the OpenAI-compatible family have adapters; eight providers explicitly return `requires a native provider adapter`: Hunyuan, Xingchen, Bedrock, Cloudflare, VertexAI, Replicate, Baidu, Xunfei. | Add native adapters in demand order. Each adapter must cover request conversion, response conversion, streaming, usage extraction, and error mapping. |
| Reconciliation surface for admin | `ReconciliationUsecase` runs on schedule but lacks an admin-facing review UI/API beyond raw `/v1/reconciliation`. | Add an admin endpoint that exposes recent reconciliation runs and discrepancies, gated behind admin auth. |

## Disabled Placeholder Routes

These routes are intentionally registered as stable disabled placeholders, distinct from NotImplemented OpenAI compatibility shims. They MUST return a stable shape and SHOULD NOT be confused with "not yet implemented" — the product decision is to keep them off until the corresponding subsystem is built.

| Route | Purpose | Required Before Enabling |
| --- | --- | --- |
| `/api/topup` | Manual / admin quota top-up | Top-up workflow implementation |
| `/api/aff/transfer` | Affiliate reward transfer | Affiliate ledger + audit |
| `/api/pay/*` | Online payment callbacks | Payment provider integration + idempotency |
| `/api/oauth/telegram/*` | Telegram OAuth login/bind | Telegram bot config and CSRF model |

## Completion Plan

Each remaining item should land as a small branch with route-level tests first, then implementation.

### 1. Web Frontend

Goal: replace the embedded admin HTML with a usable One API-style product UI.

Scope:
- Add login/logout and session/token handling against `/api/user/login`, `/api/user/logout`, and `/api/user/self`.
- Add user pages for dashboard charts, token CRUD, top-up, invitation code, and available models.
- Add admin pages for users, channels, redemptions, logs, options, status, content, and groups.
- Keep the frontend aligned to existing `/api/*` response shapes before introducing new backend routes.

Acceptance:
- A user can register/login, create and manage tokens, view dashboard usage, redeem quota, and manage their profile.
- An admin can manage users, channels, redemptions, logs, and options from the UI.
- Browser smoke tests cover the primary user and admin workflows.

### 2. Top-up / Affiliate / Online Payment

Goal: turn the disabled placeholder routes into real workflows.

Scope:
- `/api/topup`: admin-granted quota and redemption-code flow; idempotent against double-spend.
- `/api/aff/transfer`: ledger entries with audit, capped by invitation-reward configuration.
- `/api/pay/*`: at least one provider integration (Stripe or Alipay) with signed callback verification, or an explicit decision to keep disabled and remove the placeholders.

Acceptance:
- Each flow has positive, idempotent-replay, and unauthorized-caller tests.
- Audit entries are visible through admin log routes.

### 3. Balance Refresh Failure Semantics

Goal: turn "unsupported provider" silence into a documented policy.

Scope:
- Decide and document: unsupported providers keep the channel enabled with stale balance.
- Persistent fetch failures on supported providers count toward the existing channel disable threshold; otherwise leave channel state untouched.
- Surface last-error and last-success timestamps in the channel response.

Acceptance:
- Tests cover unsupported provider, transient upstream error, persistent upstream error, and stale-balance display.

### 4. Provider-Native Adapters

Goal: improve quality and reliability for non-OpenAI-compatible upstreams.

Scope:
- Add native adapters in demand order for Baidu, Hunyuan, Xunfei, Cloudflare, VertexAI, Bedrock, Replicate, Xingchen.
- For each adapter, cover request conversion, response conversion, streaming, usage extraction, and error mapping.

Acceptance:
- Each adapter has non-streaming, streaming, usage, and upstream-error tests.
- Provider defaults include base URL, supported models, and required channel config fields.

### 5. Reconciliation Review Surface

Goal: make the existing reconciliation job operationally useful from admin.

Scope:
- Admin endpoint listing recent reconciliation runs and per-user discrepancies.
- Drill-down to the underlying ledger entries.

Acceptance:
- Admin auth required; non-admin returns existing error shape.
- Tests cover empty, mixed, and discrepancy-only runs.

## Recommended Execution Order

1. Build or migrate the full web frontend against the current `/api/*` compatibility layer.
2. Implement top-up, affiliate transfer, and at least one online-payment path (or formally retire the placeholders).
3. Define and ship balance-refresh failure semantics for uncovered providers.
4. Add provider-native adapters in demand order, starting with the highest-traffic non-OpenAI-compatible channels.
5. Expose the reconciliation review surface to admins.

## Documentation Policy

Completed one-off design and implementation plan documents have been moved to `docs/archive/`. This file is the current priority source for remaining One API gaps. Architecture and deployment documents remain as reference material in `docs/`.
