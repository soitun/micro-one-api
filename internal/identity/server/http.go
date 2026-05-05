package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"micro-one-api/internal/identity/biz"
	"micro-one-api/internal/pkg/metrics"
	"micro-one-api/internal/pkg/oauth"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer wires HTTP transport for identity-service.
func NewHTTPServer(addr string, uc *biz.IdentityUsecase, oauthRegistry *oauth.ProviderRegistry) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)

	// OAuth endpoints
	if oauthRegistry != nil {
		srv.HandleFunc("/v1/oauth/providers", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"providers": oauthRegistry.Names(),
			})
		})
		srv.HandleFunc("/v1/oauth/", func(w http.ResponseWriter, r *http.Request) {
			handleOAuth(w, r, oauthRegistry, uc)
		})
	}

	srv.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	return srv
}

// handleOAuth routes /v1/oauth/{provider}/{action} requests.
func handleOAuth(w http.ResponseWriter, r *http.Request, registry *oauth.ProviderRegistry, uc *biz.IdentityUsecase) {
	// Parse: /v1/oauth/{provider}/{action}
	path := r.URL.Path[len("/v1/oauth/"):]
	// Find first slash
	idx := -1
	for i, c := range path {
		if c == '/' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		http.Error(w, `{"error":"invalid oauth path"}`, http.StatusBadRequest)
		return
	}
	providerName := path[:idx]
	action := path[idx+1:]

	provider, ok := registry.Get(providerName)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown provider: " + providerName})
		return
	}

	switch action {
	case "authorize":
		handleOAuthAuthorize(w, r, provider)
	case "callback":
		handleOAuthCallback(w, r, provider, uc)
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
	}
}

func handleOAuthAuthorize(w http.ResponseWriter, r *http.Request, provider oauth.Provider) {
	state := generateState()
	// In production, store state in session/cookie for CSRF validation
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, provider.AuthURL(state), http.StatusFound)
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, provider oauth.Provider, uc *biz.IdentityUsecase) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code"})
		return
	}

	// Validate state (optional but recommended)
	state := r.URL.Query().Get("state")
	cookie, _ := r.Cookie("oauth_state")
	if cookie != nil && cookie.Value != "" && cookie.Value != state {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
		return
	}

	userInfo, err := provider.Exchange(r.Context(), code)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	user, token, err := uc.OAuthLogin(r.Context(), userInfo.Provider, userInfo.ProviderID, userInfo.Username, userInfo.Email, userInfo.DisplayName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"user_id": user.ID,
		"username": user.Username,
	})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
