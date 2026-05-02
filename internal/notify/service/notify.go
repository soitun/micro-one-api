package service

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	"micro-one-api/internal/notify/biz"
)

// NotifyService is the transport layer entry for notify-worker.
type NotifyService struct {
	uc *biz.NotifyUsecase
}

func NewNotifyService(uc *biz.NotifyUsecase) *NotifyService {
	return &NotifyService{uc: uc}
}

// CreateNotification handles POST /v1/notifications
func (s *NotifyService) CreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Type      string `json:"type"`
		Recipient string `json:"recipient"`
		Subject   string `json:"subject"`
		Content   string `json:"content"`
	}
	if err := sonic.ConfigStd.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	n, err := s.uc.CreateNotification(r.Context(), body.Type, body.Recipient, body.Subject, body.Content)
	if err != nil {
		if err == biz.ErrInvalidNotification {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, notificationToMap(n))
}

// GetNotification handles GET /v1/notifications/{id}
func (s *NotifyService) GetNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/v1/notifications/")
	idStr = strings.TrimRight(idStr, "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	n, err := s.uc.GetNotification(r.Context(), id)
	if err != nil {
		if err == biz.ErrNotificationNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, notificationToMap(n))
}

// ListNotifications handles GET /v1/notifications?page=&page_size=&type=&status=
func (s *NotifyService) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.ParseInt(q.Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(q.Get("page_size"), 10, 32)
	notifyType := q.Get("type")
	status := q.Get("status")

	notifications, total, err := s.uc.ListNotifications(r.Context(), int32(page), int32(pageSize), notifyType, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, notificationToMap(n))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

func notificationToMap(n *biz.Notification) map[string]interface{} {
	return map[string]interface{}{
		"id":          n.ID,
		"type":        n.Type,
		"recipient":   n.Recipient,
		"subject":     n.Subject,
		"content":     n.Content,
		"status":      n.Status,
		"retry_count": n.RetryCount,
		"created_at":  n.CreatedAt,
		"sent_at":     n.SentAt,
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
