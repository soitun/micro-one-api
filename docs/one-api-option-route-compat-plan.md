# One-API Option Route Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add One API compatible `/api/option/` GET/PUT routes.

**Architecture:** Reuse existing `AdminService.GetSystemOptions` and `UpdateSystemOptions`. Add a small HTTP adapter that maps One API flat JSON to `commonv1.SystemOptions` and wraps responses in `success/message/data`.

**Tech Stack:** Go, Kratos HTTP transport, standard `net/http/httptest` tests.

---

## Files

- Modify: `internal/admin/server/http.go`
  - Register `/api/option/`.
  - Add One API option handler.
- Modify: `internal/admin/server/http_test.go`
  - Add tests for auth, GET, and PUT.
- Modify: `docs/one-api-full-gap-analysis-20260509.md`
  - Mark option route compatibility as completed.

## Task 1: HTTP Tests

**Files:**
- Modify: `internal/admin/server/http_test.go`

- [ ] **Step 1: Write failing tests**

Add:

```go
func TestAdminHTTPOptionRequiresAuth(t *testing.T)
func TestAdminHTTPOptionGetReturnsOneAPIShape(t *testing.T)
func TestAdminHTTPOptionPutAcceptsFlatBody(t *testing.T)
```

- [ ] **Step 2: Run tests to verify RED**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPOption' -count=1
```

Expected: FAIL because `/api/option/` is not registered.

## Task 2: Implementation

**Files:**
- Modify: `internal/admin/server/http.go`

- [ ] **Step 1: Register route**

Add admin-auth protected `/api/option/`.

- [ ] **Step 2: Add GET adapter**

Call `svc.GetSystemOptions`, return:

```json
{"success":true,"message":"","data":{"site_title":"...","registration_enabled":true}}
```

- [ ] **Step 3: Add PUT adapter**

Accept flat JSON and protobuf-style `{options:{...}}`, call `svc.UpdateSystemOptions`, and return One API style response.

- [ ] **Step 4: Run tests to verify GREEN**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPOption' -count=1
```

Expected: PASS.

## Task 3: Docs And Verification

**Files:**
- Modify: `docs/one-api-full-gap-analysis-20260509.md`

- [ ] **Step 1: Update gap analysis**

Move `/api/option/` into completed branch work. Keep full option key migration as a remaining gap.

- [ ] **Step 2: Run verification**

Run:

```bash
go test ./internal/admin/server -run 'TestAdminHTTPOption' -count=1
go test ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add internal/admin/server/http.go internal/admin/server/http_test.go docs/one-api-full-gap-analysis-20260509.md docs/one-api-option-route-compat-design.md docs/one-api-option-route-compat-plan.md
git commit -m "feat: add one-api option route compatibility"
```
