package handler

import (
	"encoding/json"
	"net/http"

	"github.com/elfhosted/zyclops/pkg/domain"
	"github.com/elfhosted/zyclops/pkg/service"
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
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	results, err := h.searchService.Search(req.QueryText)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
