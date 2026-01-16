package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/store"
)

// CostsHandler handles cost-related requests
type CostsHandler struct {
	calculator *cogs.Calculator
	provider   store.DataProvider
}

// NewCostsHandler creates a new costs handler
func NewCostsHandler(calculator *cogs.Calculator, provider store.DataProvider) *CostsHandler {
	return &CostsHandler{
		calculator: calculator,
		provider:   provider,
	}
}

// GetCosts handles GET /api/v1/costs
func (h *CostsHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query params
	clusters := parseCSV(r.URL.Query().Get("cluster"))
	namespaces := parseCSV(r.URL.Query().Get("namespace"))
	algorithmID := cogs.AlgorithmID(r.URL.Query().Get("algorithm"))

	// Fetch data
	nodes, err := h.provider.GetNodes(ctx, clusters)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch nodes", err.Error())
		return
	}

	pods, err := h.provider.GetPods(ctx, clusters, namespaces)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch pods", err.Error())
		return
	}

	// Recalculate Fargate costs based on actual pod requests
	nodes = h.provider.RecalculateFargateCosts(ctx, nodes, pods)

	// Calculate costs
	result, err := h.calculator.Calculate(ctx, algorithmID, nodes, pods, cogs.AlgorithmParams{})
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Failed to calculate costs", err.Error())
		return
	}

	// Build response
	resp := h.buildCostResponse(result, nodes, pods, clusters, namespaces)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetClusterCosts handles GET /api/v1/costs/clusters
