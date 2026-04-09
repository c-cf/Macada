package handler

import (
	"net/http"
	"strconv"

	"github.com/c-cf/macada/internal/infra/postgres"
)

// AnalyticsHandler serves analytics API endpoints.
type AnalyticsHandler struct {
	repo *postgres.AnalyticsRepo
}

func NewAnalyticsHandler(repo *postgres.AnalyticsRepo) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo}
}

type usageResponse struct {
	Data    []postgres.UsageDayRow `json:"data"`
	Summary usageSummary           `json:"summary"`
}

type usageSummary struct {
	TotalInput    int64 `json:"total_input"`
	TotalOutput   int64 `json:"total_output"`
	TotalRequests int64 `json:"total_requests"`
}

// Usage handles GET /v1/analytics/usage?from=&to=&model=
func (h *AnalyticsHandler) Usage(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	model := r.URL.Query().Get("model")

	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "from and to query parameters are required (YYYY-MM-DD)")
		return
	}

	wsID := workspaceIDFromCtx(r)
	rows, err := h.repo.QueryUsage(r.Context(), wsID, from, to, model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query usage data")
		return
	}
	if rows == nil {
		rows = []postgres.UsageDayRow{}
	}

	var summary usageSummary
	for _, row := range rows {
		summary.TotalInput += row.InputTokens
		summary.TotalOutput += row.OutputTokens
		summary.TotalRequests += row.RequestCount
	}

	writeJSON(w, http.StatusOK, usageResponse{
		Data:    rows,
		Summary: summary,
	})
}

type logsResponse struct {
	Data     []postgres.LogRow `json:"data"`
	NextPage *string           `json:"next_page"`
}

// Logs handles GET /v1/analytics/logs?limit=&page=
func (h *AnalyticsHandler) Logs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	var pageCursor *string
	if v := r.URL.Query().Get("page"); v != "" {
		pageCursor = &v
	}

	wsID := workspaceIDFromCtx(r)
	logs, nextPage, err := h.repo.QueryLogs(r.Context(), wsID, limit, pageCursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query logs")
		return
	}
	if logs == nil {
		logs = []postgres.LogRow{}
	}

	writeJSON(w, http.StatusOK, logsResponse{
		Data:     logs,
		NextPage: nextPage,
	})
}
