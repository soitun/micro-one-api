package service

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	"micro-one-api/internal/log/biz"
)

// LogService is the transport layer entry for log-service.
type LogService struct {
	uc *biz.LogUsecase
}

func NewLogService(uc *biz.LogUsecase) *LogService {
	return &LogService{uc: uc}
}

// GetLog handles GET /v1/logs/{id}
func (s *LogService) GetLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/v1/logs/")
	idStr = strings.TrimRight(idStr, "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid log id")
		return
	}

	entry, err := s.uc.GetLog(r.Context(), id)
	if err != nil {
		if err == biz.ErrLogNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, logEntryToMap(entry))
}

// ListLogs handles GET /v1/logs?page=&page_size=&user_id=&type=
func (s *LogService) ListLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.ParseInt(q.Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(q.Get("page_size"), 10, 32)
	level := q.Get("type")   // API "type" maps to biz "level"
	source := ""              // not exposed in API spec
	keyword := ""             // not exposed in API spec
	_ = q.Get("user_id")     // user_id available for future filtering

	entries, total, err := s.uc.ListLogs(r.Context(), int32(page), int32(pageSize), level, source, keyword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		items = append(items, logEntryToMap(e))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// IngestLog handles POST /v1/logs
func (s *LogService) IngestLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Level     string `json:"level"`
		Message   string `json:"message"`
		Source    string `json:"source"`
		RequestID string `json:"request_id"`
		UserID    int64  `json:"user_id"`
	}
	if err := sonic.ConfigStd.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry := &biz.LogEntry{
		Level:     body.Level,
		Message:   body.Message,
		Source:    body.Source,
		RequestID: body.RequestID,
		UserID:    body.UserID,
	}

	if err := s.uc.IngestLog(r.Context(), entry); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, logEntryToMap(entry))
}

func logEntryToMap(e *biz.LogEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":         e.ID,
		"level":      e.Level,
		"message":    e.Message,
		"source":     e.Source,
		"request_id": e.RequestID,
		"user_id":    e.UserID,
		"created_at": e.CreatedAt,
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
