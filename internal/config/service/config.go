package service

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	"micro-one-api/internal/config/biz"
)

// ConfigService is the transport layer entry for config-service.
type ConfigService struct {
	uc *biz.ConfigUsecase
}

func NewConfigService(uc *biz.ConfigUsecase) *ConfigService {
	return &ConfigService{uc: uc}
}

// GetConfig handles GET /v1/configs/{namespace}/{key}
func (s *ConfigService) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	namespace, key := parseTwoSegments(r.URL.Path, "/v1/configs/")
	if namespace == "" || key == "" {
		writeError(w, http.StatusBadRequest, "namespace and key are required")
		return
	}

	entry, err := s.uc.GetConfig(r.Context(), namespace, key)
	if err != nil {
		if err == biz.ErrConfigNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, configEntryToMap(entry))
}

// ListConfigs handles GET /v1/configs/{namespace}
func (s *ConfigService) ListConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	namespace := strings.TrimPrefix(r.URL.Path, "/v1/configs/")
	namespace = strings.TrimRight(namespace, "/")
	if namespace == "" {
		writeError(w, http.StatusBadRequest, "namespace is required")
		return
	}

	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 32)

	entries, total, err := s.uc.ListConfigs(r.Context(), namespace, int32(page), int32(pageSize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		items = append(items, configEntryToMap(e))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// SetConfig handles PUT /v1/configs/{namespace}/{key}
func (s *ConfigService) SetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	namespace, key := parseTwoSegments(r.URL.Path, "/v1/configs/")
	if namespace == "" || key == "" {
		writeError(w, http.StatusBadRequest, "namespace and key are required")
		return
	}

	var body struct {
		Value   string `json:"value"`
		Comment string `json:"comment"`
	}
	if err := sonic.ConfigStd.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.uc.SetConfig(r.Context(), namespace, key, body.Value, body.Comment); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteConfig handles DELETE /v1/configs/{namespace}/{key}
func (s *ConfigService) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	namespace, key := parseTwoSegments(r.URL.Path, "/v1/configs/")
	if namespace == "" || key == "" {
		writeError(w, http.StatusBadRequest, "namespace and key are required")
		return
	}

	if err := s.uc.DeleteConfig(r.Context(), namespace, key); err != nil {
		if err == biz.ErrConfigNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// parseTwoSegments extracts two path segments after a prefix.
// e.g. parseTwoSegments("/v1/configs/ns1/key1", "/v1/configs/") => ("ns1", "key1")
func parseTwoSegments(path, prefix string) (string, string) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimRight(rest, "/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func configEntryToMap(e *biz.ConfigEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":         e.ID,
		"namespace":  e.Namespace,
		"key":        e.Key,
		"value":      e.Value,
		"comment":    e.Comment,
		"updated_at": e.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	sonic.ConfigStd.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	sonic.ConfigStd.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}
