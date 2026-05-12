# One API Gap Priority List

> Branch: `feature/one-api-gap-phase1`
> Date: 2026-05-12
> Source: `docs/one-api-full-gap-analysis-20260509.md`, current `micro-one-api` implementation, and sibling `../one-api`.

## Summary

The project already has the microservice skeleton, OpenAI-compatible relay path, token validation, channel selection, billing reservation/commit flow, basic admin APIs, and partial One API compatibility routes.

It is not yet a full One API product. The largest remaining gaps are the full web experience, One API-compatible admin/user API semantics, business usage logs, dashboard usage aggregation, system options, token/channel field parity, and provider-specific adapters.

## Priority 0: Product Usability

These gaps block a One API-like product experience.

| Area | Current State | Needed Work |
| --- | --- | --- |
| Web frontend | Only a lightweight embedded admin HTML exists. | Build or migrate a real user/admin frontend covering login, user self-service, tokens, channels, redemptions, logs, settings, and dashboard views. |
| Business usage logs | `log-service` stores generic logs: level, message, source, request ID, user ID. | Add One API-style consume logs: model, token name, quota, prompt tokens, completion tokens, channel ID, elapsed time, stream flag, username. |
| User dashboard | `/api/user/dashboard` returns billing account snapshot only. | Add per-day/per-model usage aggregation for dashboard charts. |
| Admin API compatibility | Many `/api/*` routes exist only partially or via `/v1/*` alternatives. | Align common One API admin routes and response shapes for users, channels, logs, tokens, redemptions, and options. |

## Priority 1: One API Compatibility

These gaps affect frontend compatibility and operational behavior.

| Area | Current State | Needed Work |
| --- | --- | --- |
| OAuth/SSO | GitHub, Google, and generic `/v1/oauth/*` exist. | Add One API-compatible OIDC, Lark, WeChat, OAuth state, and bind flows. |
| Token fields | CRUD exists, but fields are simplified. | Align `accessed_time`, `used_quota`, `subnet`, `unlimited_quota`, remaining quota, expiration, exhausted status, and response shape. |
| Channel fields | Core fields exist. | Add weight, test time, response time, balance, balance update time, used quota, model mapping, and system prompt. |
| Channel balance | `/api/channel/update_balance` returns not implemented. | Add provider-specific balance adapters and persistence. |
| System options | Only a small subset is mapped. | Add One API option keys for auth, registration, SMTP, Turnstile, ratios, themes, notices, links, retry settings, and display flags. |

## Priority 2: Relay and Provider Depth

These gaps affect API surface parity and upstream provider coverage.

| Area | Current State | Needed Work |
| --- | --- | --- |
| OpenAI route surface | Chat, completions, embeddings, image generation, audio, moderation, models, and proxy are registered. | Add compatibility routes for edits, engines embeddings, files, fine-tuning, assistants, and threads. Unsupported routes can initially return stable NotImplemented responses. |
| Dashboard billing API | Missing `/dashboard/billing/*` and `/v1/dashboard/billing/*`. | Add subscription and usage endpoints compatible with OpenAI dashboard-style clients. |
| Provider adapters | Anthropic and Gemini have dedicated adapters; many providers use OpenAI-compatible forwarding. | Gradually add provider-specific adapters for Azure details, Baidu, Ali, Xunfei, Tencent, Zhipu, Volcano/Doubao, Ollama, Replicate, Cloudflare, VertexAI, OpenRouter, SiliconFlow, and others. |

## Recommended Execution Order

1. Add One API-style business usage logs and dashboard aggregation.
2. Align `/api/log/*`, `/api/user/dashboard`, and dashboard billing usage endpoints.
3. Expand token, channel, and option fields to match One API frontend expectations.
4. Fill admin/user route compatibility gaps.
5. Build or migrate the full web frontend.
6. Add provider-specific adapters based on actual channel demand.

## Phase 1 Scope for This Branch

This branch starts with the first P0 item:

1. Add structured usage log persistence fields.
2. Record successful relay usage into `log-service`.
3. Add per-day/per-model usage aggregation for authenticated users.
4. Expose One API-compatible dashboard usage data.
5. Keep changes narrow and test-driven.
