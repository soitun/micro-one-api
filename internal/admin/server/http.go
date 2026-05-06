package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	adminv1 "micro-one-api/api/admin/v1"
	"micro-one-api/internal/admin/service"
	"micro-one-api/internal/pkg/metrics"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer wires HTTP transport for admin-api.
func NewHTTPServer(addr string, svc *service.AdminService) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)

	// Health and metrics
	srv.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// User management
	srv.HandleFunc("/v1/users", func(w http.ResponseWriter, r *http.Request) {
		handleListUsers(w, r, svc)
	})

	// Channel management
	srv.HandleFunc("/v1/channels", func(w http.ResponseWriter, r *http.Request) {
		handleListChannels(w, r, svc)
	})

	// System options
	srv.HandleFunc("/v1/system/options", func(w http.ResponseWriter, r *http.Request) {
		handleGetSystemOptions(w, r, svc)
	})

	// Logs
	srv.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		handleListLogs(w, r, svc)
	})

	// Account
	srv.HandleFunc("/v1/account", func(w http.ResponseWriter, r *http.Request) {
		handleGetAccount(w, r, svc)
	})

	// Redeem codes
	srv.HandleFunc("/v1/redeem-codes", func(w http.ResponseWriter, r *http.Request) {
		handleListRedeemCodes(w, r, svc)
	})

	return srv
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func getQueryInt32(r *http.Request, key string, defaultVal int32) int32 {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultVal
	}
	return int32(v)
}

func getQueryInt64(r *http.Request, key string, defaultVal int64) int64 {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

func handleListUsers(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page := getQueryInt32(r, "page", 1)
	pageSize := getQueryInt32(r, "page_size", 20)
	keyword := r.URL.Query().Get("keyword")
	group := r.URL.Query().Get("group")
	status := getQueryInt32(r, "status", 0)

	resp, err := svc.ListUsers(r.Context(), &adminv1.AdminListUsersRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Group:    group,
		Status:   status,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleListChannels(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page := getQueryInt32(r, "page", 1)
	pageSize := getQueryInt32(r, "page_size", 20)
	keyword := r.URL.Query().Get("keyword")
	group := r.URL.Query().Get("group")
	status := getQueryInt32(r, "status", 0)
	chType := getQueryInt32(r, "type", 0)

	resp, err := svc.ListChannels(r.Context(), &adminv1.AdminListChannelsRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Group:    group,
		Status:   status,
		Type:     chType,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleGetSystemOptions(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	switch r.Method {
	case http.MethodGet:
		resp, err := svc.GetSystemOptions(r.Context(), &adminv1.GetSystemOptionsRequest{})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPut:
		var req adminv1.UpdateSystemOptionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		resp, err := svc.UpdateSystemOptions(r.Context(), &req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func handleListLogs(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page := getQueryInt32(r, "page", 1)
	pageSize := getQueryInt32(r, "page_size", 20)
	userID := r.URL.Query().Get("user_id")
	logType := r.URL.Query().Get("type")
	startTime := getQueryInt64(r, "start_time", 0)
	endTime := getQueryInt64(r, "end_time", 0)

	resp, err := svc.ListLogs(r.Context(), &adminv1.ListLogsRequest{
		Page:      page,
		PageSize:  pageSize,
		UserId:    userID,
		Type:      logType,
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleGetAccount(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}

	resp, err := svc.GetAccountSnapshot(r.Context(), &adminv1.GetAccountSnapshotRequest{
		UserId: userID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleListRedeemCodes(w http.ResponseWriter, r *http.Request, svc *service.AdminService) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	page := getQueryInt32(r, "page", 1)
	pageSize := getQueryInt32(r, "page_size", 20)

	resp, err := svc.ListRedeemCodes(r.Context(), &adminv1.ListRedeemCodesRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
