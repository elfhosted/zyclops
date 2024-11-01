package handler

import (
	"encoding/json"
	"net/http"

	"github.com/elfhosted/zyclops/pkg/domain"
	"github.com/elfhosted/zyclops/pkg/service"
	"github.com/rs/zerolog/log"
)

type SearchHandler struct {
	searchService service.SearchService
}

func NewSearchHandler(searchService service.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("remote_addr", r.RemoteAddr).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Logger()

	if r.Method != http.MethodPost {
		logger.Warn().Msg("Invalid request method")
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.Info().Str("query", req.QueryText).Msg("Processing search request")

	results, err := h.searchService.Search(req.QueryText)
	if err != nil {
		logger.Error().Err(err).Msg("Search failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("results", len(results)).Msg("Search completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
