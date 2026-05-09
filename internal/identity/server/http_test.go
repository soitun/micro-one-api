package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"micro-one-api/internal/identity/biz"
	identitydata "micro-one-api/internal/identity/data"
)

func TestIdentityHTTPRegisterLoginAndSelf(t *testing.T) {
	repo := identitydata.NewMemoryRepositoryForTest()
	uc := biz.NewIdentityUsecase(repo)
	srv := NewHTTPServer(":0", uc, nil)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"username":"alice","password":"password123","email":"alice@example.com"}`))
	registerRec := httptest.NewRecorder()
	srv.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status = %d, body=%s", registerRec.Code, registerRec.Body.String())
	}
	if !strings.Contains(registerRec.Body.String(), `"success":true`) {
		t.Fatalf("register failed: %s", registerRec.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"username":"alice","password":"password123"}`))
	loginRec := httptest.NewRecorder()
	srv.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", loginRec.Code, loginRec.Body.String())
	}
	body := loginRec.Body.String()
	if !strings.Contains(body, `"token"`) {
		t.Fatalf("login response missing token: %s", body)
	}

	token := extractJSONField(body, "token")
	selfReq := httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	selfReq.Header.Set("Authorization", "Bearer "+token)
	selfRec := httptest.NewRecorder()
	srv.ServeHTTP(selfRec, selfReq)
	if selfRec.Code != http.StatusOK {
		t.Fatalf("self status = %d, body=%s", selfRec.Code, selfRec.Body.String())
	}
	if !strings.Contains(selfRec.Body.String(), `"username":"alice"`) {
		t.Fatalf("self response mismatch: %s", selfRec.Body.String())
	}
}

func TestIdentityHTTPTokenCRUD(t *testing.T) {
	repo := identitydata.NewMemoryRepositoryForTest()
	uc := biz.NewIdentityUsecase(repo)
	user, err := uc.Register(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "alice", "password123", "alice@example.com", "default")
	if err != nil {
		t.Fatal(err)
	}
	loginUser, authToken, err := uc.Login(httptest.NewRequest(http.MethodGet, "/", nil).Context(), user.Username, "password123")
	if err != nil || loginUser.ID != user.ID {
		t.Fatalf("login error = %v", err)
	}
	srv := NewHTTPServer(":0", uc, nil)

	createReq := httptest.NewRequest(http.MethodPost, "/api/token/", strings.NewReader(`{"name":"test-token","models":["gpt-4o-mini"]}`))
	createReq.Header.Set("Authorization", "Bearer "+authToken)
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create token status = %d, body=%s", createRec.Code, createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `"key"`) {
		t.Fatalf("create token response missing key: %s", createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/token/", nil)
	listReq.Header.Set("Authorization", "Bearer "+authToken)
	listRec := httptest.NewRecorder()
	srv.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list token status = %d, body=%s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"total":2`) {
		t.Fatalf("list token response mismatch: %s", listRec.Body.String())
	}
}

func extractJSONField(body, key string) string {
	prefix := `"` + key + `":"`
	idx := strings.Index(body, prefix)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(prefix):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}
