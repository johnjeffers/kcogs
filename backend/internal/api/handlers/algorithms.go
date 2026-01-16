package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// AlgorithmsHandler handles algorithm-related requests
type AlgorithmsHandler struct {
	calculator *cogs.Calculator
}

// NewAlgorithmsHandler creates a new algorithms handler
func NewAlgorithmsHandler(calculator *cogs.Calculator) *AlgorithmsHandler {
	return &AlgorithmsHandler{
		calculator: calculator,
	}
}

// List handles GET /api/v1/algorithms
func (h *AlgorithmsHandler) List(w http.ResponseWriter, r *http.Request) {
	infos := h.calculator.ListAlgorithms()

	dtos := make([]AlgorithmDTO, len(infos))
	var defaultID string
	for i, info := range infos {
		dtos[i] = ToAlgorithmDTO(info)
		if info.IsDefault {
			defaultID = string(info.ID)
		}
	}

	resp := AlgorithmListResponse{
		Algorithms: dtos,
		Default:    defaultID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Get handles GET /api/v1/algorithms/{id}
func (h *AlgorithmsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	alg, err := h.calculator.GetAlgorithm(cogs.AlgorithmID(id))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Algorithm not found",
			Code:  http.StatusNotFound,
		})
		return
	}

	dto := AlgorithmDTO{
		ID:          string(alg.ID()),
		Name:        alg.Name(),
		Description: alg.Description(),
		IsDefault:   false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto)
}
