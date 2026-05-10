# One-API Redemption Route Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add One API compatible `/api/redemption/*` routes backed by existing redeem-code admin service methods.

**Architecture:** Add admin-api route aliases that delegate to the existing `/v1/redeem-codes*` handlers. Interpret the One API `:id` path segment as redeem code string in this codebase.

**Tech Stack:** Go, Kratos HTTP transport, standard `net/http/httptest` tests.

---

## Files

- Modify: `internal/admin/server/http.go`
  - Register `/api/redemption/` and prefix route.
  - Add thin path adapters.
- Modify: `internal/admin/server/http_test.go`
  - Add route compatibility tests.
- Modify: `docs/one-api-full-gap-analysis-20260509.md`
  - Mark redemption route compatibility as completed.

## Task 1: HTTP Tests

**Files:**
- Modify: `internal/admin/server/http_test.go`

- [ ] **Step 1: Write failing tests**

Add:

```go
func TestAdminHTTPRedemptionRequiresAuth(t *testing.T)
func TestAdminHTTPRedemptionListRoute(t *testing.T)
func TestAdminHTTPRedemptionSearchRoute(t *testing.T)
func TestAdminHTTPRedemptionDeleteRoute(t *testing.T)
```

- [ ] **Step 2: Run tests to verify RED**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPRedemption' -count=1
```

Expected: FAIL because `/api/redemption/*` is not registered.

## Task 2: Implementation

**Files:**
- Modify: `internal/admin/server/http.go`

- [ ] **Step 1: Register routes**

Add admin-auth protected handlers:

- `/api/redemption/`
- `/api/redemption`
- prefix `/api/redemption/`

- [ ] **Step 2: Delegate list/create/update/search**

For exact `/api/redemption/` and `/api/redemption`, delegate to `handleRedeemCodes`.

- [ ] **Step 3: Delegate path get/delete/update**

For `/api/redemption/{code}`, adapt the path to existing `handleRedeemCodeByCode` logic.

- [ ] **Step 4: Run tests to verify GREEN**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPRedemption' -count=1
```

Expected: PASS.

## Task 3: Docs And Verification

**Files:**
- Modify: `docs/one-api-full-gap-analysis-20260509.md`

- [ ] **Step 1: Update gap analysis**

Move `/api/redemption/*` route compatibility into completed branch work. Keep export/batch UX as remaining gaps if needed.

- [ ] **Step 2: Run verification**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPRedemption' -count=1
go test ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add internal/admin/server/http.go internal/admin/server/http_test.go docs/one-api-full-gap-analysis-20260509.md docs/one-api-redemption-route-compat-design.md docs/one-api-redemption-route-compat-plan.md
git commit -m "feat: add one-api redemption route compatibility"
```
