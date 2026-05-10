# One-API User Billing Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add One API compatible user billing HTTP endpoints for dashboard and redeem-code top-up.

**Architecture:** Identity HTTP server owns `/api/user/*` compatibility routes and authenticates Bearer tokens through `IdentityUsecase`. Dashboard and top-up delegate account data to an optional `billingv1.BillingServiceClient`, preserving existing constructor call sites through a variadic parameter.

**Tech Stack:** Go, Kratos HTTP transport, gRPC generated billing client, standard `net/http/httptest` tests.

---

## Files

- Modify: `internal/identity/server/http.go`
  - Add optional billing client dependency to `NewHTTPServer`.
  - Register `/api/user/dashboard` and `/api/user/topup`.
  - Implement dashboard/topup handlers with One API style `apiResponse`.
- Modify: `internal/identity/server/http_test.go`
  - Add fake billing client.
  - Add focused HTTP tests for auth, missing billing dependency, success, validation, and billing failures.
- Modify: `docs/one-api-full-gap-analysis-20260509.md`
  - Move dashboard/topup from remaining user API gaps to completed user billing compatibility.

## Task 1: Dashboard Endpoint

**Files:**
- Modify: `internal/identity/server/http_test.go`
- Modify: `internal/identity/server/http.go`

- [ ] **Step 1: Write failing tests**

Add tests:

```go
func TestIdentityHTTPDashboardRequiresAuth(t *testing.T)
func TestIdentityHTTPDashboardRequiresBillingClient(t *testing.T)
func TestIdentityHTTPDashboardReturnsAccountSnapshot(t *testing.T)
```

The success test should register and login a user, inject a fake billing client returning `commonv1.AccountSnapshot`, call `GET /api/user/dashboard`, and assert:

- HTTP 200
- `"success":true`
- `"quota":1000`
- `"used_quota":100`
- `"request_count":10`
- `"group":"default"`
- `"group_ratio":1`
- `"frozen_quota":0`

- [ ] **Step 2: Run dashboard tests to verify RED**

Run:

```bash
go test ./internal/identity/server -run 'TestIdentityHTTPDashboard' -count=1
```

Expected: FAIL because `/api/user/dashboard` is not registered and/or `NewHTTPServer` does not accept the billing client argument.

- [ ] **Step 3: Implement minimal dashboard support**

In `internal/identity/server/http.go`:

- Import `billingv1`.
- Change constructor to:

```go
func NewHTTPServer(addr string, uc *biz.IdentityUsecase, oauthRegistry *oauth.ProviderRegistry, billingClients ...billingv1.BillingServiceClient) *khttp.Server
```

- Select the first billing client if present.
- Register `GET /api/user/dashboard`.
- Authenticate with `authSnapshotFromRequest`.
- Return 503 if no billing client is configured.
- Call `GetAccountSnapshot` with `strconv.FormatInt(snapshot.UserID, 10)`.
- Return snapshot fields in One API style JSON.

- [ ] **Step 4: Run dashboard tests to verify GREEN**

Run:

```bash
go test ./internal/identity/server -run 'TestIdentityHTTPDashboard' -count=1
```

Expected: PASS.

## Task 2: TopUp Endpoint

**Files:**
- Modify: `internal/identity/server/http_test.go`
- Modify: `internal/identity/server/http.go`

- [ ] **Step 1: Write failing tests**

Add tests:

```go
func TestIdentityHTTPTopUpRequiresAuth(t *testing.T)
func TestIdentityHTTPTopUpRejectsEmptyKey(t *testing.T)
func TestIdentityHTTPTopUpReturnsRedeemedAmount(t *testing.T)
func TestIdentityHTTPTopUpReturnsBillingFailure(t *testing.T)
```

Success test should inject fake billing client, call `POST /api/user/topup` with `{"key":"CODE-1000"}`, and assert:

- HTTP 200
- `"success":true`
- `"data":1000`
- fake billing client observed code `CODE-1000`

Failure test should return `RedeemCodeResponse{Success:false, ErrorMessage:"invalid code"}` and assert HTTP 200 with `"success":false`.

- [ ] **Step 2: Run topup tests to verify RED**

Run:

```bash
go test ./internal/identity/server -run 'TestIdentityHTTPTopUp' -count=1
```

Expected: FAIL because `/api/user/topup` is not registered.

- [ ] **Step 3: Implement minimal topup support**

In `internal/identity/server/http.go`:

- Register `POST /api/user/topup`.
- Authenticate with `authSnapshotFromRequest`.
- Return 503 if no billing client is configured.
- Decode JSON body with `key`.
- Return HTTP 200 and `success=false` if key is empty.
- Call `RedeemCode` with authenticated user ID and key.
- Return amount on success.
- Return `success=false` for billing errors and failed billing responses.

- [ ] **Step 4: Run topup tests to verify GREEN**

Run:

```bash
go test ./internal/identity/server -run 'TestIdentityHTTPTopUp' -count=1
```

Expected: PASS.

## Task 3: Documentation And Full Verification

**Files:**
- Modify: `docs/one-api-full-gap-analysis-20260509.md`

- [ ] **Step 1: Update gap analysis**

Document that this branch adds:

- `/api/user/dashboard`
- `/api/user/topup`

Keep a note that One API's richer per-day/per-model dashboard chart remains a later usage-log aggregation gap.

- [ ] **Step 2: Run focused server tests**

Run:

```bash
go test ./internal/identity/server -count=1
```

Expected: PASS.

- [ ] **Step 3: Run repository verification**

Run:

```bash
go test ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

Run:

```bash
git add internal/identity/server/http.go internal/identity/server/http_test.go docs/one-api-full-gap-analysis-20260509.md docs/one-api-user-billing-compat-plan.md
git commit -m "feat: add one-api user billing endpoints"
```