func (h *CostsHandler) GetClusterCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clusters := parseCSV(r.URL.Query().Get("cluster"))
	algorithmID := cogs.AlgorithmID(r.URL.Query().Get("algorithm"))

	nodes, err := h.provider.GetNodes(ctx, clusters)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch nodes", err.Error())
		return
	}

	pods, err := h.provider.GetPods(ctx, clusters, nil)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch pods", err.Error())
		return
	}

	// Recalculate Fargate costs based on actual pod requests
	nodes = h.provider.RecalculateFargateCosts(ctx, nodes, pods)

	result, err := h.calculator.Calculate(ctx, algorithmID, nodes, pods, cogs.AlgorithmParams{})
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Failed to calculate costs", err.Error())
		return
	}

	// Build cluster-focused response
	clusterDTOs := make([]ClusterCostDTO, 0, len(result.ClusterCosts))
	for _, cc := range result.ClusterCosts {
		clusterDTOs = append(clusterDTOs, ClusterCostDTO{
			Name:           cc.ClusterName,
			TotalCost:      float64(cc.TotalCost),
			NodeCount:      cc.NodeCount,
			PodCount:       cc.PodCount,
			NamespaceCount: len(cc.NamespaceCosts),
		})
	}

	resp := CostResponse{
		Timestamp: result.Timestamp,
		Algorithm: AlgorithmMeta{ID: string(result.AlgorithmID)},
		TotalCost: float64(result.TotalCost),
		Currency:  "USD",
		Clusters:  clusterDTOs,
		Filters:   AppliedFilters{Clusters: clusters},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetNamespaceCosts handles GET /api/v1/costs/namespaces
func (h *CostsHandler) GetNamespaceCosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clusters := parseCSV(r.URL.Query().Get("cluster"))
	namespaces := parseCSV(r.URL.Query().Get("namespace"))
	algorithmID := cogs.AlgorithmID(r.URL.Query().Get("algorithm"))

	nodes, err := h.provider.GetNodes(ctx, clusters)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch nodes", err.Error())
		return
	}

	pods, err := h.provider.GetPods(ctx, clusters, namespaces)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch pods", err.Error())
		return
	}

	// Recalculate Fargate costs based on actual pod requests
	nodes = h.provider.RecalculateFargateCosts(ctx, nodes, pods)

	result, err := h.calculator.Calculate(ctx, algorithmID, nodes, pods, cogs.AlgorithmParams{})
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Failed to calculate costs", err.Error())
		return
	}

	// Build namespace-focused response
	var nsDTOs []NamespaceCostDTO
	for clusterName, cc := range result.ClusterCosts {
		for nsName, nc := range cc.NamespaceCosts {
			podCount := 0
			for _, wc := range nc.WorkloadCosts {
				podCount += len(wc.Pods)
			}
			nsDTOs = append(nsDTOs, NamespaceCostDTO{
				Cluster:   clusterName,
				Namespace: nsName,
				TotalCost: float64(nc.TotalCost),
				PodCount:  podCount,
			})
		}
	}

	resp := CostResponse{
		Timestamp:  result.Timestamp,
		Algorithm:  AlgorithmMeta{ID: string(result.AlgorithmID)},
		TotalCost:  float64(result.TotalCost),
		Currency:   "USD",
		Namespaces: nsDTOs,
		Filters:    AppliedFilters{Clusters: clusters, Namespaces: namespaces},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// buildCostResponse creates the full cost response from calculation result
func (h *CostsHandler) buildCostResponse(result *cogs.CalculationResult, nodes []cogs.NodeInfo, pods []cogs.PodInfo, clusters, namespaces []string) CostResponse {
	var clusterDTOs []ClusterCostDTO
	var nsDTOs []NamespaceCostDTO
	var wlDTOs []WorkloadCostDTO

	// Count pods per node
	podCountByNode := make(map[string]int)
	for _, pod := range pods {
		key := pod.ClusterName + "/" + pod.NodeName
		podCountByNode[key]++
	}

	// Build node DTOs
	var nodeDTOs []NodeCostDTO
	for _, node := range nodes {
		key := node.ClusterName + "/" + node.Name
		instanceType := node.InstanceType
		if node.IsFargate {
			instanceType = "Fargate"
		}
		nodeDTOs = append(nodeDTOs, NodeCostDTO{
			Cluster:        node.ClusterName,
			Name:           node.Name,
			InstanceType:   instanceType,
			Region:         node.Region,
			HourlyCost:     float64(node.HourlyCost),
			PodCount:       podCountByNode[key],
			CPUCapacity:    formatCPU(node.CPUCapacity),
			MemCapacity:    formatMemory(node.MemCapacity),
			CPUCapacityRaw: node.CPUCapacity,
			MemCapacityRaw: node.MemCapacity,
		})
	}

	for _, cc := range result.ClusterCosts {
		clusterDTOs = append(clusterDTOs, ClusterCostDTO{
			Name:           cc.ClusterName,
			TotalCost:      float64(cc.TotalCost),
			NodeCount:      cc.NodeCount,
			PodCount:       cc.PodCount,
			NamespaceCount: len(cc.NamespaceCosts),
		})

		for nsName, nc := range cc.NamespaceCosts {
			podCount := 0
			for _, wc := range nc.WorkloadCosts {
				podCount += len(wc.Pods)
				// Skip internal workload types
				if wc.Kind == "AutoscalingListener" {
					continue
				}
				wlDTOs = append(wlDTOs, WorkloadCostDTO{
					Cluster:   cc.ClusterName,
					Namespace: nsName,
					Name:      wc.Name,
					Kind:      wc.Kind,
					TotalCost: float64(wc.TotalCost),
					PodCount:  len(wc.Pods),
				})
			}
			nsDTOs = append(nsDTOs, NamespaceCostDTO{
				Cluster:   cc.ClusterName,
				Namespace: nsName,
				TotalCost: float64(nc.TotalCost),
				PodCount:  podCount,
			})
		}
	}

	return CostResponse{
		Timestamp:  time.Now(),
		Algorithm:  AlgorithmMeta{ID: string(result.AlgorithmID)},
		TotalCost:  float64(result.TotalCost),
		Currency:   "USD",
		Clusters:   clusterDTOs,
		Namespaces: nsDTOs,
		Workloads:  wlDTOs,
		Nodes:      nodeDTOs,
		Filters: AppliedFilters{
			Clusters:   clusters,
			Namespaces: namespaces,
		},
	}
}

// formatCPU formats CPU millicores to a human-readable string
func formatCPU(millicores int64) string {
	if millicores >= 1000 {
		return fmt.Sprintf("%d", millicores/1000)
	}
	return fmt.Sprintf("%dm", millicores)
}

// formatMemory formats bytes to a human-readable string
func formatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%dGi", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%dMi", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%dKi", bytes/KB)
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

func (h *CostsHandler) sendError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
