package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cluster"
)

// ClustersHandler handles cluster management requests
type ClustersHandler struct {
	manager *cluster.Manager
}

// NewClustersHandler creates a new clusters handler
func NewClustersHandler(manager *cluster.Manager) *ClustersHandler {
	return &ClustersHandler{
		manager: manager,
	}
}

// KubeconfigResponse is the response for kubeconfig operations
type KubeconfigResponse struct {
	Filename string                `json:"filename"`
	Contexts []cluster.ContextInfo `json:"contexts"`
}

// AddClusterRequest is the request body for adding a cluster
type AddClusterRequest struct {
	Name    string `json:"name"`
	Context string `json:"context"`
}

// UploadKubeconfig handles POST /api/v1/kubeconfig (multipart file upload)
// Supports multiple files - they are merged together
func (h *ClustersHandler) UploadKubeconfig(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Failed to parse form", err.Error())
		return
	}

	// Get all kubeconfig files from the form
	files := r.MultipartForm.File["kubeconfig"]
	if len(files) == 0 {
		sendJSONError(w, http.StatusBadRequest, "No kubeconfig file provided", "")
		return
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, "Failed to open file", err.Error())
			return
		}

		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, "Failed to read file", err.Error())
			return
		}

		// Merge each kubeconfig (AddKubeconfigContent handles first file case)
		if err := h.manager.AddKubeconfigContent(content, fileHeader.Filename); err != nil {
			sendJSONError(w, http.StatusBadRequest, "Invalid kubeconfig file: "+fileHeader.Filename, err.Error())
			return
		}
	}

	// Get available contexts
	contexts, err := h.manager.ListContexts()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "Failed to list contexts", err.Error())
		return
	}

	resp := KubeconfigResponse{
		Filename: h.manager.GetKubeconfigFilename(),
		Contexts: contexts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetKubeconfig handles GET /api/v1/kubeconfig
func (h *ClustersHandler) GetKubeconfig(w http.ResponseWriter, r *http.Request) {
	filename := h.manager.GetKubeconfigFilename()

	var contexts []cluster.ContextInfo
	var err error
	if h.manager.HasKubeconfig() {
		contexts, err = h.manager.ListContexts()
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, "Failed to list contexts", err.Error())
			return
		}
	}

	resp := KubeconfigResponse{
		Filename: filename,
		Contexts: contexts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ListClusters handles GET /api/v1/clusters
func (h *ClustersHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
	clusters := h.manager.ListClusters()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"clusters": clusters,
	})
}

// AddCluster handles POST /api/v1/clusters
func (h *ClustersHandler) AddCluster(w http.ResponseWriter, r *http.Request) {
	var req AddClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Name == "" {
		req.Name = req.Context
	}

	if err := h.manager.AddCluster(r.Context(), req.Name, req.Context); err != nil {
		sendJSONError(w, http.StatusInternalServerError, "Failed to add cluster", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "connected",
		"name":    req.Name,
		"context": req.Context,
	})
}

// RemoveCluster handles DELETE /api/v1/clusters/{name}
func (h *ClustersHandler) RemoveCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.manager.RemoveCluster(name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "removed",
		"name":   name,
	})
}

// RemoveContext handles DELETE /api/v1/contexts/{name}
func (h *ClustersHandler) RemoveContext(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.manager.RemoveContext(name)

	// Return updated contexts list
	contexts, err := h.manager.ListContexts()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "Failed to list contexts", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(KubeconfigResponse{
		Filename: h.manager.GetKubeconfigFilename(),
		Contexts: contexts,
	})
}

func sendJSONError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}
