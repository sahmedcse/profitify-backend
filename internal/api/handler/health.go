package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Health handles GET /health requests.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("encoding health response", "error", err)
	}
}
