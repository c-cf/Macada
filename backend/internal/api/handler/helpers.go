package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cchu-code/managed-agents/internal/api/middleware"
	"github.com/cchu-code/managed-agents/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"type":    "error",
		"error":   map[string]string{"type": http.StatusText(status), "message": message},
	})
}

func parseListParams(r *http.Request) domain.ListParams {
	params := domain.ListParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Limit = &n
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		params.Page = &v
	}
	if r.URL.Query().Get("include_archived") == "true" {
		params.IncludeArchived = true
	}
	return params
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func workspaceIDFromCtx(r *http.Request) string {
	return middleware.WorkspaceIDFromCtx(r.Context())
}
