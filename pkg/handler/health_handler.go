package handler

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("remote_addr", r.RemoteAddr).
		Str("path", r.URL.Path).
		Logger()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))

	logger.Debug().Msg("Health check processed")
}
