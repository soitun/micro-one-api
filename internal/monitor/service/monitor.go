package service

import (
	"net/http"
	"strconv"

	"github.com/bytedance/sonic"

	"micro-one-api/internal/monitor/biz"
)

// MonitorService is the transport layer entry for monitor-worker.
type MonitorService struct {
	uc *biz.MonitorUsecase
}

func NewMonitorService(uc *biz.MonitorUsecase) *MonitorService {
	return &MonitorService{uc: uc}
}

// RecordHealthCheck handles POST /v1/health-checks
func (s *MonitorService) RecordHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		ServiceName  string `json:"service_name"`
		Status       string `json:"status"`
		ResponseTime int64  `json:"response_time"`
	}
	if err := sonic.ConfigStd.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.ServiceName == "" {
		writeError(w, http.StatusBadRequest, "service_name is required")
		return
	}

	if err := s.uc.RecordHealthCheck(r.Context(), body.ServiceName, body.Status, body.ResponseTime); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// ListHealthChecks handles GET /v1/health-checks?service_name=&page=&page_size=
func (s *MonitorService) ListHealthChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	serviceName := q.Get("service_name")
	page, _ := strconv.ParseInt(q.Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(q.Get("page_size"), 10, 32)

	checks, total, err := s.uc.ListHealthChecks(r.Context(), serviceName, int32(page), int32(pageSize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(checks))
	for _, c := range checks {
		items = append(items, map[string]interface{}{
			"id":            c.ID,
			"service_name":  c.ServiceName,
			"status":        c.Status,
			"response_time": c.ResponseTime,
			"checked_at":    c.CheckedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// ListAlertRules handles GET /v1/alert-rules?page=&page_size=
func (s *MonitorService) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.ParseInt(q.Get("page"), 10, 32)
	pageSize, _ := strconv.ParseInt(q.Get("page_size"), 10, 32)

	rules, total, err := s.uc.ListAlertRules(r.Context(), int32(page), int32(pageSize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		items = append(items, alertRuleToMap(rule))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// CreateAlertRule handles POST /v1/alert-rules
func (s *MonitorService) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Name        string  `json:"name"`
		ServiceName string  `json:"service_name"`
		Metric      string  `json:"metric"`
		Threshold   float64 `json:"threshold"`
		Operator    string  `json:"operator"`
		Duration    int     `json:"duration"`
		Enabled     bool    `json:"enabled"`
	}
	if err := sonic.ConfigStd.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule := &biz.AlertRule{
		Name:        body.Name,
		ServiceName: body.ServiceName,
		Metric:      body.Metric,
		Threshold:   body.Threshold,
		Operator:    body.Operator,
		Duration:    body.Duration,
		Enabled:     body.Enabled,
	}

	if err := s.uc.CreateAlertRule(r.Context(), rule); err != nil {
		if err == biz.ErrInvalidAlertRule {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, alertRuleToMap(rule))
}

func alertRuleToMap(rule *biz.AlertRule) map[string]interface{} {
	return map[string]interface{}{
		"id":           rule.ID,
		"name":         rule.Name,
		"service_name": rule.ServiceName,
		"metric":       rule.Metric,
		"threshold":    rule.Threshold,
		"operator":     rule.Operator,
		"duration":     rule.Duration,
		"enabled":      rule.Enabled,
		"created_at":   rule.CreatedAt,
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
