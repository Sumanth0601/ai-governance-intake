package proposals

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	if req.Title == "" || req.Description == "" || req.DataSources == "" || req.IntendedUse == "" {
		writeError(w, http.StatusBadRequest, "title, description, data_sources, and intended_use are required", "MISSING_FIELDS")
		return
	}

	resp, err := h.svc.Submit(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrLLMParseFailure) {
			writeError(w, http.StatusUnprocessableEntity, "LLM returned unparseable response", "LLM_PARSE_ERROR")
			return
		}
		log.Printf("ERROR submit: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR")
		return
	}

	code := http.StatusCreated
	if resp.Status == "duplicate" {
		code = http.StatusOK
	}
	writeJSON(w, code, resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	proposals, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR")
		return
	}
	if proposals == nil {
		proposals = []Proposal{}
	}
	writeJSON(w, http.StatusOK, proposals)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	proposal, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR")
		return
	}
	if proposal == nil {
		writeError(w, http.StatusNotFound, "proposal not found", "NOT_FOUND")
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg, errCode string) {
	writeJSON(w, code, map[string]string{"error": msg, "code": errCode})
}
